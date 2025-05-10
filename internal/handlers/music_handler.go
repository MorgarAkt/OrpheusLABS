package handlers

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/morgarakt/aurify/internal/config"
	middleware "github.com/morgarakt/aurify/internal/middlewares"
	"github.com/morgarakt/aurify/internal/models"
	"github.com/morgarakt/aurify/internal/repository"
	"gorm.io/gorm"
)

type MusicHandler struct {
	repo *repository.Repository
	cfg  *config.Config
}

func NewMusicHandler(repo *repository.Repository, cfg *config.Config) *MusicHandler {
	return &MusicHandler{repo: repo, cfg: cfg}
}

type UpdateMusicTitleRequest struct {
	Title string `json:"title" binding:"required,min=1,max=200"`
}

func (h *MusicHandler) GetMusicPage(c *gin.Context) {
	musicIDStr := c.Param("id")
	musicID, err := uuid.Parse(musicIDStr)
	if err != nil {
		log.Printf("Invalid UUID format for music ID: %s, Error: %v", musicIDStr, err)
		h.renderNotFound(c)
		return
	}

	music, err := h.repo.Music.GetByIDWithRelations(musicID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			h.renderNotFound(c)
		} else {
			log.Printf("Error fetching music ID %s for detail page: %v", musicID, err)
			c.HTML(http.StatusInternalServerError, "error/unauthorized.html", gin.H{
				"title":    "Hata",
				"auth":     middleware.IsAuthenticated(c),
				"username": c.GetString("username"),
				"Message":  "Müzik detayları alınamadı.",
			})
		}
		return
	}

	requestingUserID, requestingUsername, isAuthenticated := middleware.GetUserInfoFromContext(c)
	isOwner := false
	if isAuthenticated && music.UserID != nil && *music.UserID == requestingUserID {
		isOwner = true
	}

	if !music.IsPublic && !isOwner {
		h.renderUnauthorized(c, !isAuthenticated, false, c.Request.URL.RequestURI()) // Yönlendirme için tam URI
		return
	}

	hasLiked := false
	if isAuthenticated {
		hasLiked, _ = h.repo.UserLikes.HasUserLiked(requestingUserID, music.ID)
	}

	creatorUsername := "Anonymous"
	if music.User.ID != uuid.Nil {
		creatorUsername = music.User.Username
	} else if music.UserID != nil {
		var tempUser models.User
		if h.repo.User.GetByID(*music.UserID, &tempUser) == nil {
			creatorUsername = tempUser.Username
		}
	}

	musicCoverArt := music.CoverArtPath
	if musicCoverArt == "" {
		musicCoverArt = "/static/images/placeholder_cover.png"
	}
	musicTitle := music.Title
	if musicTitle == "" {
		musicTitle = "Untitled Track"
	}

	musicDataForTemplate := gin.H{
		"ID":                  music.ID.String(),
		"Title":               musicTitle,
		"Creator":             creatorUsername,
		"MusicType":           music.MusicType.Name,
		"ModelType":           music.ModelType.Name,
		"CreationYear":        music.CreatedAt.Year(),
		"Mp3Url":              music.Mp3FilePath,
		"MidiUrl":             music.MidiFilePath,
		"CoverArtPath":        musicCoverArt,
		"IsPublic":            music.IsPublic,
		"LikesCount":          music.LikesCount,
		"GeneratedTitle":      musicTitle,
		"MusicID":             music.ID.String(),
		"Auth":                isAuthenticated,
		"Username":            requestingUsername,
		"IsOwner":             isOwner,
		"HasLiked":            hasLiked,
		"IsDetailPageContext": true,
		"CurrentURLPath":      c.Request.URL.RequestURI(), // Paylaşım butonu ve login to save redirect için
	}

	c.HTML(http.StatusOK, "music/detail.html", gin.H{
		"title":    fmt.Sprintf("%s by %s - Aurify", musicTitle, creatorUsername),
		"auth":     isAuthenticated,
		"username": requestingUsername,
		"Music":    musicDataForTemplate,
	})
}

