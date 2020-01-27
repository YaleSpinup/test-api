package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

func main() {
	log.Println("starting test-api")

	http.HandleFunc("/ping", pingPongHandler)
	http.HandleFunc("/env", envHandler)
	http.HandleFunc("/metadata/task", metadataTaskHandler)
	http.HandleFunc("/metadata/stats", metadataStatsHandler)
	http.HandleFunc("/metadata/task/stats", metadataTaskStatsHandler)

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func pingPongHandler(w http.ResponseWriter, req *http.Request) {
	_, err := io.WriteString(w, "pong\n")
	if err != nil {
		log.Printf("error writing: %s", err)
	}
}

func envHandler(w http.ResponseWriter, req *http.Request) {
	e := os.Environ()
	_, err := io.WriteString(w, strings.Join(e, "\n"))
	if err != nil {
		log.Printf("error writing: %s", err)
	}
}

func metadataTaskHandler(w http.ResponseWriter, req *http.Request) {
	metadataUrl := os.Getenv("ECS_CONTAINER_METADATA_URI")
	if metadataUrl == "" {
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte("metadata not found"))
		return
	}

	client := &http.Client{
		Timeout: time.Second * 3,
	}
	out, err := client.Get(metadataUrl + "/task")
	if err != nil {
		log.Printf("error getting metadata: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_, err = io.Copy(w, out.Body)
	if err != nil {
		log.Printf("error writing: %s", err)
	}
}

func metadataStatsHandler(w http.ResponseWriter, req *http.Request) {
	metadataUrl := os.Getenv("ECS_CONTAINER_METADATA_URI")
	if metadataUrl == "" {
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte("metadata not found"))
		return
	}

	client := &http.Client{
		Timeout: time.Second * 3,
	}
	out, err := client.Get(metadataUrl + "/stats")
	if err != nil {
		log.Printf("error getting metadata: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_, err = io.Copy(w, out.Body)
	if err != nil {
		log.Printf("error writing: %s", err)
	}
}
func metadataTaskStatsHandler(w http.ResponseWriter, req *http.Request) {
	metadataUrl := os.Getenv("ECS_CONTAINER_METADATA_URI")
	if metadataUrl == "" {
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte("metadata not found"))
		return
	}

	client := &http.Client{
		Timeout: time.Second * 3,
	}
	out, err := client.Get(metadataUrl + "/task/stats")
	if err != nil {
		log.Printf("error getting metadata: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_, err = io.Copy(w, out.Body)
	if err != nil {
		log.Printf("error writing: %s", err)
	}
}
