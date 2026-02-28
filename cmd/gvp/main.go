package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func startFolders() {
	if err := os.MkdirAll("./uploads", os.ModePerm); err != nil {
		log.Fatal("failed to create uploads dir: ", err)
	}
}

func main() {
	log.Println("Initing...")
	startFolders()

	multiplexer := http.NewServeMux()

	multiplexer.HandleFunc("/video/upload", videoUpload)

	server := &http.Server{
		Addr:    ":1313",
		Handler: multiplexer,
	}

	server.ListenAndServe()

}

type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

func writeJSON(w http.ResponseWriter, status int, success bool, message string, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Response{
		Success: success,
		Message: message,
		Data:    data,
	})
}

func videoUpload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, false, "Failed to parse form", nil)
		return
	}

	file, header, err := r.FormFile("video")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, false, "Video not found", nil)
		return
	}
	defer file.Close()

	buffer := make([]byte, 512)
	if _, err := file.Read(buffer); err != nil {
		writeJSON(w, http.StatusInternalServerError, false, "Failed to read file", nil)
		return
	}

	contentType := http.DetectContentType(buffer)
	if !strings.HasPrefix(contentType, "video/") {
		writeJSON(w, http.StatusBadRequest, false, "Only video files are allowed", nil)
		return
	}

	// seek back to the start before copying
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		writeJSON(w, http.StatusInternalServerError, false, "Failed to process file", nil)
		return
	}

	ext := filepath.Ext(header.Filename)
	tmp, err := os.CreateTemp("uploads/", "upload-*"+ext)

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, false, "Server error", nil)
		log.Println("Error creating temp file")
		return
	}
	defer tmp.Close()

	if _, err := io.Copy(tmp, file); err != nil {
		writeJSON(w, http.StatusInternalServerError, false, "failed to save file", nil)
		return
	}

	writeJSON(w, http.StatusOK, true, "video uploaded", nil)
}
