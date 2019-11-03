package main

import (
	"regexp"
	"path"
	"time"
	"strings"
	"io/ioutil"
	"log"
	"os"
	"fmt"
	"net/http"
	"flag"
)

var (
	dir = flag.String("dir", getwd(), "directory")
	port = flag.Int("port", 3000, "the port that the server will listen to")
	host = flag.String("host", "127.0.0.1", "the ip that the server will listen to")
	loggingEnabled = flag.Bool("verbose", false, "enable logging")
)

func getwd() string {
	d, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	return d
}

func init() {
	flag.Parse()
}

func main() {
	output, err := os.Open(os.DevNull)
	if err != nil {
		log.Fatal(err)
	}
	if *loggingEnabled {
		_ = output.Close()
		output = os.Stdout
	}
	logger := log.New(output, "http: ", log.LstdFlags)

	logging := func (handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger.Printf("%s %s\n", r.Method, r.URL)
			handler.ServeHTTP(w, r)
		})
	}


	dumbGit := func (handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			pattern := regexp.MustCompile(`^/.*/info/refs$`)
			if !pattern.MatchString(path.Clean(r.URL.Path)) {
				handler.ServeHTTP(w, r)
				return
			}
			repodir := path.Join(*dir, r.URL.Path, "..", "..")
			resp := make([]string, 0)

			files, err := ioutil.ReadDir(path.Join(repodir, "refs/heads"))
			if err != nil {
				http.NotFound(w, r)
				return
			}

			for _, f := range files {
				fb, err := ioutil.ReadFile(fmt.Sprintf(path.Join(repodir, "refs/heads/%s"), f.Name()))
				if err != nil {
					http.NotFound(w, r)
					return
				}
				resp = append(resp, fmt.Sprintf("%s\trefs/heads/%s", strings.TrimSuffix(string(fb), "\n"), f.Name()))
			}

			files, err = ioutil.ReadDir(path.Join(repodir, "refs/tags"))
			if err != nil {
				logger.Fatal(err)
			}

			for _, f := range files {
				fb, err := ioutil.ReadFile(fmt.Sprintf(path.Join(repodir, "refs/heads/%s"), f.Name()))
				if err != nil {
					http.NotFound(w, r)
					return
				}
				resp = append(resp, fmt.Sprintf("%s\trefs/tags/%s", strings.TrimSuffix(string(fb), "\n"), f.Name()))
				resp = append(resp, fmt.Sprintf("%s\trefs/tags/%s^{}", strings.TrimSuffix(string(fb), "\n"), f.Name()))
			}

			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprintln(w, strings.Join(resp, "\n"))
		})
	}

	fs := http.FileServer(http.Dir(*dir))

	server := &http.Server{
		Addr: fmt.Sprintf("%s:%d", *host, *port),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
		Handler: logging(dumbGit(fs)),
		ErrorLog: logger,
	}

	logger.Printf("Listening on %s\n", server.Addr)

	if err = server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
