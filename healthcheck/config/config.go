package config

import (
	"log"
	"os"

	"github.com/echo-webkom/cenv"
)

type Config struct {
	Port string
}

func Load() *Config {
	if err := cenv.Verify(); err != nil {
		log.Fatal(err)
	}

	return &Config{
		Port: toGoPort(os.Getenv("PORT")),
	}
}

func toGoPort(port string) string {
	if port[0] != ':' {
		return ":" + port
	}
	return port
}
