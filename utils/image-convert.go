package utils

import (
	"bytes"
	"errors"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/kolesa-team/go-webp/decoder"
	"github.com/kolesa-team/go-webp/encoder"
	"github.com/kolesa-team/go-webp/webp"
	"github.com/nfnt/resize"
)

// ResizeAndConvertToWebP resizes the input image to a specified width and converts it to WebP.
func ResizeAndConvertToWebP(inputPath string, outputPath string, width uint) error {
	// Open the source image file
	file, err := os.Open(inputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Read the first few bytes to check for WebP signature
	header := make([]byte, 4)
	if _, err := file.Read(header); err != nil {
		return err
	}

	// Check if the file is a WebP file
	if bytes.Equal(header, []byte("RIFF")) {
		// Seek back to the beginning of the file
		_, err = file.Seek(0, 0)
		if err != nil {
			return err
		}

		// Decode the WebP image
		img, err := webp.Decode(file, &decoder.Options{})
		if err != nil {
			return err
		}

		return processImage(img, outputPath, width)
	}

	// Reset file pointer after checking header before decoding the image
	_, err = file.Seek(0, 0)
	if err != nil {
		return err
	}

	// Use image.DecodeConfig to get the image format without relying on the file extension
	_, format, err := image.DecodeConfig(file)
	if err != nil {
		return err
	}

	// Reset file pointer after DecodeConfig since it reads part of the file
	_, err = file.Seek(0, 0)
	if err != nil {
		return err
	}

	// Declare the image variable
	var img image.Image

	// Decode the image based on its actual format
	switch format {
	case "png":
		img, err = png.Decode(file)
		if err != nil {
			return err
		}
	case "jpeg":
		img, err = jpeg.Decode(file)
		if err != nil {
			return err
		}
	default:
		return errors.New("unsupported file type") // Unsupported file type
	}

	return processImage(img, outputPath, width)
}

// processImage resizes the image to the specified width while maintaining aspect ratio and encodes it to WebP format.
func processImage(img image.Image, outputPath string, width uint) error {
	// Get the original dimensions
	bounds := img.Bounds()
	originalWidth := bounds.Dx()
	originalHeight := bounds.Dy()

	// Calculate the new height while maintaining the aspect ratio
	newHeight := uint(float64(originalHeight) * (float64(width) / float64(originalWidth)))

	// Resize the image to the specified width while maintaining the aspect ratio
	resizedImg := resize.Resize(width, newHeight, img, resize.Lanczos3)

	// Ensure the output directory exists, if not, create it
	outputDir := filepath.Dir(outputPath)
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		err := os.MkdirAll(outputDir, os.ModePerm) // Create the directory with permissions
		if err != nil {
			return err
		}
	}

	// Create the output WebP file
	output, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer output.Close()

	// Set WebP encoding options (using lossy compression with quality = 75)
	options, err := encoder.NewLossyEncoderOptions(encoder.PresetDefault, 75)
	if err != nil {
		return err
	}

	// Encode the resized image to WebP format
	if err := webp.Encode(output, resizedImg, options); err != nil {
		return err
	}

	return nil
}
