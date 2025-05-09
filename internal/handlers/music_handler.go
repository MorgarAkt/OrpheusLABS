// File: OrpheusLABS/internal/handlers/music_handler.go
package handlers

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings" // strings.ToLower vb. için

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/morgarakt/aurify/internal/config"
	middleware "github.com/morgarakt/aurify/internal/middlewares" // GetUserInfoFromContext için
	"github.com/morgarakt/aurify/internal/repository"
	"gorm.io/gorm" // gorm.ErrRecordNotFound için
)

// MusicHandler holds dependencies for music-related actions.
type MusicHandler struct {
	repo *repository.Repository
	cfg  *config.Config
}

// NewMusicHandler creates a new handler for music-related actions.
func NewMusicHandler(repo *repository.Repository, cfg *config.Config) *MusicHandler {
	return &MusicHandler{repo: repo, cfg: cfg}
}

// SaveMusicRequest struct'ı, müzik kaydetme/güncelleme isteği için.
type SaveMusicRequest struct {
	MusicID      string `json:"music_id" binding:"required,uuid"` // Güncellenecek müziğin ID'si
	Title        string `json:"title" binding:"required"`
	Mp3FilePath  string `json:"mp3_file_path"`                 // Genellikle güncellemede kullanılmaz
	MidiFilePath string `json:"midi_file_path"`                // Genellikle güncellemede kullanılmaz
	MusicType    string `json:"music_type" binding:"required"` // Genellikle güncellemede kullanılmaz
	ModelType    string `json:"model_type" binding:"required"` // Genellikle güncellemede kullanılmaz
	IsPublic     bool   `json:"is_public"`                     // Güncellenecek alanlardan biri
}

// UpdateMusicTitleRequest struct'ı başlık güncelleme isteği için
type UpdateMusicTitleRequest struct {
	Title string `json:"title" binding:"required,min=1,max=200"`
}

// GetMusicPage handles requests for individual music tracks via /musics/:id
func (h *MusicHandler) GetMusicPage(c *gin.Context) {
	musicIDStr := c.Param("id")
	musicID, err := uuid.Parse(musicIDStr)
	if err != nil {
		log.Printf("Invalid UUID format for music ID: %s, Error: %v", musicIDStr, err)
		h.renderNotFound(c)
		return
	}

	log.Printf("Fetching music page for ID: %s", musicID)

	music, err := h.repo.Music.GetByIDWithRelations(musicID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("Music not found for ID: %s", musicID)
			h.renderNotFound(c)
		} else {
			log.Printf("Error fetching music ID %s: %v", musicID, err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Could not retrieve music details."})
		}
		return
	}

	requestingUserID, requestingUsername, isAuthenticated := middleware.GetUserInfoFromContext(c)
	canAccess := false

	if music.IsPublic {
		canAccess = true
		log.Printf("Access granted to public music ID %s", musicID)
	} else if music.UserID == nil {
		canAccess = true
		log.Printf("Access granted to ownerless music ID %s", musicID)
	} else {
		if isAuthenticated && *music.UserID == requestingUserID {
			canAccess = true
			log.Printf("Access granted to private music ID %s for owner %s", musicID, requestingUserID)
		} else {
			log.Printf("Access DENIED to private music ID %s. Owner: %v, Requester: %s (Auth: %v)",
				musicID, *music.UserID, requestingUserID, isAuthenticated)
		}
	}

	if !canAccess {
		h.renderUnauthorized(c, !isAuthenticated, false)
		return
	}

	musicData := gin.H{
		"ID":             music.ID.String(),
		"Title":          music.Title,
		"Creator":        "Anonymous",
		"MusicType":      "Unknown",
		"ModelType":      "Unknown",
		"CreationYear":   music.CreatedAt.Year(),
		"Mp3FilePath":    music.Mp3FilePath,
		"MidiFilePath":   music.MidiFilePath,
		"CoverArtPath":   music.CoverArtPath, // Bu artık worker'dan gelen veya placeholder olabilir
		"IsPublic":       music.IsPublic,
		"Likes":          music.LikesCount,
		"Listens":        music.ListensCount,
		"GeneratedTitle": music.Title,       // Player partial'ı için
		"MusicID":        music.ID.String(), // Player partial'ı için (inline edit vb.)
		"Auth":           isAuthenticated,
		"Username":       requestingUsername,
	}
	if music.UserID != nil && music.User.Username != "" {
		musicData["Creator"] = music.User.Username
	}
	if music.MusicType.Name != "" {
		musicData["MusicType"] = music.MusicType.Name
	}
	if music.ModelType.Name != "" {
		musicData["ModelType"] = music.ModelType.Name
	}

	c.HTML(http.StatusOK, "music/detail.html", gin.H{
		"title":    fmt.Sprintf("%s - Aurify", music.Title),
		"auth":     isAuthenticated,
		"username": requestingUsername,
		"Music":    musicData,
	})
}

