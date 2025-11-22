package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port        string
	SecretKey   string
	DatabaseURL string
}

func LoadConfig() Config {
	err := godotenv.Load()
	if err != nil {
		log.Println(".env file not found")
	}

	port := os.Getenv("PORT")
	secretKey := os.Getenv("SECRET_KEY")
	databaseURL := os.Getenv("DATABASE_URL")

	return Config{
		Port:        port,
		SecretKey:   secretKey,
		DatabaseURL: databaseURL,
	}
}
