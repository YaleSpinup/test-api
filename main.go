package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

func main() {
	log.Println("starting test-api")

	http.HandleFunc("/ping", pingPongHandler)
	http.HandleFunc("/env", envHandler)

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
