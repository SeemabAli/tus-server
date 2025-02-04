package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path"

	"github.com/LinuxSploit/TusAce/debug"
	"github.com/LinuxSploit/TusAce/middleware"
	"github.com/LinuxSploit/TusAce/transcoder"
	"github.com/LinuxSploit/TusAce/tus"
)

func init() {
	err := os.MkdirAll("/storage/tus/thumbnail/", os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("init loaded successfully!")
}

func main() {
	log.Println("Service started")
	// Create TUS video Upload handler, init basePath, storageDir
	videoHandler, err := tus.SetupTusVideoHandler("https://tus-server-production.up.railway.app/video/", "/storage/tus/videos")
	if err != nil {
		log.Fatalf("Unable to create photo handler: %v", err)
	}

	// Create TUS video Upload handler, init basePath, storageDir
	imageHandler, err := tus.SetupTusImageHandler("https://tus-server-production.up.railway.app/image/", "/storage/tus/images")
	if err != nil {
		log.Fatalf("Unable to create photo handler: %v", err)
	}

	mux := http.NewServeMux()

	// Register TUS video Upload handler to /upload/ route
	mux.Handle("/video/", http.StripPrefix("/video/", videoHandler))
	// Serve HLS video streams with CORS middleware
	videoFileServer := http.StripPrefix("/hls/", NoDirListingFileServer(http.Dir("/storage/tus/hls/")))
	mux.Handle("/hls/", middleware.CORSMiddleware(videoFileServer))
	//thumbnail server
	thumbnailFileServer := http.StripPrefix("/thumbnail/", NoDirListingFileServer(http.Dir("/storage/tus/thumbnail/")))
	mux.Handle("/thumbnail/", middleware.CORSMiddleware(thumbnailFileServer))

	// Register TUS image Upload handler to /image-upload/ route
	mux.Handle("/image/", http.StripPrefix("/image/", imageHandler))

	mux.HandleFunc("/media", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		fmt.Fprintln(w, id)
	})

	// Serve the home page with demo upload page
	mux.HandleFunc("/video-demo", func(w http.ResponseWriter, r *http.Request) {
		tmpl := template.Must(template.ParseFiles("./video-demo.html"))
		tmpl.Execute(w, nil)
	})

	mux.HandleFunc("/image-demo", func(w http.ResponseWriter, r *http.Request) {
		tmpl := template.Must(template.ParseFiles("./image-demo.html"))
		tmpl.Execute(w, nil)
	})

	// mux.Handle("/geoip", middleware.CORSMiddleware(http.HandlerFunc(geoip.GeoIP)))

	mux.HandleFunc("/ls", debug.DebugFilesListHandler)

	// Start the transcode worker
	transcoder.StartTranscodeWorker("/storage/tus/videos/", "/storage/tus/hls/")

	// Start the HTTP server
	if err := http.ListenAndServe("0.0.0.0:8080", mux); err != nil {
		log.Fatalf("Unable to start server: %v", err)
	}
}

// NoDirListingFileServer wraps the http.FileServer to disable directory listings
func NoDirListingFileServer(root http.FileSystem) http.Handler {
	fs := http.FileServer(root)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the absolute path to prevent directory traversal
		upath := path.Clean(r.URL.Path)

		// Open the file
		f, err := root.Open(upath)
		if err != nil {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		defer f.Close()

		// Get file information
		info, err := f.Stat()
		if err != nil {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		// If it's a directory, deny access
		if info.IsDir() {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		// Serve the file
		fs.ServeHTTP(w, r)
	})
}
