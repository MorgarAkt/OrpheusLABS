// Modify: OrpheusLABS/internal/handlers/auth_handler.go
package handlers

import (
	// "fmt" // fmt loglama veya debug için kullanılmıyorsa kaldırılabilir
	"log" // Hata loglama için eklendi
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/morgarakt/aurify/internal/config"
	"github.com/morgarakt/aurify/internal/models"
	"github.com/morgarakt/aurify/internal/repository"
	"github.com/morgarakt/aurify/internal/utils"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	repo *repository.Repository
	cfg  *config.Config
}

func NewAuthHandler(repo *repository.Repository, cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		repo: repo,
		cfg:  cfg,
	}
}

type LoginRequest struct {
	Email    string `json:"email" form:"email" binding:"required,email"`
	Password string `json:"password" form:"password" binding:"required"`
}

type RegisterRequest struct {
	Username string `json:"username" form:"username" binding:"required"`
	Email    string `json:"email" form:"email" binding:"required,email"`
	Password string `json:"password" form:"password" binding:"required,min=8"`
	// Hatalı kayıt denemesinde şifre alanını doldurmak için eklenebilir (opsiyonel):
	// ConfirmPassword string `json:"confirm_password" form:"confirm_password"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

// Hata durumunda HTML partial döndürmek için yardımcı fonksiyon (Opsiyonel ama önerilir)
func renderAuthError(c *gin.Context, targetID string, errorMessage string) {
	// `_auth_error.html` gibi bir partial oluşturup onu render edebilirsiniz.
	// Şimdilik basit bir metin döndürelim veya mevcut error partial'ı kullanalım.
	// c.HTML(http.StatusBadRequest, "_auth_error.html", gin.H{"Target": targetID, "Error": errorMessage})

	// Veya mevcut global error partial'ını hedefleyerek kullanalım (eğer register-result'ı hedeflemek istemiyorsak)
	// Ama form içindeki hedef daha mantıklı. Şimdilik loglayıp JSON dönelim.
	log.Printf("Auth Error for target %s: %s", targetID, errorMessage)
	// İstemci tarafında (HTMX ile) bu JSON hatasını alıp #register-result içine yazdırabilirsiniz.
	// Veya sunucu tarafında doğrudan HTML döndürebilirsiniz.
	// Şimdilik JSON dönelim, HTMX bunu doğrudan #register-result'a yazamayacak ama loglarda görebiliriz.
	// En iyi yaklaşım, hatayı içeren bir HTML partial dönmek olurdu.
	c.JSON(http.StatusBadRequest, ErrorResponse{Error: errorMessage})
}

func (h *AuthHandler) LoginHandler(c *gin.Context) {
	var req LoginRequest
	// BindJSON yerine Bind kullanmak hem form hem JSON destekler
	if err := c.ShouldBind(&req); err != nil {
		renderAuthError(c, "#login-inner-box", "Geçersiz istek verisi.") // Hedef login formu olabilir
		return
	}

	user, err := h.repo.User.GetByEmail(req.Email)
	if err != nil {
		renderAuthError(c, "#login-inner-box", "Geçersiz e-posta veya şifre.")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		renderAuthError(c, "#login-inner-box", "Geçersiz e-posta veya şifre.")
		return
	}

	err = utils.GenerateToken(c, user.ID.String(), user.Username, h.cfg.JWTSecret, time.Hour*24)
	if err != nil {
		renderAuthError(c, "#login-inner-box", "Oturum başlatılamadı.")
		return
	}

	// --- Yönlendirme Değişikliği ---
	// c.Redirect yerine HX-Redirect header'ı kullan
	redirectURL := "/?success=" + url.QueryEscape("login_ok")
	c.Header("HX-Redirect", redirectURL)
	c.Status(http.StatusOK) // HTMX'in header'ı işlemesi için genellikle 2xx yanıt gerekir
}

func (h *AuthHandler) RegisterHandler(c *gin.Context) {
	var req RegisterRequest
	// Hem JSON hem form verisi için ShouldBind kullan
	if err := c.ShouldBind(&req); err != nil {
		// Hata mesajını #register-result içine render etmek en iyisi olurdu.
		// Örnek: c.HTML(http.StatusBadRequest, "partials/_form_error.html", gin.H{"Error": "Geçersiz istek: " + err.Error()})
		// Şimdilik JSON dönüyoruz. HTMX tarafında bu hata işlenmeli veya sunucu HTML dönmeli.
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Geçersiz istek verisi: " + err.Error()})
		return
	}

	// TODO: Şifre tekrarı kontrolü eklenebilir (req.ConfirmPassword ile)

	_, err := h.repo.User.GetByEmail(req.Email)
	if err == nil {
		// c.HTML(http.StatusConflict, "partials/_form_error.html", gin.H{"Error": "Bu e-posta adresi zaten kullanılıyor."})
		c.JSON(http.StatusConflict, ErrorResponse{Error: "Bu e-posta adresi zaten kullanılıyor."})
		return
	}

	// Kullanıcı adı kontrolü de eklenebilir
	// _, err = h.repo.User.GetByUsername(req.Username) ...

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Password hashing failed: %v", err)
		// c.HTML(http.StatusInternalServerError, "partials/_form_error.html", gin.H{"Error": "Hesap oluşturulamadı."})
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Hesap oluşturulamadı."})
		return
	}

	user := &models.User{
		ID:           uuid.New(),
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
	}

	if err := h.repo.User.Create(user); err != nil {
		log.Printf("User creation failed: %v", err)
		// c.HTML(http.StatusInternalServerError, "partials/_form_error.html", gin.H{"Error": "Hesap oluşturulamadı."})
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Hesap oluşturulamadı."})
		return
	}

	// Başarılı kayıt sonrası token oluştur
	err = utils.GenerateToken(c, user.ID.String(), user.Username, h.cfg.JWTSecret, time.Hour*24)
	if err != nil {
		log.Printf("Token generation failed after registration for user %s: %v", user.Email, err)
		// Kullanıcı oluşturuldu ama token verilemedi. Bu durumu kullanıcıya bildirmek önemli.
		// Belki login sayfasına yönlendirip bir mesaj gösterilebilir.
		// Şimdilik sadece hata dönelim.
		// c.HTML(http.StatusInternalServerError, "partials/_form_error.html", gin.H{"Error": "Hesap oluşturuldu ancak oturum başlatılamadı. Lütfen giriş yapmayı deneyin."})
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Hesap oluşturuldu ancak oturum başlatılamadı."})
		return
	}

	// --- Yönlendirme Değişikliği ---
	// c.Redirect yerine HX-Redirect header'ı kullan
	redirectURL := "/?success=" + url.QueryEscape("register_ok")
	c.Header("HX-Redirect", redirectURL)
	c.Status(http.StatusOK) // HTMX header'ı işlesin diye 2xx yanıt
}

func (h *AuthHandler) LogoutHandler(c *gin.Context) {
	err := utils.RevokeToken(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Logout failed"})
		return
	}
	// Navbar'daki JS yönlendirmesi çalıştığı için bu JSON yanıtı kalabilir.
	// Eğer JS yönlendirmesi olmasaydı, buradan da HX-Redirect header ile yönlendirme yapılabilirdi.
	c.JSON(http.StatusOK, gin.H{"message": "Logout successful"})
}