// UpdateMusicTitleHandler müziğin başlığını günceller
func (h *MusicHandler) UpdateMusicTitleHandler(c *gin.Context) {
	requestingUserID, _, isAuthenticated := middleware.GetUserInfoFromContext(c)
	if !isAuthenticated || requestingUserID == uuid.Nil {
		c.Header("HX-Trigger", `{"showNotification": {"type": "error", "message": "Please log in to edit titles."}}`)
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	musicIDStr := c.Param("id")
	musicID, err := uuid.Parse(musicIDStr)
	if err != nil {
		c.Header("HX-Trigger", `{"showNotification": {"type": "error", "message": "Invalid music ID format."}}`)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	var req UpdateMusicTitleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("Update title request binding error for MusicID %s: %v", musicID, err)
		// Hata mesajını daha spesifik hale getirebiliriz
		var errorMsg strings.Builder
		errorMsg.WriteString("Invalid title: ")
		// TODO: 'err' içeriğini daha iyi ayrıştırıp kullanıcıya göstermek
		// Şimdilik genel bir mesaj
		errorMsg.WriteString("Title must be between 1 and 200 characters.")

		c.Header("HX-Trigger", fmt.Sprintf(`{"showNotification": {"type": "error", "message": "%s"}}`, errorMsg.String()))
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	existingMusic, findErr := h.repo.Music.GetByIDWithRelations(musicID)
	if findErr != nil {
		if errors.Is(findErr, gorm.ErrRecordNotFound) {
			c.Header("HX-Trigger", `{"showNotification": {"type": "error", "message": "Music not found."}}`)
			c.AbortWithStatus(http.StatusNotFound)
		} else {
			log.Printf("Error finding music %s for title update: %v", musicID, findErr)
			c.Header("HX-Trigger", `{"showNotification": {"type": "error", "message": "Error finding music."}}`)
			c.AbortWithStatus(http.StatusInternalServerError)
		}
		return
	}

	if existingMusic.UserID == nil || *existingMusic.UserID != requestingUserID {
		log.Printf("User %s attempted to update title of music %s owned by %v", requestingUserID, musicID, existingMusic.UserID)
		c.Header("HX-Trigger", `{"showNotification": {"type": "error", "message": "You cannot change this title."}}`)
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	if existingMusic.Title == req.Title {
		log.Printf("Title for music %s is already '%s'. No update needed.", musicID, req.Title)
		c.String(http.StatusOK, req.Title) // Mevcut başlığı döndür, HTMX swap için
		return
	}

	existingMusic.Title = req.Title
	err = h.repo.Music.Update(existingMusic)
	if err != nil {
		log.Printf("Error updating title for music %s: %v", musicID, err)
		c.Header("HX-Trigger", `{"showNotification": {"type": "error", "message": "Failed to update title."}}`)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	log.Printf("Title for music %s updated to '%s' by user %s", musicID, req.Title, requestingUserID)
	c.String(http.StatusOK, req.Title) // Güncellenmiş başlığı metin olarak döndür
}

// SaveGeneratedMusicHandler artık mevcut bir müziği günceller (Title ve IsPublic).
func (h *MusicHandler) SaveGeneratedMusicHandler(c *gin.Context) {
	userID, _, auth := middleware.GetUserInfoFromContext(c)
	if !auth || userID == uuid.Nil {
		c.Header("HX-Trigger", `{"showNotification": {"type": "error", "message": "Please log in to save changes."}}`)
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	var req SaveMusicRequest // MusicID, Title, IsPublic alanlarını kullanır
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("Save/Update request binding error: %v", err)
		c.Header("HX-Trigger", `{"showNotification": {"type": "error", "message": "Invalid save request."}}`)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	musicID, _ := uuid.Parse(req.MusicID) // Binding uuid kontrolü yaptı
	existingMusic, findErr := h.repo.Music.GetByIDWithRelations(musicID)
	if findErr != nil {
		if errors.Is(findErr, gorm.ErrRecordNotFound) {
			c.Header("HX-Trigger", `{"showNotification": {"type": "error", "message": "Music not found."}}`)
			c.AbortWithStatus(http.StatusNotFound)
		} else {
			log.Printf("Error finding music %s for update: %v", musicID, findErr)
			c.Header("HX-Trigger", `{"showNotification": {"type": "error", "message": "Error finding music."}}`)
			c.AbortWithStatus(http.StatusInternalServerError)
		}
		return
	}

	if existingMusic.UserID == nil || *existingMusic.UserID != userID {
		log.Printf("User %s attempted to save/update music %s owned by %v", userID, musicID, existingMusic.UserID)
		c.Header("HX-Trigger", `{"showNotification": {"type": "error", "message": "You cannot modify this music."}}`)
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	updatePerformed := false
	if existingMusic.Title != req.Title {
		existingMusic.Title = req.Title
		updatePerformed = true
	}
	if existingMusic.IsPublic != req.IsPublic {
		existingMusic.IsPublic = req.IsPublic
		updatePerformed = true
	}
	// Diğer alanlar (Mp3FilePath, MusicType, ModelType) isteğe bağlı olarak güncellenebilir
	// ancak bu "Save" butonu genellikle sadece başlık/görünürlük gibi metadataları günceller.
	// Şimdilik bu alanlar güncellenmiyor.

	if updatePerformed {
		err := h.repo.Music.Update(existingMusic)
		if err != nil {
			log.Printf("Error updating music %s via Save button: %v", musicID, err)
			c.Header("HX-Trigger", `{"showNotification": {"type": "error", "message": "Failed to save changes."}}`)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		log.Printf("Music %s (Title/IsPublic) updated successfully by user %s via Save button", musicID, userID)
	} else {
		log.Printf("No changes detected for music %s via Save button, skipping update.", musicID)
	}

	c.Status(http.StatusOK) // İçerik döndürmeden başarılı yanıt
}

// renderNotFound renders the standard 404 page.
func (h *MusicHandler) renderNotFound(c *gin.Context) {
	_, username, auth := middleware.GetUserInfoFromContext(c)
	c.HTML(http.StatusNotFound, "error/notfound.html", gin.H{
		"title":    "Sayfa Bulunamadı - Aurify",
		"auth":     auth,
		"username": username,
	})
}

// renderUnauthorized renders the standard unauthorized page.
func (h *MusicHandler) renderUnauthorized(c *gin.Context, isLoginRequired bool, isAdminRequired bool) {
	_, username, auth := middleware.GetUserInfoFromContext(c)
	c.HTML(http.StatusForbidden, "error/unauthorized.html", gin.H{
		"title":           "Erişim Yetkiniz Yok",
		"auth":            auth,
		"username":        username,
		"IsLoginRequired": isLoginRequired,
		"IsAdminRequired": isAdminRequired,
		"RedirectURL":     c.Request.URL.Path,
	})
}
