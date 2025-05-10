package repository

import (
	"fmt"
	"log"
	"strings"

	"github.com/google/uuid"
	"github.com/morgarakt/aurify/internal/models" // Projenizin model yolu
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// MusicQueryParams struct (önceki yanıttaki gibi)
type MusicQueryParams struct {
	SearchQuery     string
	MusicTypeFilter string
	SortBy          string
	Page            int
	PerPage         int
}

// MusicRepository arayüzü (önceki yanıttaki gibi)
type MusicRepository interface {
	Create(music *models.Music) error
	Delete(music *models.Music) error
	GetByID(id any, music *models.Music) error
	GetByIDWithRelations(id uuid.UUID) (*models.Music, error)
	Update(music *models.Music) error
	QueryUserMusic(userID uuid.UUID, params MusicQueryParams) ([]models.Music, int64, error)
	QueryPublicMusic(params MusicQueryParams) ([]models.Music, int64, error)
	UpdateLikesCount(musicID uuid.UUID, change int) error
	UpdateVisibility(musicID uuid.UUID, isPublic bool) error
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

func (r *musicRepo) Create(music *models.Music) error {
	return r.db.Create(music).Error
}

func (r *musicRepo) Delete(music *models.Music) error {
	return r.db.Delete(music).Error
}

func (r *musicRepo) GetByIDWithRelations(id uuid.UUID) (*models.Music, error) {
	var music models.Music
	err := r.db.Preload("User").Preload("MusicType").Preload("ModelType").First(&music, "musics.id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &music, nil
}

func (r *musicRepo) Update(music *models.Music) error {
	return r.db.Save(music).Error
}

func (r *musicRepo) UpdateLikesCount(musicID uuid.UUID, change int) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var music models.Music
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&music, musicID).Error; err != nil {
			return err
		}
		newLikesCount := music.LikesCount + change
		if newLikesCount < 0 {
			newLikesCount = 0
		}
		return tx.Model(&models.Music{}).Where("id = ?", musicID).Update("likes_count", newLikesCount).Error
	})
}

func (r *musicRepo) UpdateVisibility(musicID uuid.UUID, isPublic bool) error {
	return r.db.Model(&models.Music{}).Where("id = ?", musicID).Update("is_public", isPublic).Error
}

// QueryUserMusic kullanıcıya ait müzikleri filtreler, sıralar ve sayfalar.
func (r *musicRepo) QueryUserMusic(userID uuid.UUID, params MusicQueryParams) ([]models.Music, int64, error) {
	var musicList []models.Music
	var totalItems int64

	// Temel sorgu "musics" tablosu üzerinden başlar ve kullanıcıya göre filtrelenir.
	baseQuery := r.db.Table("musics").Where("musics.user_id = ?", userID)

	// JOIN'leri ekle (hem count hem de data sorgusu için gerekli olacaklar)
	// Alias (takma ad) kullanarak tabloları ayırt et.
	queryWithJoins := baseQuery.
		Joins("LEFT JOIN users AS u ON u.id = musics.user_id"). // UserID nullable değilse INNER JOIN olabilir
		Joins("LEFT JOIN music_types AS mt ON mt.id = musics.music_type_id").
		Joins("LEFT JOIN model_types AS mdlt ON mdlt.id = musics.model_type_id")

	// Filtreleme koşullarını oluştur
	filterSession := queryWithJoins
	if params.SearchQuery != "" {
		sq := "%" + strings.ToLower(params.SearchQuery) + "%"
		filterSession = filterSession.Where(
			r.db.Where("LOWER(musics.title) LIKE ?", sq).
				Or("LOWER(u.username) LIKE ?", sq).
				Or("LOWER(mt.name) LIKE ?", sq),
		)
	}
	if params.MusicTypeFilter != "" {
		filterSession = filterSession.Where("mt.name = ?", params.MusicTypeFilter)
	}

	// Toplam öğe sayısını al
	// SELECT count(musics.id) FROM musics ... (joinler ve where koşulları ile)
	// Model(&models.Music{}) yerine Table("musics") kullandığımız için count için de bunu kullanalım
	// veya Model'i burada kullanalım ama join'leri dikkatli yönetelim.
	// En temizi, filterSession üzerinden count almak.
	if err := filterSession.Select("count(musics.id)").Count(&totalItems).Error; err != nil {
		log.Printf("Error in countQuery for UserMusic (User: %s): %v", userID, err)
		// GORM'un ürettiği SQL'i görmek için:
		// log.Printf("Failed SQL for count: %s", r.db.ToSQL(func(tx *gorm.DB) *gorm.DB { return filterSession.Select("count(musics.id)").Count(&totalItems) }))
		return nil, 0, fmt.Errorf("counting user music failed: %w", err)
	}

	// Asıl veri çekme sorgusu için sıralama ve sayfalama ekle
	dataSession := filterSession // Filtrelenmiş sorguyu kullan
	orderClause := "musics.created_at desc"
	switch params.SortBy {
	case "added_desc":
		orderClause = "musics.created_at desc"
	case "title_asc":
		orderClause = "musics.title asc"
	case "title_desc":
		orderClause = "musics.title desc"
	}
	dataSession = dataSession.Order(orderClause)

	offset := (params.Page - 1) * params.PerPage
	// Veriyi çekerken Preload ile ilişkili modelleri de yükle.
	// GORM, ana sorgudaki JOIN'leri ve Preload'ları akıllıca birleştirmeye çalışır.
	// Veya Preload için ayrı sorgular yapar.
	// Burada musics.* seçerek ve sonra Preload yaparak daha temiz olabilir.
	err := dataSession.Select("musics.*").
		Offset(offset).Limit(params.PerPage).
		Preload("User").Preload("MusicType").Preload("ModelType").
		Find(&musicList).Error

	if err != nil {
		log.Printf("Error in dataQuery for UserMusic (User: %s): %v", userID, err)
		// log.Printf("Failed SQL for data: %s", r.db.ToSQL(func(tx *gorm.DB) *gorm.DB { return dataSession.Select("musics.*").Offset(offset).Limit(params.PerPage).Preload("User").Preload("MusicType").Preload("ModelType").Find(&musicList) }))
		return nil, 0, fmt.Errorf("finding user music failed: %w", err)
	}

	return musicList, totalItems, nil
}

