package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

var (
	// Version is the main version number
	Version = "0.0.0"

	// VersionPrerelease is a prerelease marker
	VersionPrerelease = ""

	// Buildstamp is the timestamp the binary was built, it should be set at buildtime with ldflags
	Buildstamp = "No BuildStamp Provided"

	// Githash is the git sha of the built binary, it should be set at buildtime with ldflags
	Githash = "No Git Commit Provided"

	debug   = flag.Bool("debug", false, "Display debug logging.")
	version = flag.Bool("version", false, "Display version information and exit.")
)

type server struct {
	router *mux.Router
}

func vers() {
	fmt.Printf("Test-API Version: %s%s\n", Version, VersionPrerelease)
	os.Exit(0)
}

func main() {
	flag.Parse()
	if *version {
		vers()
	}

	log.Infof("Starting Test-API version %s%s", Version, VersionPrerelease)

	if *debug {
		log.SetLevel(log.DebugLevel)
		log.Debug("debug logging enabled")
	}

	s := server{
		router: mux.NewRouter(),
	}

	s.routes()

	h := handlers.RecoveryHandler(handlers.PrintRecoveryStack(true))(handlers.CombinedLoggingHandler(os.Stdout, s.router))
	srv := &http.Server{
		Handler:      h,
		Addr:         ":8000",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())
}

func (s *server) routes() {
	s.router.Handle("/metrics", promhttp.Handler())

	api := s.router.PathPrefix("/v1/test").Subrouter()
	api.HandleFunc("/env", s.envHandler)
	api.HandleFunc("/metadata/task", s.metadataTaskHandler)
	api.HandleFunc("/metadata/stats", s.metadataStatsHandler)
	api.HandleFunc("/metadata/task/stats", s.metadataTaskStatsHandler)
	api.HandleFunc("/mirror", s.mirrorHandler)
	api.HandleFunc("/panic", s.panicHandler)
	api.HandleFunc("/ping", s.pingPongHandler)
	api.HandleFunc("/routes", s.routesHandler)
	api.HandleFunc("/status", s.statusHandler)
	api.HandleFunc("/version", s.versionHandler)
}

func (s *server) routesHandler(w http.ResponseWriter, req *http.Request) {
	log.Debug("routesHandler")

	pathRoutes := []string{}
	if err := s.router.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		pathTemplate, err := route.GetPathTemplate()
		if err != nil {
			return err
		}

		log.Debugf("path template: %s", pathTemplate)

		methods, err := route.GetMethods()
		if err != nil {
			methods = []string{"ANY"}
		}

		for _, m := range methods {
			p := fmt.Sprintf("%s %s", m, pathTemplate)
			log.Debugf("appending route %s", p)
			pathRoutes = append(pathRoutes, p)
		}

		return nil
	}); err != nil {
		log.Errorf("error walking routes: %s", err)
	}

	log.Debugf("router paths %+v", pathRoutes)

	out, err := json.Marshal(pathRoutes)
	if err != nil {
		log.Errorf("error marshaling routes: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(out)
	if err != nil {
		log.Errorf("error writing: %s", err)
	}
}

func (s *server) statusHandler(w http.ResponseWriter, req *http.Request) {
	q := req.URL.Query()
	log.Debugf("got querys %+v", q)
	if s, ok := q["code"]; ok && len(s) > 0 {
		i, err := strconv.Atoi(s[0])
		if err != nil {
			log.Errorf("error converting %s to status code int: %s", s[0], err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if i < 100 || i > 599 {
			log.Errorf("invalid HTTP status code int: %d", i)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf("invalid HTTP status code %d\n", i)))
			return
		}

		w.WriteHeader(i)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *server) mirrorHandler(w http.ResponseWriter, req *http.Request) {
	log.Debug("mirrorHandler")

	out, err := httputil.DumpRequest(req, true)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Errorf("failed dumping request: %s", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	if _, err = w.Write(out); err != nil {
		log.Errorf("error writing: %s", err)
	}
}

func (s *server) panicHandler(w http.ResponseWriter, req *http.Request) {
	log.Debug("panicHandler")

	log.Error("panicing")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	panic("boom")
}

func (s *server) pingPongHandler(w http.ResponseWriter, req *http.Request) {
	log.Debug("pingPongHandler")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := io.WriteString(w, "pong"); err != nil {
		log.Errorf("error writing: %s", err)
	}
}

func (s *server) envHandler(w http.ResponseWriter, req *http.Request) {
	log.Debug("envHandler")

	out, err := json.Marshal(os.Environ())
	if err != nil {
		log.Errorf("error marshaling environment: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(out)
	if err != nil {
		log.Errorf("error writing: %s", err)
	}
}

func (s *server) versionHandler(w http.ResponseWriter, req *http.Request) {
	log.Debug("versionHandler")

	version := struct {
		Version    string
		PreRelease string
		Buildstamp string
		Githash    string
	}{Version, VersionPrerelease, Buildstamp, Githash}

	out, err := json.Marshal(version)
	if err != nil {
		log.Errorf("error marshaling version: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(out)
	if err != nil {
		log.Errorf("error writing: %s", err)
	}
}

func (s *server) metadataTaskHandler(w http.ResponseWriter, req *http.Request) {
	log.Debug("metadataTaskHandler")

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
		log.Errorf("error getting metadata: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, err = io.Copy(w, out.Body)
	if err != nil {
		log.Errorf("error writing: %s", err)
	}
}

func (s *server) metadataStatsHandler(w http.ResponseWriter, req *http.Request) {
	log.Debug("metadataStatsHandler")

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
		log.Errorf("error getting metadata: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, err = io.Copy(w, out.Body)
	if err != nil {
		log.Errorf("error writing: %s", err)
	}
}
func (s *server) metadataTaskStatsHandler(w http.ResponseWriter, req *http.Request) {
	log.Debug("metadataTaskStatsHandler")

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
		log.Errorf("error getting metadata: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, err = io.Copy(w, out.Body)
	if err != nil {
		log.Errorf("error writing: %s", err)
	}
}
