package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"math" // Min fonksiyonu için
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/morgarakt/aurify/internal/config"                 // Projenizin config yolu
	middleware "github.com/morgarakt/aurify/internal/middlewares" // Projenizin middleware yolu
	"github.com/morgarakt/aurify/internal/models"                 // Projenizin model yolu
	"github.com/morgarakt/aurify/internal/repository"             // Projenizin repository yolu
	"github.com/morgarakt/aurify/internal/services"               // Projenizin services yolu
	"github.com/morgarakt/aurify/internal/utils"                  // Projenizin utils yolu
)

type FrontendHandler struct {
	repo           *repository.Repository
	cfg            *config.Config
	rabbitmqClient *services.RabbitMQClient
}

func NewFrontendHandler(repo *repository.Repository, cfg *config.Config, rmqClient *services.RabbitMQClient) *FrontendHandler {
	return &FrontendHandler{
		repo:           repo,
		cfg:            cfg,
		rabbitmqClient: rmqClient,
	}
}

// GenerateMusicRabbitMQRequest ve Response struct'ları aynı kalabilir.
type GenerateMusicRabbitMQRequest struct {
	TaskID string                 `json:"task_id"`
	Params map[string]interface{} `json:"params"`
}

type GenerateMusicRabbitMQResponse struct {
	TaskID          string `json:"task_id"`
	Status          string `json:"status"`
	Message         string `json:"message,omitempty"`
	Mp3Url          string `json:"mp3_url,omitempty"`
	MidiUrl         string `json:"midi_url,omitempty"`
	RawMidiUrl      string `json:"raw_midi_url,omitempty"`
	WavUrl          string `json:"wav_url,omitempty"`
	ImageUrl        string `json:"image_url,omitempty"`
	ModelUsed       string `json:"model_used,omitempty"`
	LengthGenerated int    `json:"length_generated,omitempty"`
}

// getPerPageFromQueryOrDefault config dosyasından varsayılanları alabilir.
// Şimdilik sabit değerler kullanılıyor.
func getPerPageFromQueryOrDefault(c *gin.Context, defaultVal, minVal, maxVal int) int {
	s := c.Query("per_page")
	if s == "" {
		return defaultVal
	}
	i, err := strconv.Atoi(s)
	if err != nil || i < minVal {
		return minVal
	}
	if i > maxVal {
		return maxVal
	}
	return i
}

