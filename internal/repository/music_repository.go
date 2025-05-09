package repository

import (
	"strings" // strings.ToLower vb. için

	"github.com/google/uuid"
	"github.com/morgarakt/aurify/internal/models" // Projenizin model yolu
	"gorm.io/gorm"
)

// MusicQueryParams veritabanı sorguları için parametreleri taşır
type MusicQueryParams struct {
	SearchQuery     string
	MusicTypeFilter string
	SortBy          string
	Page            int
	PerPage         int
}

// MusicRepository arayüzü güncellendi
type MusicRepository interface {
	Create(music *models.Music) error
	Delete(music *models.Music) error
	GetByID(id any, music *models.Music) error // Bu generic kalabilir
	GetByIDWithRelations(id uuid.UUID) (*models.Music, error)
	Update(music *models.Music) error

	// Filtreleme, sıralama ve sayfalama için yeni metodlar
	QueryUserMusic(userID uuid.UUID, params MusicQueryParams) ([]models.Music, int64, error)
	QueryPublicMusic(params MusicQueryParams) ([]models.Music, int64, error)
}

type musicRepo struct {
	*GenericRepository[models.Music]
	db *gorm.DB
}

func NewMusicRepository(db *gorm.DB) MusicRepository {
	return &musicRepo{
		GenericRepository: NewGenericRepository[models.Music](db),
		db:                db,
	}
}

// GetByIDWithRelations ve Update metodları (varsa) aynı kalır.
// Örnek:
func (r *musicRepo) GetByIDWithRelations(id uuid.UUID) (*models.Music, error) {
	var music models.Music
	err := r.db.Preload("User").Preload("MusicType").Preload("ModelType").First(&music, id).Error
	if err != nil {
		return nil, err
	}
	return &music, nil
}

func (r *musicRepo) Update(music *models.Music) error {
	return r.db.Save(music).Error
}

// QueryUserMusic kullanıcıya ait müzikleri filtreler, sıralar ve sayfalar.
// Döndürülen değerler: bulunan müzik listesi, toplam filtrelenmiş öğe sayısı, hata.
func (r *musicRepo) QueryUserMusic(userID uuid.UUID, params MusicQueryParams) ([]models.Music, int64, error) {
	var musicList []models.Music
	var totalItems int64

	query := r.db.Model(&models.Music{}).
		Joins("User"). // User.Username için (eğer User tablosunda username varsa)
		Joins("MusicType").
		Joins("ModelType").
		Where("musics.user_id = ?", userID) // `musics.` ile tablo adını belirtmek belirsizliği önler

	// Filtreleme
	if params.SearchQuery != "" {
		lowerSearchQuery := "%" + strings.ToLower(params.SearchQuery) + "%"
		query = query.Where(
			r.db.Where("LOWER(musics.title) LIKE ?", lowerSearchQuery).
				Or("LOWER(\"User\".username) LIKE ?", lowerSearchQuery). // Çift tırnak PostgreSQL'de büyük/küçük harf duyarlılığı için
				Or("LOWER(\"MusicType\".name) LIKE ?", lowerSearchQuery),
		)
	}
	if params.MusicTypeFilter != "" {
		query = query.Where("\"MusicType\".name = ?", params.MusicTypeFilter)
	}

	// Toplam öğe sayısını almak için filtreli sorgu (sayfalama öncesi)
	// Count sorgusunu ana sorgudan ayırmak, özellikle JOIN'ler olduğunda daha güvenilir olabilir.
	countQuery := r.db.Model(&models.Music{}).
		Joins("JOIN users AS \"User\" ON \"User\".id = musics.user_id").
		Joins("JOIN music_types AS \"MusicType\" ON \"MusicType\".id = musics.music_type_id").
		Where("musics.user_id = ?", userID)
	if params.SearchQuery != "" {
		lowerSearchQuery := "%" + strings.ToLower(params.SearchQuery) + "%"
		countQuery = countQuery.Where(
			r.db.Where("LOWER(musics.title) LIKE ?", lowerSearchQuery).
				Or("LOWER(\"User\".username) LIKE ?", lowerSearchQuery).
				Or("LOWER(\"MusicType\".name) LIKE ?", lowerSearchQuery),
		)
	}
	if params.MusicTypeFilter != "" {
		countQuery = countQuery.Where("\"MusicType\".name = ?", params.MusicTypeFilter)
	}
	if err := countQuery.Count(&totalItems).Error; err != nil {
		return nil, 0, err
	}

	// Sıralama
	orderClause := "musics.created_at desc" // Varsayılan (en yeni)
	switch params.SortBy {
	case "added_desc":
		orderClause = "musics.created_at desc"
	case "title_asc":
		orderClause = "musics.title asc"
	case "title_desc":
		orderClause = "musics.title desc"
		// Gelecekte eklenebilecek diğer sıralama seçenekleri
	}
	query = query.Order(orderClause)

	// Sayfalama
	offset := (params.Page - 1) * params.PerPage
	query = query.Offset(offset).Limit(params.PerPage)

	// İlişkileri Preload ile yükle (JOIN'ler zaten yapıldıysa bazıları gerekmeyebilir, ama GORM yönetir)
	if err := query.Find(&musicList).Error; err != nil {
		return nil, 0, err
	}

	return musicList, totalItems, nil
}

