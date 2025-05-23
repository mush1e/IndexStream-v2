package config

import (
	"os"
	"strconv"
	"sync"
)

var (
	cfg  *Config
	once sync.Once
)

type Config struct {
	Port        int
	DataURL     string
	SearchDepth int
}

func load() *Config {
	cfg = &Config{
		Port:        8080,
		DataURL:     "./data/webpages",
		SearchDepth: 2,
	}

	if port, err := strconv.Atoi(os.Getenv("PORT")); err == nil {
		cfg.Port = port
	}

	if dataURL := os.Getenv("DATA_URL"); dataURL != "" {
		cfg.DataURL = dataURL
	}

	if depth, err := strconv.Atoi(os.Getenv("SEARCH_DEPTH")); err == nil {
		cfg.SearchDepth = depth
	}

	return cfg
}

func Get() *Config {
	// Runs Load only the first time Get is called other times this condition gets ignored
	once.Do(func() {
		cfg = load()
	})
	return cfg
}