// min helper for int64
func minInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func (h *FrontendHandler) preparePaginationData(
	c *gin.Context,
	musicList []models.Music,
	totalItems int64,
	page int,
	perPage int,
	viewPath string,
	currentQuery url.Values,
	isAuthenticated bool,
	requestingUserID uuid.UUID,
) ([]gin.H, gin.H) {
	formattedMusic := []gin.H{}
	for _, m := range musicList {
		hasLiked := false
		if isAuthenticated && requestingUserID != uuid.Nil {
			var errLikeCheck error
			hasLiked, errLikeCheck = h.repo.UserLikes.HasUserLiked(requestingUserID, m.ID)
			if errLikeCheck != nil {
				log.Printf("Error checking like status for music %s, user %s: %v", m.ID, requestingUserID, errLikeCheck)
			}
		}

		isOwner := false
		if isAuthenticated && m.UserID != nil && *m.UserID == requestingUserID {
			isOwner = true
		}

		creatorUsername := "Anonymous"
		if m.User.ID != uuid.Nil && m.User.Username != "" {
			creatorUsername = m.User.Username
		}
		musicTypeName := "Unknown"
		if m.MusicType.ID != uuid.Nil && m.MusicType.Name != "" {
			musicTypeName = m.MusicType.Name
		}
		modelTypeName := "Unknown"
		if m.ModelType.ID != uuid.Nil && m.ModelType.Name != "" {
			modelTypeName = m.ModelType.Name
		}
		coverArtPath := m.CoverArtPath
		if coverArtPath == "" {
			coverArtPath = "/static/images/placeholder_cover.png"
		}
		title := m.Title
		if title == "" {
			title = "Untitled Track"
		}

		formattedMusic = append(formattedMusic, gin.H{
			"ID":           m.ID.String(),
			"Title":        title,
			"Creator":      creatorUsername,
			"MusicType":    musicTypeName,
			"ModelType":    modelTypeName,
			"CreationYear": m.CreatedAt.Year(),
			"Mp3FilePath":  m.Mp3FilePath,  // Kartta play butonu için
			"MidiFilePath": m.MidiFilePath, // Kartta indirme için
			"CoverArtPath": coverArtPath,
			"LikesCount":   m.LikesCount,
			"HasLiked":     hasLiked,
			"IsOwner":      isOwner,    // YENİ EKLENDİ
			"IsPublic":     m.IsPublic, // _visibility_toggle_partial için kartta da gerekebilir
		})
	}

	// ... (Sayfalama hesaplamaları (totalPages, pageNumbers, linkParams, startItem, endItem) aynı kalır)
	// Bir önceki yanıttaki gibi devam eder...
	totalPages := 0
	if totalItems > 0 && perPage > 0 {
		totalPages = int(math.Ceil(float64(totalItems) / float64(perPage)))
	}
	if page > totalPages && totalPages > 0 {
		page = totalPages
	}
	if page < 1 && totalItems > 0 {
		page = 1
	}
	if totalItems == 0 {
		page = 1
		totalPages = 0
	}

	pageNumbers := []int{}
	maxPagesToShow := 5
	if totalPages > 0 {
		if totalPages <= maxPagesToShow {
			for i := 1; i <= totalPages; i++ {
				pageNumbers = append(pageNumbers, i)
			}
		} else {
			startP := page - (maxPagesToShow / 2)
			endP := page + (maxPagesToShow / 2)
			if maxPagesToShow%2 == 0 {
				endP--
			}
			if startP < 1 {
				endP += (1 - startP)
				startP = 1
			}
			if endP > totalPages {
				startP -= (endP - totalPages)
				endP = totalPages
			}
			if startP < 1 {
				startP = 1
			}
			for i := startP; i <= endP; i++ {
				pageNumbers = append(pageNumbers, i)
			}
		}
	}

	linkParams := url.Values{}
	if qVal := currentQuery.Get("q"); qVal != "" {
		linkParams.Set("q", qVal)
	}
	if mtVal := currentQuery.Get("musictype"); mtVal != "" {
		linkParams.Set("musictype", mtVal)
	}
	if sortVal := currentQuery.Get("sort"); sortVal != "" {
		linkParams.Set("sort", sortVal)
	}
	linkParams.Set("per_page", strconv.Itoa(perPage))
	linkQueryString := linkParams.Encode()

	var startItem, endItem int64
	if totalItems > 0 {
		startItem = int64((page-1)*perPage + 1)
		endItem = minInt64(int64(page*perPage), totalItems)
	} else {
		startItem = 0
		endItem = 0
	}

	pagination := gin.H{
		"CurrentPage":     page,
		"TotalPages":      totalPages,
		"TotalItems":      totalItems,
		"PerPage":         perPage,
		"HasPrev":         page > 1 && totalItems > 0,
		"HasNext":         page < totalPages,
		"PrevPage":        page - 1,
		"NextPage":        page + 1,
		"Pages":           pageNumbers,
		"SearchQuery":     currentQuery.Get("q"),
		"MusicTypeFilter": currentQuery.Get("musictype"),
		"SortBy":          currentQuery.Get("sort"),
		"BaseLink":        viewPath,
		"LinkQuery":       linkQueryString,
		"StartItem":       startItem,
		"EndItem":         endItem,
	}
	return formattedMusic, pagination
}

