package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

type RequestStats struct {
	mu          sync.Mutex
	uniqueIDs   map[int]struct{}
	logFile     *os.File
	requestChan chan RequestRecord
}

type RequestRecord struct {
	ID       int
	Endpoint string
}

func NewRequestStats() *RequestStats {
	logFile, err := os.OpenFile("unique_requests.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	return &RequestStats{
		uniqueIDs:   make(map[int]struct{}),
		logFile:     logFile,
		requestChan: make(chan RequestRecord, 10000),
	}
}

func (rs *RequestStats) Close() {
	rs.logFile.Close()
}

func (rs *RequestStats) StartLogging() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			rs.logUniqueRequestCount()
		case record := <-rs.requestChan:
			rs.processRequest(record)
		}
	}
}

func (rs *RequestStats) logUniqueRequestCount() {
	rs.mu.Lock()
	count := len(rs.uniqueIDs)
	rs.uniqueIDs = make(map[int]struct{}) // Reset for next interval
	rs.mu.Unlock()

	logMessage := fmt.Sprintf("%s - Unique requests in the last minute: %d\n", time.Now().Format(time.RFC3339), count)
	_, err := rs.logFile.WriteString(logMessage)
	if err != nil {
		log.Printf("Failed to write log: %v", err)
	}
}

func (rs *RequestStats) processRequest(record RequestRecord) {
	rs.mu.Lock()
	if _, exists := rs.uniqueIDs[record.ID]; !exists {
		rs.uniqueIDs[record.ID] = struct{}{}
		if record.Endpoint != "" {
			rs.sendHTTPPost(record.Endpoint, len(rs.uniqueIDs))
		}
	}
	rs.mu.Unlock()
}

func (rs *RequestStats) sendHTTPPost(endpoint string, uniqueCount int) {
	payload := map[string]interface{}{
		"unique_count": uniqueCount,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Failed to marshal payload: %v", err)
		return
	}

	resp, err := http.Post(endpoint, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("Failed to send POST request: %v", err)
		return
	}
	defer resp.Body.Close()
	log.Printf("POST request to %s returned status: %s", endpoint, resp.Status)
}

func main() {
	stats := NewRequestStats()
	defer stats.Close()

	go stats.StartLogging()

	http.HandleFunc("/api/verve/accept", func(w http.ResponseWriter, r *http.Request) {
		idParam := r.URL.Query().Get("id")
		endpoint := r.URL.Query().Get("endpoint")

		if idParam == "" {
			http.Error(w, "Missing required parameter: id", http.StatusBadRequest)
			return
		}

		id, err := strconv.Atoi(idParam)
		if err != nil {
			http.Error(w, "Invalid id parameter", http.StatusBadRequest)
			return
		}

		stats.requestChan <- RequestRecord{ID: id, Endpoint: endpoint}
		fmt.Fprintln(w, "ok")
	})

	log.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
