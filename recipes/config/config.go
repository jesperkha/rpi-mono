package config

import (
	"log"
	"os"

	"github.com/echo-webkom/cenv"
)

type Config struct {
	Port         string
	PasswordHash string
}

func Load() *Config {
	// Try loading .env file first (local dev). If it doesn't exist,
	// fall back to verifying env vars are already set (Docker/production).
	if _, err := os.Stat(".env"); err == nil {
		if err := cenv.Load(); err != nil {
			log.Fatal(err)
		}
	} else {
		if err := cenv.Verify(); err != nil {
			log.Fatal(err)
		}
	}

	return &Config{
		Port:         toGoPort(os.Getenv("PORT")),
		PasswordHash: os.Getenv("PASSWORD_HASH"),
	}
}

func toGoPort(port string) string {
	if port[0] != ':' {
		return ":" + port
	}
	return port
}