func (h *FrontendHandler) GetMusicsLibrary(c *gin.Context) {
	userID, _, isAuthenticated := middleware.GetUserInfoFromContext(c) // Auth bilgisini de al
	if !isAuthenticated {                                              // Auth bilgisini kontrol et
		c.Header("HX-Trigger", `{"showNotification": {"type": "error", "message": "Please log in to view your library."}}`)
		c.Status(http.StatusUnauthorized)
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	perPage := getPerPageFromQueryOrDefault(c, h.cfg.DefaultPerPage, h.cfg.MinPerPage, h.cfg.MaxPerPage)

	queryParams := repository.MusicQueryParams{
		SearchQuery:     c.Query("q"),
		MusicTypeFilter: c.Query("musictype"),
		SortBy:          c.Query("sort"),
		Page:            page,
		PerPage:         perPage,
	}

	musicList, totalItems, err := h.repo.Music.QueryUserMusic(userID, queryParams)
	if err != nil {
		log.Printf("Error querying user music for %s: %v", userID, err)
		c.Header("HX-Trigger", `{"showNotification": {"type": "error", "message": "Could not load your music."}}`)
		c.Status(http.StatusInternalServerError)
		return
	}

	viewPath := "/library"
	pushedURLValues := c.Request.URL.Query()               // Gelen tüm query parametrelerini al
	pushedURLValues.Set("page", strconv.Itoa(page))        // Sayfayı güncelle/ekle
	pushedURLValues.Set("per_page", strconv.Itoa(perPage)) // per_page'i de URL'e ekle
	c.Header("HX-Push-Url", viewPath+"?"+pushedURLValues.Encode())

	musicsGinH, paginationData := h.preparePaginationData(c, musicList, totalItems, page, perPage, viewPath, c.Request.URL.Query(), isAuthenticated, userID)

	c.HTML(http.StatusOK, "partials/musics-pagination.html", gin.H{
		"Music":      musicsGinH,
		"Pagination": paginationData,
		"Auth":       isAuthenticated, // Auth bilgisini partial'a gönder
	})
}

// GetExploreMusicData API Handler'ını güncelle
func (h *FrontendHandler) GetExploreMusicData(c *gin.Context) {
	requestingUserID, _, isAuthenticated := middleware.GetUserInfoFromContext(c) // Beğeni durumu için kullanıcı ID'si ve Auth durumu

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	perPage := getPerPageFromQueryOrDefault(c, h.cfg.DefaultPerPage, h.cfg.MinPerPage, h.cfg.MaxPerPage)

	queryParams := repository.MusicQueryParams{
		SearchQuery:     c.Query("q"),
		MusicTypeFilter: c.Query("musictype"),
		SortBy:          c.Query("sort"),
		Page:            page,
		PerPage:         perPage,
	}

	musicList, totalItems, err := h.repo.Music.QueryPublicMusic(queryParams)
	if err != nil {
		log.Printf("Error querying public music: %v", err)
		c.Header("HX-Trigger", `{"showNotification": {"type": "error", "message": "Could not load music to explore."}}`)
		c.Status(http.StatusInternalServerError)
		return
	}

	viewPath := "/explore"
	pushedURLValues := c.Request.URL.Query()
	pushedURLValues.Set("page", strconv.Itoa(page))
	pushedURLValues.Set("per_page", strconv.Itoa(perPage))
	c.Header("HX-Push-Url", viewPath+"?"+pushedURLValues.Encode())

	musicsGinH, paginationData := h.preparePaginationData(c, musicList, totalItems, page, perPage, viewPath, c.Request.URL.Query(), isAuthenticated, requestingUserID)

	c.HTML(http.StatusOK, "partials/musics-pagination.html", gin.H{
		"Music":      musicsGinH,
		"Pagination": paginationData,
		"Auth":       isAuthenticated, // Auth bilgisini partial'a gönder
	})
}

// Library ve Explore ana sayfa handler'larını da Auth bilgisini ve userID'yi preparePaginationData'ya geçecek şekilde güncelle
func (h *FrontendHandler) Library(c *gin.Context) {
	userID, username, isAuthenticated := middleware.GetUserInfoFromContext(c)
	if !isAuthenticated {
		c.Redirect(http.StatusSeeOther, "/login?redirect=/library")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	perPage := getPerPageFromQueryOrDefault(c, h.cfg.DefaultPerPage, h.cfg.MinPerPage, h.cfg.MaxPerPage)

	musicTypes, _ := h.repo.MusicType.GetAll()

	queryParams := repository.MusicQueryParams{
		SearchQuery:     c.Query("q"),
		MusicTypeFilter: c.Query("musictype"),
		SortBy:          c.Query("sort"),
		Page:            page,
		PerPage:         perPage,
	}
	musicList, totalItems, err := h.repo.Music.QueryUserMusic(userID, queryParams)
	if err != nil {
		log.Printf("Error initial load QueryUserMusic for %s: %v", userID, err)
		c.HTML(http.StatusInternalServerError, "error/unauthorized.html", gin.H{"title": "Error", "message": "Could not load your library."})
		return
	}

	musicsGinH, paginationData := h.preparePaginationData(c, musicList, totalItems, page, perPage, "/library", c.Request.URL.Query(), isAuthenticated, userID)

	c.HTML(http.StatusOK, "music/library.html", gin.H{
		"title":          "Your Music Library - Aurify",
		"auth":           isAuthenticated,
		"username":       username,
		"MusicType":      musicTypes,
		"Music":          musicsGinH,
		"Pagination":     paginationData,
		"HXGetURL":       "/api/v1/music",
		"InitialPerPage": perPage,
	})
}

func (h *FrontendHandler) Explore(c *gin.Context) {
	requestingUserID, username, isAuthenticated := middleware.GetUserInfoFromContext(c)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	perPage := getPerPageFromQueryOrDefault(c, h.cfg.DefaultPerPage, h.cfg.MinPerPage, h.cfg.MaxPerPage)

	musicTypes, _ := h.repo.MusicType.GetAll()

	queryParams := repository.MusicQueryParams{
		SearchQuery:     c.Query("q"),
		MusicTypeFilter: c.Query("musictype"),
		SortBy:          c.Query("sort"),
		Page:            page,
		PerPage:         perPage,
	}
	musicList, totalItems, err := h.repo.Music.QueryPublicMusic(queryParams)
	if err != nil {
		log.Printf("Error initial load QueryPublicMusic: %v", err)
		c.HTML(http.StatusInternalServerError, "error/unauthorized.html", gin.H{"title": "Error", "message": "Could not load music to explore."})
		return
	}

	musicsGinH, paginationData := h.preparePaginationData(c, musicList, totalItems, page, perPage, "/explore", c.Request.URL.Query(), isAuthenticated, requestingUserID)

	c.HTML(http.StatusOK, "music/explore.html", gin.H{
		"title":          "Explore Music - Aurify",
		"auth":           isAuthenticated,
		"username":       username,
		"MusicType":      musicTypes,
		"Music":          musicsGinH,
		"Pagination":     paginationData,
		"HXGetURL":       "/api/v1/explore-music-data",
		"InitialPerPage": perPage,
	})
}

// HomePage, Login, Register, NotFoundPage, GenerateMusicHandler, GetEditTitleFormPartial, GetTitleTextPartial metodları aynı kalabilir.
// ... (diğer handler metodlarınız)
func (h *FrontendHandler) HomePage(c *gin.Context) {
	userID, username, auth := middleware.GetUserInfoFromContext(c)
	musicTypes, err := h.repo.MusicType.GetAll()
	if err != nil {
		log.Printf("Error fetching music types: %v", err)
	}
	modelTypes, err := h.repo.ModelType.GetAll()
	if err != nil {
		log.Printf("Error fetching model types: %v", err)
	}
	data := gin.H{
		"title":     "Aurify - Create an aura",
		"auth":      auth,
		"userID":    userID.String(), // UUID'yi string'e çevir
		"username":  username,
		"MusicType": musicTypes,
		"ModelType": modelTypes,
	}
	c.HTML(http.StatusOK, "general/home.html", data)
}

func (h *FrontendHandler) Login(c *gin.Context) {
	_, _, auth := middleware.GetUserInfoFromContext(c)
	if auth {
		c.Redirect(http.StatusFound, "/")
		return
	}
	data := gin.H{"title": "Login to Aurify", "auth": false}
	c.HTML(http.StatusOK, "auth/login.html", data)
}

func (h *FrontendHandler) Register(c *gin.Context) {
	_, _, auth := middleware.GetUserInfoFromContext(c)
	if auth {
		c.Redirect(http.StatusFound, "/")
		return
	}
	data := gin.H{"title": "Register to Aurify", "auth": false}
	c.HTML(http.StatusOK, "auth/register.html", data)
}

func (h *FrontendHandler) GenerateMusicHandler(c *gin.Context) {
	userID, username, auth := middleware.GetUserInfoFromContext(c)
	var formReq struct {
		MusicType string `json:"musicType"`
		AIModel   string `json:"aiModel"`
	}
	if err := c.ShouldBindJSON(&formReq); err != nil {
		log.Printf("Error binding JSON in GenerateMusicHandler: %v", err)
		c.HTML(http.StatusBadRequest, "partials/play_button.html", gin.H{"Error": "Invalid request format.", "auth": auth})
		return
	}
	musicTypeName := formReq.MusicType
	aiModelName := formReq.AIModel
	log.Printf("DEBUG: Received Generation Request - MusicType: '%s', AIModel: '%s'", musicTypeName, aiModelName)
	if musicTypeName == "" || aiModelName == "" {
		c.HTML(http.StatusBadRequest, "partials/play_button.html", gin.H{"Error": "Music Type and AI Model are required.", "auth": auth})
		return
	}
	if h.rabbitmqClient == nil {
		log.Println("CRITICAL: RabbitMQ client is not initialized in FrontendHandler")
		c.HTML(http.StatusInternalServerError, "partials/play_button.html", gin.H{"Error": "Generation service is currently unavailable.", "auth": auth})
		return
	}
	reqParams := map[string]interface{}{
		"run_mode": aiModelName, "model_type": aiModelName, "music_type": musicTypeName,
		"start_sequence": []int{60, 64, 67, 72}, "length": 150, "temperature": 0.85,
		"bpm": 120, "note_duration": 0.4, "instrument_program": 0,
	}
	taskID := uuid.New().String()
	rabbitRequest := GenerateMusicRabbitMQRequest{TaskID: taskID, Params: reqParams}
	requestBody, err := json.Marshal(rabbitRequest)
	if err != nil {
		log.Printf("Error marshalling RabbitMQ request: %v", err)
		c.HTML(http.StatusInternalServerError, "partials/play_button.html", gin.H{"Error": "Failed to prepare generation request.", "auth": auth})
		return
	}
	log.Printf("Sending request to RabbitMQ (TaskID: %s)...", taskID)
	responseBody, err := h.rabbitmqClient.Call("music_requests", requestBody, 60*time.Second)
	if err != nil {
		log.Printf("Error calling RabbitMQ service (TaskID: %s): %v", taskID, err)
		c.HTML(http.StatusInternalServerError, "partials/play_button.html", gin.H{"Error": "Music generation timed out or service is unavailable.", "auth": auth})
		return
	}
	log.Printf("Received response from RabbitMQ (TaskID: %s)", taskID)
	var rabbitResponse GenerateMusicRabbitMQResponse
	if err := json.Unmarshal(responseBody, &rabbitResponse); err != nil {
		log.Printf("Error unmarshalling RabbitMQ response (TaskID: %s): %v\nResponse Body: %s", taskID, err, string(responseBody))
		c.HTML(http.StatusInternalServerError, "partials/play_button.html", gin.H{"Error": "Invalid response from generation service.", "auth": auth})
		return
	}
	log.Printf("DEBUG: RabbitMQ Response Parsed - Status: '%s', ModelUsed: '%s', ImageUrl: '%s'", rabbitResponse.Status, rabbitResponse.ModelUsed, rabbitResponse.ImageUrl)
	if rabbitResponse.Status == "error" {
		log.Printf("Music generation failed via worker (TaskID: %s): %s", taskID, rabbitResponse.Message)
		c.HTML(http.StatusInternalServerError, "partials/play_button.html", gin.H{"Error": fmt.Sprintf("Generation failed: %s", rabbitResponse.Message), "auth": auth})
		return
	}
	if rabbitResponse.Status != "completed" || (rabbitResponse.Mp3Url == "" && rabbitResponse.MidiUrl == "") {
		log.Printf("Music generation incomplete or invalid response (TaskID: %s): Status=%s, Mp3=%s, Midi=%s", taskID, rabbitResponse.Status, rabbitResponse.Mp3Url, rabbitResponse.MidiUrl)
		c.HTML(http.StatusInternalServerError, "partials/play_button.html", gin.H{"Error": "Generation service returned an incomplete or invalid result.", "auth": auth})
		return
	}

	musicType, err := h.repo.MusicType.GetByName(musicTypeName)
	if err != nil {
		log.Printf("Error finding MusicType '%s' for saving: %v", musicTypeName, err)
		c.HTML(http.StatusInternalServerError, "partials/play_button.html", gin.H{"Error": "Failed to process music type details.", "auth": auth})
		return
	}
	modelType, err := h.repo.ModelType.GetByName(rabbitResponse.ModelUsed)
	if err != nil {
		log.Printf("Error finding ModelType '%s' for saving: %v", rabbitResponse.ModelUsed, err)
		c.HTML(http.StatusInternalServerError, "partials/play_button.html", gin.H{"Error": "Failed to process AI model details.", "auth": auth})
		return
	}

	generatedTitle := utils.GenerateCreativeTitle(musicTypeName, rabbitResponse.ModelUsed)
	log.Printf("Generated creative title: %s", generatedTitle)

	coverArtPath := "/static/images/placeholder_cover.png"
	if rabbitResponse.ImageUrl != "" {
		coverArtPath = rabbitResponse.ImageUrl
		log.Printf("Using cover art from worker: %s", coverArtPath)
	} else {
		log.Printf("No image_url from worker, using placeholder for cover art.")
	}

	newMusic := models.Music{
		Title: generatedTitle, Mp3FilePath: rabbitResponse.Mp3Url, MidiFilePath: rabbitResponse.MidiUrl,
		CoverArtPath: coverArtPath, IsPublic: false, // Varsayılan olarak özel
		MusicTypeID: musicType.ID, ModelTypeID: modelType.ID,
	}
	if auth && userID != uuid.Nil {
		newMusic.UserID = &userID
	}
	if err := h.repo.Music.Create(&newMusic); err != nil {
		log.Printf("!!! CRITICAL: Failed to auto-save music (TaskID: %s, UserID: %s): %v", taskID, userID, err)
		c.HTML(http.StatusInternalServerError, "partials/play_button.html", gin.H{"Error": "Failed to save the generated music.", "auth": auth})
		return
	}
	log.Printf("Music auto-saved: ID=%s, UserID=%s, CoverArt=%s", newMusic.ID, userID, newMusic.CoverArtPath)

	data := gin.H{
		"Auth": auth, "Username": username, "Mp3Url": rabbitResponse.Mp3Url, "MidiUrl": rabbitResponse.MidiUrl,
		"CoverArtPath": newMusic.CoverArtPath, "MusicType": musicTypeName, "ModelType": rabbitResponse.ModelUsed,
		"GeneratedTitle": generatedTitle, "MusicID": "",
	}
	if newMusic.ID != uuid.Nil {
		data["MusicID"] = newMusic.ID.String()
	}
	c.HTML(http.StatusOK, "partials/music_player.html", data)
}

func (h *FrontendHandler) NotFoundPage(c *gin.Context) {
	_, username, auth := middleware.GetUserInfoFromContext(c)
	c.HTML(http.StatusNotFound, "error/notfound.html", gin.H{
		"title": "Sayfa Bulunamadı - Aurify", "auth": auth, "username": username,
	})
}

func (h *FrontendHandler) GetEditTitleFormPartial(c *gin.Context) {
	musicID := c.Query("musicID")
	currentTitle := c.Query("currentTitle")
	if _, err := uuid.Parse(musicID); err != nil {
		c.String(http.StatusBadRequest, "Invalid Music ID format provided to partial.")
		return
	}
	c.HTML(http.StatusOK, "partials/edit-title-form.html", gin.H{
		"MusicID":      musicID,
		"CurrentTitle": currentTitle,
	})
}

func (h *FrontendHandler) GetTitleTextPartial(c *gin.Context) {
	musicID := c.Query("musicID")
	currentTitle := c.Query("currentTitle")
	if _, err := uuid.Parse(musicID); err != nil {
		c.String(http.StatusBadRequest, "Invalid Music ID format provided to partial.")
		return
	}
	c.HTML(http.StatusOK, "partials/title-text.html", gin.H{
		"MusicID":      musicID,
		"CurrentTitle": currentTitle,
	})
}
