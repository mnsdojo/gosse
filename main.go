package main

import (
	"encoding/json"
	"net/http"
	"os"
	"sync"
	"time"
)

type Config struct {
	Port     string `json:"port"`
	Folder   string `json:"folder"`
	LogLevel string `json:"log_level"`
	Delay    int    `json:"delay"`
}

var (
	mu                sync.Mutex
	lastModifications time.Time
	// clients manage
	clients = make(map[http.ResponseWriter]bool)
	cfg     Config
)

func LoadConfig(filepath string) (Config, error) {
	var config Config
	data, err := os.ReadFile(filepath)
	if err != nil {
		return config, err
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return config, err
	}
	return config, nil
}

func main() {
}
