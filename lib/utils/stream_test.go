package utils

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

/* cspell:ignore mjpeg */

func TestWriteMJPEGFrame(t *testing.T) {
	g := setup(t)
	frame := []byte("test-image-data")

	// Set up the test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Set the Content-Type header for MJPEG
		w.Header().Set("Content-Type", "multipart/x-mixed-replace; boundary=frame")

		// Ensure the ResponseWriter supports flushing
		flusher, ok := w.(http.Flusher)
		g.E(ok)

		// Write a single MJPEG frame
		if err := WriteMJPEGFrame(w, frame, flusher); err != nil {
			t.Fatalf("WriteMJPEGFrame failed: %v", err)
		}
	}))
	defer server.Close()

	// Make a request to the server
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 2 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
	g.E(err)

	resp, err := client.Do(req)
	g.E(err)

	defer func() {
		err := resp.Body.Close()
		g.E(err)
	}()

	// Verify the Content-Type header
	if resp.Header.Get("Content-Type") != "multipart/x-mixed-replace; boundary=frame" {
		t.Fatalf("Unexpected Content-Type header: %v", resp.Header.Get("Content-Type"))
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil && err.Error() != "EOF" { // Ignore EOF errors for streaming
		t.Fatalf("Failed to read response body: %v", err)
	}

	// Verify the MJPEG frame content
	expected := "--frame\r\nContent-Type: image/jpeg\r\nContent-Length: 15\r\n\r\ntest-image-data\r\n"
	if string(body) != expected {
		t.Fatalf("Unexpected response body:\nGot:\n%s\nExpected:\n%s", string(body), expected)
	}
}