// ToggleLikeMusic, ToggleMusicVisibility, UpdateMusicTitleHandler (önceki yanıtlardaki gibi)
func (h *MusicHandler) ToggleLikeMusic(c *gin.Context) {
	userID, _, isAuthenticated := middleware.GetUserInfoFromContext(c)
	if !isAuthenticated {
		c.Header("HX-Reswap", "none")
		c.Header("HX-Trigger", `{"showNotification": {"type": "error", "message": "Please log in to like music."}}`)
		c.Status(http.StatusUnauthorized)
		return
	}

	musicIDStr := c.Param("id")
	musicID, err := uuid.Parse(musicIDStr)
	if err != nil {
		c.Header("HX-Reswap", "none")
		c.Header("HX-Trigger", `{"showNotification": {"type": "error", "message": "Invalid music ID."}}`)
		c.Status(http.StatusBadRequest)
		return
	}

	_, err = h.repo.Music.GetByIDWithRelations(musicID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.Header("HX-Reswap", "none")
			c.Header("HX-Trigger", `{"showNotification": {"type": "error", "message": "Music not found."}}`)
			c.Status(http.StatusNotFound)
			return
		}
		log.Printf("Error checking music existence for like toggle (MusicID: %s): %v", musicID, err)
		c.Header("HX-Reswap", "none")
		c.Header("HX-Trigger", `{"showNotification": {"type": "error", "message": "Could not process like."}}`)
		c.Status(http.StatusInternalServerError)
		return
	}

	hasLiked, err := h.repo.UserLikes.HasUserLiked(userID, musicID)
	if err != nil {
		log.Printf("Error checking if user %s liked music %s: %v", userID, musicID, err)
		c.Header("HX-Reswap", "none")
		c.Header("HX-Trigger", `{"showNotification": {"type": "error", "message": "Could not process like."}}`)
		c.Status(http.StatusInternalServerError)
		return
	}

	var likeChange int
	if hasLiked {
		if err := h.repo.UserLikes.RemoveLike(userID, musicID); err != nil {
			log.Printf("Error removing like for user %s, music %s: %v", userID, musicID, err)
			c.Header("HX-Reswap", "none")
			c.Header("HX-Trigger", `{"showNotification": {"type": "error", "message": "Failed to unlike."}}`)
			c.Status(http.StatusInternalServerError)
			return
		}
		likeChange = -1
	} else {
		likeEntry := &models.UserLikesMusic{
			UserID:    userID,
			MusicID:   musicID,
			CreatedAt: time.Now(),
		}
		if err := h.repo.UserLikes.AddLike(likeEntry); err != nil {
			log.Printf("Error adding like for user %s, music %s: %v", userID, musicID, err)
			c.Header("HX-Reswap", "none")
			c.Header("HX-Trigger", `{"showNotification": {"type": "error", "message": "Failed to like."}}`)
			c.Status(http.StatusInternalServerError)
			return
		}
		likeChange = 1
	}

	if err := h.repo.Music.UpdateLikesCount(musicID, likeChange); err != nil {
		log.Printf("Error updating likes_count for music %s: %v", musicID, err)
	}

	updatedMusic, err := h.repo.Music.GetByIDWithRelations(musicID) // Beğeni sayısını yeniden oku
	if err != nil {                                                 // Hata durumunda logla ama devam et, en azından UI güncellensin
		log.Printf("Error refetching music after like for music %s: %v", musicID, err)
		// newLikesCount'u manuel olarak ayarla
		currentMusic, _ := h.repo.Music.GetByIDWithRelations(musicID) // Eski değeri al
		if currentMusic != nil {
			newLikesCount := currentMusic.LikesCount // Hata olsa bile eski değeri kullanmaya çalış
			c.HTML(http.StatusOK, "partials/_like_button_partial.html", gin.H{
				"ID":         musicID.String(),
				"LikesCount": newLikesCount,
				"HasLiked":   !hasLiked,
				"Auth":       true,
			})
			return
		}
		c.Header("HX-Reswap", "none")
		c.Header("HX-Trigger", `{"showNotification": {"type": "error", "message": "Like status updated, but count refresh failed."}}`)
		c.Status(http.StatusInternalServerError)
		return
	}
	newLikesCount := updatedMusic.LikesCount

	c.HTML(http.StatusOK, "partials/_like_button_partial.html", gin.H{
		"ID":         musicID.String(),
		"LikesCount": newLikesCount,
		"HasLiked":   !hasLiked,
		"Auth":       true,
	})
}

