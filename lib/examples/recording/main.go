package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/go-rod/rod"
)

var (
	urls = []string{
		"https://golang.org",
		"https://github.com",
		"https://github.com/go-rod/rod",
		"https://go-rod.github.io/",
		"https://go-rod.github.io/#/get-started/README",
		"https://go-rod.github.io/#/selectors/README",
		"https://github.com/go-rod/rod/blob/main/examples_test.go",
		"https://pkg.go.dev/github.com/go-rod/rod",
	}
)

// cspell:disable

func main() {
	// Create temporary directory for frames
	tempDir, err := os.MkdirTemp("", "screencast-*")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize browser
	browser := rod.New().MustConnect()
	defer browser.MustClose()

	// Create initial page
	page := browser.MustPage(urls[0])

	quality := 90
	everyNthFrame := 1

	// Configure screencast options
	opts := &rod.ScreencastOptions{
		Format:        "jpeg",
		Quality:       &quality,
		EveryNthFrame: &everyNthFrame,
		BufferSize:    100,
	}

	// Start the screencast and get the frames channel
	frames, err := page.StartScreencast(opts)
	if err != nil {
		log.Fatalf("Failed to start screencast: %v", err)
	}

	frameCount := 0

	// Start saving frames in the main goroutine
	go func() {
		for frame := range frames {
			framePath := filepath.Join(tempDir, fmt.Sprintf("frame_%06d.jpg", frameCount))
			if err := os.WriteFile(framePath, frame, 0644); err != nil {
				log.Printf("Error saving frame: %v", err)
				continue
			}
			frameCount++
		}
	}()

	// Navigate through pages and do some scrolling
	for _, url := range urls {
		log.Printf("Navigating to: %s", url)
		page.MustNavigate(url)
		time.Sleep(time.Second)
		page.Mouse.MustScroll(0, 500)
		time.Sleep(time.Millisecond * 500)
		page.Mouse.MustScroll(0, -500)
		time.Sleep(time.Millisecond * 500)
	}

	// Stop screencast
	if err := page.StopScreencast(); err != nil {
		log.Printf("Error stopping screencast: %v", err)
	}

	// Give a short time for any remaining frames to be processed
	time.Sleep(time.Second)

	// Create output video using FFmpeg
	outputPath := "screencast.mp4"
	log.Printf("%d frames captured, generating video using ffmpeg", frameCount)
	err = createVideo(tempDir, outputPath)
	if err != nil {
		log.Fatalf("Failed to create video: %v", err)
	}

	log.Printf("Video created successfully: %s", outputPath)
}

func createVideo(framesDir, outputPath string) error {
	cmd := exec.Command("ffmpeg",
		"-y",               // Overwrite output file if it exists
		"-framerate", "30", // Input framerate
		"-pattern_type", "sequence", // Use sequential pattern
		"-i", filepath.Join(framesDir, "frame_%06d.jpg"), // Input pattern
		"-c:v", "libx264", // Use H.264 codec
		"-preset", "medium", // Balanced encoding speed/quality
		"-crf", "23", // Constant rate factor (quality)
		"-movflags", "+faststart", // Enable fast start for web playback
		"-pix_fmt", "yuv420p", // Compatible pixel format
		"-vf", "format=yuv420p", // Ensure yuv420p format
		"-an", // No audio
		outputPath,
	)

	// Capture FFmpeg output for logging
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