// QueryPublicMusic herkese açık müzikleri filtreler, sıralar ve sayfalar.
func (r *musicRepo) QueryPublicMusic(params MusicQueryParams) ([]models.Music, int64, error) {
	var musicList []models.Music
	var totalItems int64

	query := r.db.Model(&models.Music{}).
		Joins("User").
		Joins("MusicType").
		Joins("ModelType").
		Where("musics.is_public = ?", true)

	// Filtreleme
	if params.SearchQuery != "" {
		lowerSearchQuery := "%" + strings.ToLower(params.SearchQuery) + "%"
		query = query.Where(
			r.db.Where("LOWER(musics.title) LIKE ?", lowerSearchQuery).
				Or("LOWER(\"User\".username) LIKE ?", lowerSearchQuery).
				Or("LOWER(\"MusicType\".name) LIKE ?", lowerSearchQuery),
		)
	}
	if params.MusicTypeFilter != "" {
		query = query.Where("\"MusicType\".name = ?", params.MusicTypeFilter)
	}

	countQuery := r.db.Model(&models.Music{}).
		Joins("JOIN users AS \"User\" ON \"User\".id = musics.user_id"). // UserID nullable olduğu için LEFT JOIN daha uygun olabilir
		Joins("JOIN music_types AS \"MusicType\" ON \"MusicType\".id = musics.music_type_id").
		Where("musics.is_public = ?", true)
	if params.SearchQuery != "" {
		lowerSearchQuery := "%" + strings.ToLower(params.SearchQuery) + "%"
		countQuery = countQuery.Where(
			r.db.Where("LOWER(musics.title) LIKE ?", lowerSearchQuery).
				Or("LOWER(\"User\".username) LIKE ?", lowerSearchQuery).
				Or("LOWER(\"MusicType\".name) LIKE ?", lowerSearchQuery),
		)
	}
	if params.MusicTypeFilter != "" {
		countQuery = countQuery.Where("\"MusicType\".name = ?", params.MusicTypeFilter)
	}
	if err := countQuery.Count(&totalItems).Error; err != nil {
		return nil, 0, err
	}

	// Sıralama
	orderClause := "musics.created_at desc"
	switch params.SortBy {
	case "added_desc":
		orderClause = "musics.created_at desc"
	case "title_asc":
		orderClause = "musics.title asc"
	case "title_desc":
		orderClause = "musics.title desc"
	}
	query = query.Order(orderClause)

	// Sayfalama
	offset := (params.Page - 1) * params.PerPage
	query = query.Offset(offset).Limit(params.PerPage)

	if err := query.Find(&musicList).Error; err != nil {
		return nil, 0, err
	}
	return musicList, totalItems, nil
}
