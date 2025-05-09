package config

import (
	"fmt"
	"log"
	"os"
	"strconv" // String'den int'e çevrim için eklendi

	"github.com/joho/godotenv"
)

type Config struct {
	Port        string
	DBHost      string
	DBPort      string
	DBUser      string
	DBPass      string
	DBName      string
	DBSSLMode   string
	JWTSecret   string `mapstructure:"JWT_SECRET"`
	RabbitMQURL string `mapstructure:"RABBITMQ_URL"`

	// Sayfalama için yeni config alanları
	DefaultPerPage int `mapstructure:"DEFAULT_PER_PAGE"`
	MinPerPage     int `mapstructure:"MIN_PER_PAGE"`
	MaxPerPage     int `mapstructure:"MAX_PER_PAGE"`
}

// Helper function to get environment variable or default
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	// Fallback "" ise ve değişken bulunamazsa uyarı verilebilir, ancak LoadConfig içinde yönetiliyor.
	return fallback
}

// Helper function to get environment variable as int or default
func getEnvAsInt(key string, fallback int) int {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return fallback
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		log.Printf("Warning: Invalid integer value for %s: %s. Using fallback: %d", key, valueStr, fallback)
		return fallback
	}
	return value
}

func LoadConfig() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		// .env dosyası bulunamazsa hata vermek yerine uyarı verip devam edebilir.
		// Eğer .env dosyası zorunlu ise burada fatal error verilebilir.
		log.Println("Warning: Error loading .env file, proceeding with environment variables if set:", err)
	}

	// Varsayılan değerleri burada tanımla
	defaultPerPageFallback := 4
	minPerPageFallback := 1
	maxPerPageFallback := 20

	config := &Config{
		Port:        getEnv("PORT", "8080"),
		DBHost:      getEnv("DB_HOST", ""), // Boş bırakılırsa validasyonda hata verecek
		DBPort:      getEnv("DB_PORT", ""),
		DBUser:      getEnv("DB_USER", ""),
		DBPass:      getEnv("DB_PASS", ""), // Şifre boş olabilir, validasyon size kalmış
		DBName:      getEnv("DB_NAME", ""),
		DBSSLMode:   getEnv("DB_SSLMODE", "disable"),
		JWTSecret:   getEnv("JWT_SECRET", ""),
		RabbitMQURL: getEnv("RABBITMQ_URL", ""),

		DefaultPerPage: getEnvAsInt("DEFAULT_PER_PAGE", defaultPerPageFallback),
		MinPerPage:     getEnvAsInt("MIN_PER_PAGE", minPerPageFallback),
		MaxPerPage:     getEnvAsInt("MAX_PER_PAGE", maxPerPageFallback),
	}

	// Temel validasyonlar
	if config.DBHost == "" {
		return nil, fmt.Errorf("missing required configuration variable: DB_HOST")
	}
	if config.DBPort == "" {
		return nil, fmt.Errorf("missing required configuration variable: DB_PORT")
	}
	if config.DBUser == "" {
		return nil, fmt.Errorf("missing required configuration variable: DB_USER")
	}
	// DB_PASS için validasyon isteğe bağlı
	if config.DBName == "" {
		return nil, fmt.Errorf("missing required configuration variable: DB_NAME")
	}
	if config.JWTSecret == "" {
		return nil, fmt.Errorf("missing required configuration variable: JWT_SECRET")
	}
	if config.RabbitMQURL == "" {
		return nil, fmt.Errorf("missing required configuration variable: RABBITMQ_URL")
	}

	// Sayfalama değerleri için ek validasyonlar
	if config.MinPerPage <= 0 {
		log.Printf("Warning: MIN_PER_PAGE (%d) is invalid, setting to 1.", config.MinPerPage)
		config.MinPerPage = 1
	}
	if config.DefaultPerPage < config.MinPerPage {
		log.Printf("Warning: DEFAULT_PER_PAGE (%d) is less than MIN_PER_PAGE (%d). Setting DEFAULT_PER_PAGE to MIN_PER_PAGE.", config.DefaultPerPage, config.MinPerPage)
		config.DefaultPerPage = config.MinPerPage
	}
	if config.MaxPerPage < config.DefaultPerPage {
		log.Printf("Warning: MAX_PER_PAGE (%d) is less than DEFAULT_PER_PAGE (%d). Setting MAX_PER_PAGE to DEFAULT_PER_PAGE.", config.MaxPerPage, config.DefaultPerPage)
		config.MaxPerPage = config.DefaultPerPage
	}
	if config.MaxPerPage == 0 && maxPerPageFallback > 0 { // Eğer MaxPerPage 0 gelirse ama fallback varsa onu kullan
		log.Printf("Warning: MAX_PER_PAGE evaluated to 0, using fallback %d", maxPerPageFallback)
		config.MaxPerPage = maxPerPageFallback
	}

	log.Println("Configuration loaded successfully.")
	return config, nil
}

func (cfg *Config) DBConnectionStringWName() string {
	return fmt.Sprintf("host=%s port=%s user=%s dbname=%s password=%s sslmode=%s",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBName, cfg.DBPass, cfg.DBSSLMode)
}

func (cfg *Config) DBConnectionStringWOName() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s sslmode=%s",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPass, cfg.DBSSLMode)
}
