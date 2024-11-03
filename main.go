package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
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

func setupServer() error {
	var err error
	cfg, err = LoadConfig("config.json")
	if err != nil {
		return err
	}
	fs := http.FileServer(http.Dir(cfg.Folder))
	http.Handle("/", fs)
	http.HandleFunc("/poll", handlePoll)
	log.Printf("server listening on port :%s ", cfg.Port)
	return http.ListenAndServe(":"+cfg.Port, nil)
}

func handlePoll(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	mu.Lock()
	clients[w] = true
	mu.Unlock()

	defer func() {
		mu.Lock()
		delete(clients, w)
		mu.Unlock()
	}()

	// keep conneection active/alive
	//
}

func watchFiles(folder string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("error setting up file watcher :%v", err)
	}
	defer watcher.Close()

	if err := watcher.Add(folder); err != nil {
		log.Fatalf("Error watching folder: %v", err)
	}
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				log.Printf("File modified: %s", event.Name)
				lastModifications = time.Now()

			}
		case err := <-watcher.Errors:
			log.Printf("watcher error :%v", err)
		}
	}
}

func main() {
	if err := setupServer(); err != nil {
		log.Fatalf("Error setting up server :%v", err)
	}
}
