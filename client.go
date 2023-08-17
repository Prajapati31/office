package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

func server(client *http.Client, chunk []byte, filename string, chunkNumber, totalChunks int) error {
	url := "https://localhost:8443/upload"
	payload := strings.NewReader(string(chunk))

	req, err := http.NewRequest("POST", url, payload)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Filename", filename)
	req.Header.Set("Chunk-Number", strconv.Itoa(chunkNumber))
	req.Header.Set("Total-Chunks", strconv.Itoa(totalChunks))

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf(" non-OK status: %s", resp.Status)
	}

	return nil
}

func chunks(client *http.Client, filePath string, chunkSize int64, resultChan chan<- error) {
	file, err := os.Open(filePath)
	if err != nil {
		resultChan <- err
		return
	}
	defer file.Close()

	fileStat, _ := file.Stat()
	totalChunks := int(fileStat.Size() / chunkSize)
	if fileStat.Size()%chunkSize != 0 {
		totalChunks++
	}

	for chunkNumber := 0; ; chunkNumber++ {
		buffer := make([]byte, chunkSize)
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			resultChan <- err
			return
		}
		if n == 0 {
			break
		}

		err = server(client, buffer[:n], filepath.Base(filePath), chunkNumber, totalChunks)
		if err != nil {
			resultChan <- err
			return
		}
	}

	resultChan <- nil
}

func main() {
	chunkSize := int64(1024 * 1024 * 5)

	filePaths := []string{
		"sample.pdf",
		"large-file.json",
		"lottery.json",
		"vehicle.json",
		"samplejpg1.jpg",
		"Sampljepg2.jpg",
	}
	// certPath := "path/to/your/client.pem"
	// keyPath := "path/to/your/client.key"

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				Certificates: []tls.Certificate{},
				InsecureSkipVerify: true, 
			},
		},
	}

	var wg sync.WaitGroup
	for _, filePath := range filePaths {
		wg.Add(1)
		go func(filePath string) {
			defer wg.Done()

			resultChan := make(chan error)
			go chunks(client, filePath, chunkSize, resultChan)

			err := <-resultChan
			if err != nil {
				fmt.Println("Error in chunks:", err)
			} else {
				fmt.Println("File", filePath, "chunks sent successfully")
			}
		}(filePath)
	}

	wg.Wait()
	fmt.Scanln()
	
}