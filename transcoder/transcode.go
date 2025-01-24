package transcoder

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/LinuxSploit/TusAce/utils"
)

// Queue for storing upload IDs
var TranscodeQueue = make(chan string, 100)

// TranscodePipeline performs video transcoding and manages temporary files
func TranscodePipeline(id string, input_path, output_path string) error {

	scriptPath := "./run.sh"
	if err := RunFFmpegScript(scriptPath, input_path+id, output_path+id, "master", 4, 25, 100, "veryfast", "640x360", "1280x720", "1920x1080"); err != nil {
		return fmt.Errorf("transcoding error: %w", err)
	}

	if err := os.Remove(input_path + id); err != nil {
		return fmt.Errorf("failed to remove temporary uploads: %w", err)
	}

	if err := os.Remove(input_path + id + ".info"); err != nil {
		return fmt.Errorf("failed to remove temporary uploads: %w", err)
	}

	return nil
}

// RunFFmpegScript executes an external Bash script with the specified parameters, thumbnailFrame is percentage from 0 to 100
func RunFFmpegScript(scriptPath, videoIn, out_dir, videoOut string, hlsTime, fps, gopSize int, presetP, vSize3, vSize5, vSize6 string) error {
	cmd := exec.Command("/bin/bash", scriptPath,
		videoIn,
		videoOut,
		fmt.Sprintf("%d", hlsTime),
		fmt.Sprintf("%d", fps),
		fmt.Sprintf("%d", gopSize),
		presetP,
		vSize3,
		vSize5,
		vSize6,
		out_dir,
		"50",
	)

	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run FFmpeg script: %w", err)
	}

	fmt.Println(cmd.String())

	return nil
}

// startTranscodeWorker processes files from the queue one by one
func StartTranscodeWorker(input_path, output_path string) {
	go func() {

		for id := range TranscodeQueue {
			log.Printf("####### ==> Starting transcoding for upload: %s\n", id)
			if err := TranscodePipeline(id, input_path, output_path); err != nil {
				log.Printf("Error during transcoding uploaded file - %s: %v", id, err)
			} else {
				log.Printf("Successfully transcoded uploaded file: %s", id)

				err := utils.ResizeAndConvertToWebP("/storage/tus/hls/"+id+"/thumbnail.jpg", "/storage/tus/thumbnail/"+id+"-500w.webp", 500)
				if err != nil {
					fmt.Println("covert err: ", err)
				}
			}
		}
	}()
}
