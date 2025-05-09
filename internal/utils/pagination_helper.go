package utils

import (
	"log"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/morgarakt/aurify/internal/models"
)

// GetPaginatedMusic prepares music data for frontend display with pagination.
// It now correctly handles nullable UserID (*uuid.UUID).
// Pass a non-nil uuid pointer for userID for library filtering.
// Pass uuid.Nil for targetUserID if no user filtering is needed (e.g., public explore).
func GetPaginatedMusic(musicList []models.Music, page int, perPage int, searchQuery string, musicTypeFilter string, targetUserID uuid.UUID) ([]gin.H, gin.H) {

	var filteredMusic []models.Music

	for _, music := range musicList {
		if targetUserID != uuid.Nil {
			// Müziğin sahibi yoksa (UserID nil) veya
			// müziğin sahibi var ama istenen kullanıcı değilse, bu müziği atla.
			if music.UserID == nil || *music.UserID != targetUserID {
				continue
			}
		}

		if musicTypeFilter != "" && music.MusicType.Name != musicTypeFilter {
			continue
		}

		// Search Query filtresi (Title, Creator Username, Music Type Name)
		if searchQuery != "" {
			searchLower := strings.ToLower(searchQuery)
			titleMatch := strings.Contains(strings.ToLower(music.Title), searchLower)

			// Creator ve MusicType adını güvenli bir şekilde al
			creatorMatch := false
			// User ilişkisi yüklendi mi VE UserID null değil mi diye kontrol et
			if music.UserID != nil && music.User.Username != "" {
				creatorMatch = strings.Contains(strings.ToLower(music.User.Username), searchLower)
			}
			typeMatch := false
			if music.MusicType.Name != "" { // MusicType ilişkisi yüklendi mi?
				typeMatch = strings.Contains(strings.ToLower(music.MusicType.Name), searchLower)
			}

			if !titleMatch && !creatorMatch && !typeMatch {
				continue
			}
		}

		// Tüm filtrelerden geçtiyse listeye ekle
		filteredMusic = append(filteredMusic, music)
	}

	// Map to gin.H for template consumption
	var allMusicData []gin.H
	for _, music := range filteredMusic {
		// Creator adını güvenli bir şekilde al
		creatorUsername := "Anonymous" // Varsayılan
		if music.UserID != nil && music.User.Username != "" {
			creatorUsername = music.User.Username
		}

		// Diğer alanları güvenli bir şekilde al
		musicTypeName := "Unknown"
		if music.MusicType.Name != "" {
			musicTypeName = music.MusicType.Name
		}
		modelTypeName := "Unknown"
		if music.ModelType.Name != "" {
			modelTypeName = music.ModelType.Name
		}
		coverArt := music.CoverArtPath
		if coverArt == "" {
			coverArt = "/static/images/placeholder_cover.png"
		}
		title := music.Title
		if title == "" {
			title = "Untitled Generation"
		}

		allMusicData = append(allMusicData, gin.H{
			"ID":           music.ID.String(),
			"Title":        title,
			"Creator":      creatorUsername, // Güvenli erişim sonrası
			"MusicType":    musicTypeName,   // Güvenli erişim sonrası
			"ModelType":    modelTypeName,
			"CreationYear": music.CreatedAt.Year(),
			"Mp3FilePath":  music.Mp3FilePath,
			"MidiFilePath": music.MidiFilePath,
			"CoverArtPath": coverArt,
			"Likes":        music.LikesCount,
			"Listens":      music.ListensCount,
			"IsPublic":     music.IsPublic,
			// "IsOwner": music.UserID != nil && *music.UserID == targetUserID, // Gerekirse eklenebilir
		})
	}

	totalItems := len(allMusicData)
	totalPages := (totalItems + perPage - 1) / perPage
	if page > totalPages && totalPages > 0 {
		page = totalPages
	} else if page < 1 {
		page = 1
	}

	startIndex := (page - 1) * perPage
	endIndex := startIndex + perPage
	if endIndex > totalItems {
		endIndex = totalItems
	}

	var paginatedMusicData []gin.H
	if totalItems == 0 || startIndex >= totalItems {
		paginatedMusicData = []gin.H{}
	} else {
		paginatedMusicData = allMusicData[startIndex:endIndex]
	}

	// Page numbers for pagination controls (Değişiklik yok)
	pageNumbers := []int{}
	maxPagesToShow := 5
	if totalPages > 0 {
		if totalPages <= maxPagesToShow {
			for i := 1; i <= totalPages; i++ {
				pageNumbers = append(pageNumbers, i)
			}
		} else {
			startPage := page - (maxPagesToShow / 2)
			endPage := page + (maxPagesToShow / 2)
			if startPage < 1 {
				endPage += (1 - startPage)
				startPage = 1
			}
			if endPage > totalPages {
				startPage -= (endPage - totalPages)
				endPage = totalPages
			}
			if startPage < 1 {
				startPage = 1
			}
			for i := startPage; i <= endPage; i++ {
				pageNumbers = append(pageNumbers, i)
			}
		}
	}

	// Adjust StartItem/EndItem display for empty results (Değişiklik yok)
	displayStartItem := 0
	displayEndItem := 0
	if totalItems > 0 {
		displayStartItem = startIndex + 1
		displayEndItem = endIndex
	}

	paginationInfo := gin.H{
		"CurrentPage":     page,
		"TotalPages":      totalPages,
		"TotalItems":      totalItems,
		"PerPage":         perPage,
		"HasPrev":         page > 1,
		"HasNext":         page < totalPages,
		"PrevPage":        page - 1,
		"NextPage":        page + 1,
		"StartItem":       displayStartItem,
		"EndItem":         displayEndItem,
		"Pages":           pageNumbers,
		"SearchQuery":     searchQuery,
		"MusicTypeFilter": musicTypeFilter,
		// "SortBy": sortBy, // Sortlama eklenirse buraya eklenir
	}

	return paginatedMusicData, paginationInfo
}

// Helper function for GetLibrary in handler (specific to User's library)
// Artık targetUserID olarak uuid.UUID alıyor.
func GetUserLibraryPaginated(musicList []models.Music, page int, perPage int, searchQuery string, musicTypeFilter string, userID uuid.UUID) ([]gin.H, gin.H) {
	if userID == uuid.Nil {
		log.Println("Warning: GetUserLibraryPaginated called with nil userID.")
		return []gin.H{}, gin.H{} // Boş sonuç döndür
	}
	// Ana fonksiyona userID'yi doğrudan gönder
	return GetPaginatedMusic(musicList, page, perPage, searchQuery, musicTypeFilter, userID)
}

// Helper function for GetExplore in handler (public music, no user filter)
// Değişiklik yok, uuid.Nil göndererek kullanıcı filtresi olmadığını belirtiyor.
func GetExplorePaginated(musicList []models.Music, page int, perPage int, searchQuery string, musicTypeFilter string) ([]gin.H, gin.H) {
	var publicList []models.Music
	for _, m := range musicList {
		if m.IsPublic {
			publicList = append(publicList, m)
		}
	}
	return GetPaginatedMusic(publicList, page, perPage, searchQuery, musicTypeFilter, uuid.Nil) // Kullanıcı filtresi yok
}
