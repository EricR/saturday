package config

import (
	"log"
	"os"
)

type Config struct {
	Logger   *log.Logger
	VarDecay float64
	ClaDecay float64
	Models   uint
}

func New() *Config {
	return &Config{
		Logger: log.New(os.Stderr, "", log.Ldate|log.Ltime),
	}
}
