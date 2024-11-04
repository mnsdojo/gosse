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
	clients           = make(map[chan []byte]bool)
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
	cfg, err := LoadConfig("config.json")
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}
	go watchFiles(cfg.Folder, cfg.Delay) // Start watching files
	fs := http.FileServer(http.Dir(cfg.Folder))
	http.Handle("/", fs)
	http.HandleFunc("/poll", handlePoll)
	log.Printf("Server listening on port %s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, nil); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}

func handlePoll(w http.ResponseWriter, r *http.Request) {
}

func NotifyClients(content []byte) {
	mu.Lock()
	defer mu.Unlock()
	for clientChan := range clients {
		clientChan <- content // Send the actual content to the channel
	}
}

func watchFiles(folder string, delay int) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	err = watcher.Add(folder)
	if err != nil {
		log.Fatal(err)
	}

	var lastChange time.Time
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				log.Printf("File modified: %s", event.Name)

				// Debounce logic
				lastChange = time.Now()
				time.AfterFunc(time.Duration(delay)*time.Millisecond, func() {
					if time.Since(lastChange) >= time.Duration(delay)*time.Millisecond {
						content, err := os.ReadFile(event.Name)
						if err == nil {
							NotifyClients(content) // Notify clients with new content
						}
					}
				})
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Println("Error:", err)
		}
	}
}
