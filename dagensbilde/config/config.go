package config

import (
	"log"
	"os"

	"github.com/echo-webkom/cenv"
)

type Config struct {
	Port         string
	PasswordHash string
	DBPath       string
	ImageDir     string
}

func Load() *Config {
	if err := cenv.Verify(); err != nil {
		log.Fatal(err)
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "/data/app.db"
	}

	imageDir := os.Getenv("IMAGE_DIR")
	if imageDir == "" {
		imageDir = "/data/images"
	}

	return &Config{
		Port:         toGoPort(os.Getenv("PORT")),
		PasswordHash: os.Getenv("PASSWORD_HASH"),
		DBPath:       dbPath,
		ImageDir:     imageDir,
	}
}

func toGoPort(port string) string {
	if port[0] != ':' {
		return ":" + port
	}
	return port
}
