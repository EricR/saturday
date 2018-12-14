package config

import (
	"log"
	"os"
)

type Config struct {
	Logger     *log.Logger
	OutputPath string
	VarDecay   float64
	ClaDecay   float64
	Models     uint
	Verbose    bool
}

func New() *Config {
	return &Config{
		Logger: log.New(os.Stdout, "", log.Ldate|log.Ltime),
	}
}