// QueryPublicMusic herkese açık müzikleri filtreler, sıralar ve sayfalar.
func (r *musicRepo) QueryPublicMusic(params MusicQueryParams) ([]models.Music, int64, error) {
	var musicList []models.Music
	var totalItems int64

	baseQuery := r.db.Table("musics").Where("musics.is_public = ?", true)

	queryWithJoins := baseQuery.
		Joins("LEFT JOIN users AS u ON u.id = musics.user_id"). // UserID nullable olduğu için LEFT JOIN
		Joins("LEFT JOIN music_types AS mt ON mt.id = musics.music_type_id").
		Joins("LEFT JOIN model_types AS mdlt ON mdlt.id = musics.model_type_id")

	filterSession := queryWithJoins
	if params.SearchQuery != "" {
		sq := "%" + strings.ToLower(params.SearchQuery) + "%"
		filterSession = filterSession.Where(
			r.db.Where("LOWER(musics.title) LIKE ?", sq).
				Or("LOWER(u.username) LIKE ?", sq). // u.username nil olabilir, sorgu bunu handle etmeli
				Or("LOWER(mt.name) LIKE ?", sq),
		)
	}
	if params.MusicTypeFilter != "" {
		filterSession = filterSession.Where("mt.name = ?", params.MusicTypeFilter)
	}

	if err := filterSession.Select("count(musics.id)").Count(&totalItems).Error; err != nil {
		log.Printf("Error in countQuery for PublicMusic: %v", err)
		return nil, 0, fmt.Errorf("counting public music failed: %w", err)
	}

	dataSession := filterSession
	orderClause := "musics.created_at desc"
	switch params.SortBy {
	case "added_desc":
		orderClause = "musics.created_at desc"
	case "title_asc":
		orderClause = "musics.title asc"
	case "title_desc":
		orderClause = "musics.title desc"
	}
	dataSession = dataSession.Order(orderClause)

	offset := (params.Page - 1) * params.PerPage
	err := dataSession.Select("musics.*").
		Offset(offset).Limit(params.PerPage).
		Preload("User").Preload("MusicType").Preload("ModelType").
		Find(&musicList).Error

	if err != nil {
		log.Printf("Error in dataQuery for PublicMusic: %v", err)
		return nil, 0, fmt.Errorf("finding public music failed: %w", err)
	}

	return musicList, totalItems, nil
}
