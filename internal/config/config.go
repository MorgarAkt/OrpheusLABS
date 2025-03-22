package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

// TODO: Add project name into config
type Config struct {
	Port      string
	DBHost    string
	DBPort    string
	DBUser    string
	DBPass    string
	DBName    string
	DBSSLMode string
	JWTSecret string `mapstructure:"JWT_SECRET"`
}

func LoadConfig() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
		return nil, err
	}

	config := &Config{
		Port:      os.Getenv("PORT"),
		DBHost:    os.Getenv("DB_HOST"),
		DBPort:    os.Getenv("DB_PORT"),
		DBUser:    os.Getenv("DB_USER"),
		DBPass:    os.Getenv("DB_PASS"),
		DBName:    os.Getenv("DB_NAME"),
		DBSSLMode: os.Getenv("DB_SSLMODE"),
		JWTSecret: os.Getenv("JWT_SECRET"),
	}

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