func (h *MusicHandler) ToggleMusicVisibility(c *gin.Context) {
	requestingUserID, _, isAuthenticated := middleware.GetUserInfoFromContext(c)
	if !isAuthenticated {
		c.Header("HX-Reswap", "none")
		c.Header("HX-Trigger", `{"showNotification": {"type": "error", "message": "Please log in."}}`)
		c.Status(http.StatusUnauthorized)
		return
	}

	musicIDStr := c.Param("id")
	musicID, err := uuid.Parse(musicIDStr)
	if err != nil {
		c.Header("HX-Reswap", "none")
		c.Header("HX-Trigger", `{"showNotification": {"type": "error", "message": "Invalid music ID."}}`)
		c.Status(http.StatusBadRequest)
		return
	}

	// Müziği ve mevcut IsPublic durumunu al
	music, err := h.repo.Music.GetByIDWithRelations(musicID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.Header("HX-Reswap", "none")
			c.Header("HX-Trigger", `{"showNotification": {"type": "error", "message": "Music not found."}}`)
			c.Status(http.StatusNotFound)
		} else {
			log.Printf("Error fetching music %s for visibility toggle: %v", musicID, err)
			c.Header("HX-Reswap", "none")
			c.Header("HX-Trigger", `{"showNotification": {"type": "error", "message": "Could not retrieve music details."}}`)
			c.Status(http.StatusInternalServerError)
		}
		return
	}

	// Sahiplik kontrolü
	if music.UserID == nil || *music.UserID != requestingUserID {
		c.Header("HX-Reswap", "none")
		c.Header("HX-Trigger", `{"showNotification": {"type": "error", "message": "You are not authorized to change visibility."}}`)
		c.Status(http.StatusForbidden)
		return
	}

	// Mevcut IsPublic durumunun tersini al
	newIsPublicState := !music.IsPublic

	// Veritabanını yeni durumla güncelle
	if err := h.repo.Music.UpdateVisibility(musicID, newIsPublicState); err != nil {
		log.Printf("Error updating visibility for music %s to %t: %v", musicID, newIsPublicState, err)
		c.Header("HX-Reswap", "none")
		c.Header("HX-Trigger", `{"showNotification": {"type": "error", "message": "Failed to update visibility."}}`)
		c.Status(http.StatusInternalServerError)
		return
	}

	log.Printf("Visibility for music %s updated to %t by user %s", musicID, newIsPublicState, requestingUserID)

	// Partial'ı render ederken YENİ durumu kullan
	c.HTML(http.StatusOK, "partials/_visibility_toggle_partial.html", gin.H{
		"MusicID":  musicID.String(),
		"IsPublic": newIsPublicState, // Veritabanından alınan ve tersi çevrilen yeni durum
		"IsOwner":  true,
	})
}

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
		var errorMsg strings.Builder
		errorMsg.WriteString("Invalid title: ")
		errorMsg.WriteString(err.Error())

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
		c.String(http.StatusOK, req.Title)
		return
	}

	existingMusic.Title = req.Title
	if errUpdate := h.repo.Music.Update(existingMusic); errUpdate != nil {
		log.Printf("Error updating title for music %s: %v", musicID, errUpdate)
		c.Header("HX-Trigger", `{"showNotification": {"type": "error", "message": "Failed to update title."}}`)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	log.Printf("Title for music %s updated to '%s' by user %s", musicID, req.Title, requestingUserID)
	c.String(http.StatusOK, req.Title)
}

func (h *MusicHandler) renderNotFound(c *gin.Context) {
	_, username, auth := middleware.GetUserInfoFromContext(c)
	c.HTML(http.StatusNotFound, "error/notfound.html", gin.H{
		"title":    "Sayfa Bulunamadı - Aurify",
		"auth":     auth,
		"username": username,
	})
}

func (h *MusicHandler) renderUnauthorized(c *gin.Context, isLoginRequired bool, isAdminRequired bool, currentPathForRedirect ...string) {
	_, username, auth := middleware.GetUserInfoFromContext(c)

	redirectPath := c.Request.URL.RequestURI()
	if len(currentPathForRedirect) > 0 && currentPathForRedirect[0] != "" {
		redirectPath = currentPathForRedirect[0]
	}

	data := gin.H{
		"title":           "Erişim Yetkiniz Yok",
		"auth":            auth,
		"username":        username,
		"IsLoginRequired": isLoginRequired,
		"IsAdminRequired": isAdminRequired,
		"RedirectURL":     url.QueryEscape(redirectPath),
	}
	if auth && !isLoginRequired && !isAdminRequired {
		// Bu mesaj zaten unauthorized.html içinde genel bir fallback olarak var,
		// ama burada daha spesifik bir mesaj eklenebilir istenirse.
		// data["MessageSpecific"] = "Bu özel içeriği görüntüleme yetkiniz bulunmamaktadır."
	}
	c.HTML(http.StatusForbidden, "error/unauthorized.html", data)
}
