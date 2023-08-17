package main

import (
    "bytes"
    "fmt"
    "io"
    "log"
    "net/http"
    "os"
    "path/filepath"
    "strconv"
    "sync"
    "time"
)

type ChunkInfo struct {
    TotalChunks int
    Chunks     map[int][]byte
}

var chunkMap = make(map[string]ChunkInfo)
var wg sync.WaitGroup

func handleUpload(w http.ResponseWriter, r *http.Request) {
    filename := r.Header.Get("Filename")
    if filename == "" {
        http.Error(w, "Missing filename", http.StatusBadRequest)
        return
    }

    chunkNumber, _ := strconv.Atoi(r.Header.Get("Chunk-Number"))
    totalChunks, _ := strconv.Atoi(r.Header.Get("Total-Chunks"))

    body, err := io.ReadAll(r.Body)
    if err != nil {
        http.Error(w, "Error in reading", http.StatusInternalServerError)
        return
    }
    wg.Add(2)

    go func() {
        defer wg.Done()

        if _, exists := chunkMap[filename]; !exists {
            chunkMap[filename] = ChunkInfo{
                TotalChunks: totalChunks,
                Chunks:     make(map[int][]byte),
            }
        }
        chunkMap[filename].Chunks[chunkNumber] = body

        if len(chunkMap[filename].Chunks) == totalChunks {
            assemble(filename, totalChunks)
        }
    }()

    go func() {
        defer wg.Done()

        logLine := WebLog(filename, chunkNumber, string(body))
        LogFile(filename, chunkNumber, logLine)
    }()

    w.WriteHeader(http.StatusOK)
}

func handleUpdate(w http.ResponseWriter, r *http.Request) {
	filename := r.Header.Get("Filename")
	if filename == "" {
		http.Error(w, "Missing filename", http.StatusBadRequest)
		return
	}
	outputFilePath := filepath.Join("assemble", filename)
	existingContent, err := os.ReadFile(outputFilePath)
	if err != nil {
		http.Error(w, "Error reading existing assembled file content", http.StatusInternalServerError)
		return
	}

	updatedContent := append(existingContent, []byte("")...)
	err = os.WriteFile(outputFilePath, updatedContent, 0644)
	if err != nil {
		http.Error(w, "Error updating assembled file", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}


func handleDelete(w http.ResponseWriter, r *http.Request) {
	filename := r.Header.Get("Filename")
	if filename == "" {
		http.Error(w, "Missing filename", http.StatusBadRequest)
		return
	}
	FilePath := filepath.Join("assemble", filename)
	if err := os.Remove(FilePath); err != nil {
		http.Error(w, "Error deleting assembled file", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}


func LogFile(filename string, chunkNumber int, logLine string) {
    logFileName := fmt.Sprintf("%s_%d.log", filename, chunkNumber)
    logFilePath := filepath.Join("logs", logFileName)

    logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        log.Println("Error opening log file:", err)
        return
    }
    defer logFile.Close()

    _, err = logFile.WriteString(logLine + "\n")
    if err != nil {
        log.Println("Error writing to log file:", err)
    }
}

func assemble(filename string, totalChunks int) {
    chunks := make([][]byte, totalChunks)

    for chunkNumber := 0; chunkNumber < totalChunks; chunkNumber++ {
        chunk := chunkMap[filename].Chunks[chunkNumber]
        chunks[chunkNumber] = chunk
    }

    assembledContent := bytes.Join(chunks, []byte{})
    outputFilePath := filepath.Join("assemble", filename)

    err := os.WriteFile(outputFilePath, assembledContent, 0644)
    if err != nil {
        log.Println("Error writing assembled file:", err)
        return
    }

    log.Println("File", filename, "assembled and stored successfully")
}

func WebLog(filename string, chunkNumber int, content string) string {
    now := time.Now().Format("02/Jan/2006:15:04:05 -0700")
    return fmt.Sprintf("127.0.0.1 - - [%s] \"GET /%s Chunk: %d\" 200 %d", now, content, chunkNumber, len(content))
}

func main() {
    os.Mkdir("logs", os.ModePerm)
    os.Mkdir("assemble", os.ModePerm)
    certPath := "server.pem"
    keyPath := "server.key"
    
    http.HandleFunc("/upload", handleUpload)
    http.HandleFunc("/update", handleUpdate) 
	http.HandleFunc("/delete", handleDelete)

    
    err := http.ListenAndServeTLS(":8443", certPath, keyPath, nil)
    if err != nil {
        log.Fatal("ListenAndServe:", err)
    }
}
