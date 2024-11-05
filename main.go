package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/fsnotify/fsnotify"
)

type Config struct {
	Folder string `json:"folder"`
	Port   int    `json:"port"`
	Delay  int    `json:"delay"`
}

var config Config

func LoadConfig(filepath string) (Config, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return config, err
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return config, err
	}
	return config, nil
}

// handling file watch...
func watchFiles(folder string, delay int) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err) // Exit if we fail to create the watcher
	}
	defer watcher.Close() // Ensure the watcher is closed when the function exits

	// Add the specified folder to the watcher
	err = watcher.Add(folder)
	if err != nil {
		log.Fatal(err) // Exit if we fail to add the folder to watch
	}

	var changeTimer *time.Timer // Declare a pointer to a Timer that will handle change delays

	for {
		select {
		case event, ok := <-watcher.Events: // Wait for events from the watcher
			if !ok {
				return // Exit if the channel is closed
			}
			// Check if the event is a write operation (file modification)
			if event.Op&fsnotify.Write == fsnotify.Write {
				log.Printf("Modified file: %s\n", event.Name) // Log the modified file name

				if changeTimer != nil {
					changeTimer.Stop() // Stop the existing timer if it's running
				}
				// Start a new timer with the specified delay
				changeTimer = time.AfterFunc(time.Duration(delay)*time.Millisecond, func() {
					log.Println("Reloading files...")
					// Here you could implement the logic to reload the files
				})
			}
		case err, ok := <-watcher.Errors: // Wait for errors from the watcher
			if !ok {
				return // Exit if the channel is closed
			}
			log.Println("Error:", err)
		}
	}
}

func handlePoll(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
}

func main() {
	cfg, err := LoadConfig("config.json")
	if err != nil {
		log.Fatalf("error loading config : %v", err)
	}
	go watchFiles(cfg.Folder, cfg.Delay)

	fs := http.FileServer(http.Dir(cfg.Folder))
	http.Handle("/", fs)

	http.HandleFunc("/poll", handlePoll)

	log.Printf("server listening on port: %d", cfg.Port)

	if err := http.ListenAndServe(fmt.Sprintf(":%d", cfg.Port), nil); err != nil {
		log.Fatalf("error starting server: %v", err)
	}
}
