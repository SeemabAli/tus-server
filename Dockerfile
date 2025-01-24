# Use an official Golang image as the base
FROM golang:1.22.5-alpine

# Install dependencies for Go, FFmpeg, and build tools
RUN apk add --no-cache bc ffmpeg bash gcc g++ libc-dev libwebp libwebp-tools libwebp-dev wget curl vim git

# Set the working directory inside the container
WORKDIR /app

# Copy the Go module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application code
COPY . .
RUN mkdir -p /app/uploads

RUN go mod tidy
RUN go get -u github.com/kolesa-team/go-webp

# Build the Go application
RUN go build -o server .

# Copy the static files (e.g., HTML templates, scripts)
COPY image-demo.html ./image-demo.html
COPY video-demo.html ./video-demo.html
COPY run.sh ./run.sh

# Expose the port the app runs on
EXPOSE 8080

# Command to run the server
CMD ["./server"]
