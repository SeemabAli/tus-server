package debug

import (
	"net/http"
	"os"
	"os/exec"
)

func DebugFilesListHandler(w http.ResponseWriter, r *http.Request) {
	pwd := r.URL.Query().Get("location")

	// Validate that the location parameter is not empty
	if pwd == "" {
		http.Error(w, "Location parameter is missing", http.StatusBadRequest)
		return
	}

	// Ensure the directory exists and is valid
	if _, err := os.Stat(pwd); os.IsNotExist(err) {
		http.Error(w, "Directory does not exist", http.StatusBadRequest)
		return
	} else if err != nil {
		http.Error(w, "Failed to access directory: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Use exec.Command with proper argument passing
	cmd := exec.Command("ls", "-l", pwd) // `-l` flag for detailed listing, optional

	// Capture output to buffer
	output, err := cmd.Output()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Write output back to the client
	w.Write(output)
}
