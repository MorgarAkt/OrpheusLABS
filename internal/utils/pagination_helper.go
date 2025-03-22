package utils

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/morgarakt/aurify/internal/models"
)

func GetLibrary(musicList []models.Music, page int, perPage int, searchQuery string, musicType string) ([]gin.H, gin.H) {
	var filteredMusic []models.Music

	for _, music := range musicList {
		if musicType != "" && music.MusicType.Name != musicType {
			continue
		}

		if searchQuery != "" {
			searchLower := strings.ToLower(searchQuery)
			titleMatch := strings.Contains(strings.ToLower(music.Title), searchLower)
			creatorMatch := strings.Contains(strings.ToLower(music.User.Username), searchLower)
			typeMatch := strings.Contains(strings.ToLower(music.MusicType.Name), searchLower)

			if !titleMatch && !creatorMatch && !typeMatch {
				continue
			}
		}

		filteredMusic = append(filteredMusic, music)
	}

	var allMusic []gin.H
	for _, music := range filteredMusic {
		allMusic = append(allMusic, gin.H{
			"ID":           music.ID.String(),
			"Title":        music.Title,
			"creator":      music.User.Username,
			"filepath":     music.FilePath,
			"musictype":    music.MusicType.Name,
			"creationYear": music.CreatedAt.Year(),
			"image":        "/api/placeholder/400/400",
			"likes":        music.LikesCount,
		})
	}

	totalItems := len(allMusic)
	totalPages := (totalItems + perPage - 1) / perPage
	if page > totalPages && totalPages > 0 {
		page = totalPages
	} else if page < 1 || totalPages == 0 {
		page = 1
	}

	startIndex := (page - 1) * perPage
	endIndex := startIndex + perPage
	if endIndex > totalItems {
		endIndex = totalItems
	}

	var musics []gin.H
	if len(allMusic) == 0 {
		musics = []gin.H{}
	} else if startIndex < len(allMusic) {
		if endIndex <= len(allMusic) {
			musics = allMusic[startIndex:endIndex]
		} else {
			musics = allMusic[startIndex:]
		}
	} else {
		musics = []gin.H{}
	}

	pageNumbers := []int{}
	maxPages := 5
	if totalPages <= maxPages {
		for i := 1; i <= totalPages; i++ {
			pageNumbers = append(pageNumbers, i)
		}
	} else {
		if page <= 3 {
			for i := 1; i <= 5; i++ {
				pageNumbers = append(pageNumbers, i)
			}
		} else if page >= totalPages-2 {
			for i := totalPages - 4; i <= totalPages; i++ {
				pageNumbers = append(pageNumbers, i)
			}
		} else {
			for i := page - 2; i <= page+2; i++ {
				pageNumbers = append(pageNumbers, i)
			}
		}
	}

	pagination := gin.H{
		"CurrentPage": page,
		"TotalPages":  totalPages,
		"TotalItems":  totalItems,
		"PerPage":     perPage,
		"HasPrev":     page > 1,
		"HasNext":     page < totalPages,
		"PrevPage":    page - 1,
		"NextPage":    page + 1,
		"StartItem":   startIndex + 1,
		"EndItem":     endIndex,
		"Pages":       pageNumbers,
		"SearchQuery": searchQuery,
		"MusicType":   musicType,
	}

	return musics, pagination
}
