package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
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

	CommandLineArgs = []string{}
)

type server struct {
	router     *mux.Router
	volEnable  bool
	volPath    string
	listenPort uint16
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

	CommandLineArgs = flag.Args()
	log.Debugf("args: %+v", CommandLineArgs)

	s := server{
		router: mux.NewRouter(),
	}

	s.configure()
	s.routes()

	h := handlers.RecoveryHandler(handlers.PrintRecoveryStack(true))(handlers.CombinedLoggingHandler(os.Stdout, s.router))
	srv := &http.Server{
		Handler:      h,
		Addr:         fmt.Sprintf(":%s", strconv.Itoa(int(s.listenPort))),
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())
}

func (s *server) configure() {
	s.listenPort = 8080
	if listenPort, ok := os.LookupEnv("LISTEN_PORT"); ok {
		lp, err := strconv.ParseUint(listenPort, 10, 16)
		if err != nil {
			log.Errorf("LISTEN_PORT value '%s' is not a valid integer: %s", listenPort, err)
		} else {
			s.listenPort = uint16(lp)
		}
	}

	if volEnable, ok := os.LookupEnv("VOLUME_ENABLE"); ok {
		ve, err := strconv.ParseBool(volEnable)
		if err != nil {
			log.Errorf("VOLUME_ENABLE value '%s' is not a valid boolean: %s", volEnable, err)
		} else {
			s.volEnable = ve
		}
	}

	s.volPath = "uploads"
	if volPath, ok := os.LookupEnv("VOLUME_PATH"); ok {
		s.volPath = volPath
	}
}

func (s *server) routes() {
	s.router.Handle("/metrics", promhttp.Handler())

	api := s.router.PathPrefix("/v1/test").Subrouter()
	api.HandleFunc("/args", s.argHandler)
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

	// if volumes are enabled, create upload and liste endpoints
	if s.volEnable {
		api.HandleFunc("/upload", s.uploadFileHandler)
		api.HandleFunc("/list", s.listFileHandler)
	}
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

func (s server) argHandler(w http.ResponseWriter, req *http.Request) {
	log.Debug("argHandler")

	out, err := json.Marshal(struct {
		Args []string
	}{
		Args: CommandLineArgs,
	})
	if err != nil {
		log.Errorf("error marshaling commandlineargs: %s", err)
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

func (s *server) uploadFileHandler(w http.ResponseWriter, req *http.Request) {
	log.Debug("uploadFileHandler")

	err := req.ParseMultipartForm(10 << 20)
	if err != nil {
		msg := fmt.Sprintf("error parsing form: %s", err)
		log.Errorf(msg)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(msg))
		return
	}

	file, handler, err := req.FormFile("file")
	if err != nil {
		msg := fmt.Sprintf("error retrieving file: %s", err)
		log.Errorf(msg)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(msg))
		return
	}
	defer file.Close()

	log.Debugf("Uploaded File: %+v, File Size: %+v, MIME Header: %+v", handler.Filename, handler.Size, handler.Header)

	tempFile, err := ioutil.TempFile(s.volPath, "*-"+handler.Filename)
	if err != nil {
		msg := fmt.Sprintf("error writing file: %s", err)
		log.Errorf(msg)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(msg))
		return
	}
	defer tempFile.Close()

	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		msg := fmt.Sprintf("error reading bytes from file: %s", err)
		log.Errorf(msg)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(msg))
		return
	}
	tempFile.Write(fileBytes)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

func (s *server) listFileHandler(w http.ResponseWriter, req *http.Request) {
	log.Debug("listFileHandler")

	files, err := ioutil.ReadDir(s.volPath)
	if err != nil {
		msg := fmt.Sprintf("error reading directory contents: %s", err)
		log.Errorf(msg)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(msg))
		return
	}

	type FileInfo struct {
		Name    string
		Size    string
		ModTime string
	}

	flist := []FileInfo{}
	for _, f := range files {
		flist = append(flist, FileInfo{
			Name:    f.Name(),
			Size:    humanizeByteSize(f.Size()),
			ModTime: f.ModTime().In(time.Local).Format(time.ANSIC),
		})
	}

	out, err := json.Marshal(struct {
		Files []FileInfo
	}{
		Files: flist,
	})
	if err != nil {
		log.Errorf("error marshaling file list: %s", err)
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

func humanizeByteSize(b int64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "kMGTPE"[exp])
}
