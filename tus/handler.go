package tus

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"time"

	"github.com/LinuxSploit/TusAce/middleware"
	"github.com/LinuxSploit/TusAce/transcoder"
	"github.com/LinuxSploit/TusAce/utils"
	"github.com/tus/tusd/v2/pkg/filelocker"
	"github.com/tus/tusd/v2/pkg/filestore"
	"github.com/tus/tusd/v2/pkg/handler"
)

var (
	ImageFileTypes map[string]bool = map[string]bool{
		"image/png":  true,
		"image/webp": true,
		"image/jpeg": true,
	}

	VideoFileTypes map[string]bool = map[string]bool{
		"video/mp4":        true,
		"video/webm":       true,
		"video/quicktime":  true,
		"video/avi":        true,
		"video/x-matroska": true,
	}
)

// setupTusHandler initializes the tusd handler for managing uploads
func SetupTusVideoHandler(basePath, storageDir string) (*handler.Handler, error) {
	store := filestore.New(storageDir)
	locker := filelocker.New(storageDir)
	composer := handler.NewStoreComposer()
	store.UseIn(composer)
	locker.UseIn(composer)

	tusdHandler, err := handler.NewHandler(handler.Config{
		BasePath:              basePath,
		StoreComposer:         composer,
		NotifyCompleteUploads: true,
		NotifyUploadProgress:  true,
		DisableDownload:       true,
		Cors: &handler.CorsConfig{
			Disable:          false,
			AllowOrigin:      regexp.MustCompile(".*"),
			AllowCredentials: false,
			AllowMethods:     "POST, HEAD, PATCH, OPTIONS, GET, DELETE",
			AllowHeaders:     "Authorization, x-email-address, Origin, X-Requested-With, X-Request-ID, X-HTTP-Method-Override, Content-Type, Upload-Length, Upload-Offset, Tus-Resumable, Upload-Metadata, Upload-Defer-Length, Upload-Concat, Upload-Incomplete, Upload-Complete, Upload-Draft-Interop-Version",
			MaxAge:           "86400",
			ExposeHeaders:    "Upload-Offset, Location, Upload-Length, Tus-Version, Tus-Resumable, Tus-Max-Size, Tus-Extension, Upload-Metadata, Upload-Defer-Length, Upload-Concat, Upload-Incomplete, Upload-Complete, Upload-Draft-Interop-Version",
		},
		PreUploadCreateCallback: func(hook handler.HookEvent) (handler.HTTPResponse, handler.FileInfoChanges, error) {
			// Extract session token from the headers
			sessionToken := hook.HTTPRequest.Header.Get("Authorization")
			email := hook.HTTPRequest.Header.Get("x-email-address")

			// Validate the session token (e.g., check it against your auth service or database)
			if sessionToken == "" || email == "" || middleware.ValidateSessionAndPerm(sessionToken, email) == 0 {
				return handler.HTTPResponse{
					StatusCode: http.StatusUnauthorized,
					Body:       "Invalid or missing session token",
				}, handler.FileInfoChanges{}, nil
			}

			fileType, ok := hook.Upload.MetaData["filetype"]
			if !ok {
				return handler.HTTPResponse{
					StatusCode: http.StatusUnauthorized,
					Body:       "missing filetype",
				}, handler.FileInfoChanges{}, nil
			}

			isAllowed, ok := VideoFileTypes[fileType]
			if !ok || !isAllowed {
				return handler.HTTPResponse{
					StatusCode: http.StatusUnauthorized,
					Body:       "Invalid filetype",
				}, handler.FileInfoChanges{}, nil
			}

			// If the session token is valid, you can add additional metadata to the FileInfo if needed
			newMeta := hook.Upload.MetaData
			newMeta["createdDate"] = time.Now().UTC().Format(time.RFC3339) // Add CreatedDate

			fileInfoChanges := handler.FileInfoChanges{
				MetaData: newMeta,
			}

			// No changes to the HTTP response in this case, just return a success
			return handler.HTTPResponse{}, fileInfoChanges, nil
		},
		PreFinishResponseCallback: func(hook handler.HookEvent) (handler.HTTPResponse, error) {

			fmt.Println("passing video into transcoding queue", hook.Upload.IsFinal)
			transcoder.TranscodeQueue <- hook.Upload.ID // Add upload to queue for transcoding
			return handler.HTTPResponse{}, nil
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create tusd handler: %w", err)
	}

	// Start a goroutine to handle completed uploads
	go func() {
		for event := range tusdHandler.CompleteUploads {
			log.Printf("> Upload %s completed\n", event.Upload.ID)
			// transcoder.TranscodeQueue <- event.Upload.ID // Add upload to queue for transcoding
		}

		//
		for event := range tusdHandler.CreatedUploads {
			log.Printf("> Upload %s Created\n", event.Upload.MetaData["filetype"])
		}
	}()

	return tusdHandler, nil
}

// setupTusHandler initializes the tusd handler for managing uploads
func SetupTusImageHandler(basePath, storageDir string) (*handler.Handler, error) {
	store := filestore.New(storageDir)
	locker := filelocker.New(storageDir)
	composer := handler.NewStoreComposer()
	store.UseIn(composer)
	locker.UseIn(composer)

	tusdHandler, err := handler.NewHandler(handler.Config{
		BasePath:              basePath,
		StoreComposer:         composer,
		NotifyCompleteUploads: true,
		NotifyUploadProgress:  true,
		Cors: &handler.CorsConfig{
			Disable:          false,
			AllowOrigin:      regexp.MustCompile(".*"),
			AllowCredentials: false,
			AllowMethods:     "POST, HEAD, PATCH, OPTIONS, GET, DELETE",
			AllowHeaders:     "Authorization, x-email-address, Origin, X-Requested-With, X-Request-ID, X-HTTP-Method-Override, Content-Type, Upload-Length, Upload-Offset, Tus-Resumable, Upload-Metadata, Upload-Defer-Length, Upload-Concat, Upload-Incomplete, Upload-Complete, Upload-Draft-Interop-Version",
			MaxAge:           "86400",
			ExposeHeaders:    "Upload-Offset, Location, Upload-Length, Tus-Version, Tus-Resumable, Tus-Max-Size, Tus-Extension, Upload-Metadata, Upload-Defer-Length, Upload-Concat, Upload-Incomplete, Upload-Complete, Upload-Draft-Interop-Version",
		},
		PreUploadCreateCallback: func(hook handler.HookEvent) (handler.HTTPResponse, handler.FileInfoChanges, error) {
			// Extract session token from the headers
			sessionToken := hook.HTTPRequest.Header.Get("Authorization")
			email := hook.HTTPRequest.Header.Get("x-email-address")

			fmt.Println(sessionToken, email)

			// Validate the session token (e.g., check it against your auth service or database)
			if sessionToken == "" || email == "" || middleware.ValidateSessionAndPerm(sessionToken, email) == 0 {
				return handler.HTTPResponse{
					StatusCode: http.StatusUnauthorized,
					Body:       "Invalid or missing session token",
				}, handler.FileInfoChanges{}, nil
			}

			fileType, ok := hook.Upload.MetaData["filetype"]
			if !ok {
				return handler.HTTPResponse{
					StatusCode: http.StatusUnauthorized,
					Body:       "missing filetype",
				}, handler.FileInfoChanges{}, nil
			}

			isAllowed, ok := ImageFileTypes[fileType]
			if !ok || !isAllowed {
				fmt.Println(fileType)
				return handler.HTTPResponse{
					StatusCode: http.StatusUnauthorized,
					Body:       "Invalid filetype",
				}, handler.FileInfoChanges{}, nil
			}

			// If the session token is valid, you can add additional metadata to the FileInfo if needed
			newMeta := hook.Upload.MetaData
			newMeta["createdDate"] = time.Now().UTC().Format(time.RFC3339) // Add CreatedDate

			fileInfoChanges := handler.FileInfoChanges{
				MetaData: newMeta,
			}

			// No changes to the HTTP response in this case, just return a success
			return handler.HTTPResponse{}, fileInfoChanges, nil
		},
		PreFinishResponseCallback: func(hook handler.HookEvent) (handler.HTTPResponse, error) {

			fmt.Println("this is prefinish hook callback", "/storage/tus/images/"+hook.Upload.ID, "/storage/tus/thumbnail/"+hook.Upload.ID+"-500w.webp")
			err := utils.ResizeAndConvertToWebP("/storage/tus/images/"+hook.Upload.ID, "/storage/tus/thumbnail/"+hook.Upload.ID+"-500w.webp", 500)
			if err != nil {
				fmt.Println("covert err: ", err)
			}
			return handler.HTTPResponse{}, nil
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create tusd handler: %w", err)
	}

	// Start a goroutine to handle completed uploads
	go func() {
		for event := range tusdHandler.CompleteUploads {
			log.Printf("> Upload %s completed\n", event.Upload.ID)
		}

		//
		for event := range tusdHandler.CreatedUploads {
			log.Printf("> Upload %s Created\n", event.Upload.MetaData["filetype"])
		}
	}()

	return tusdHandler, nil
}
