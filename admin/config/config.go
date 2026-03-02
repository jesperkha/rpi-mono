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
	if err := cenv.Verify(); err != nil {
		log.Fatal(err)
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
