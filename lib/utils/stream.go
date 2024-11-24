package utils

import (
	"fmt"
	"net/http"
)

// WriteMJPEGFrame writes a single MJPEG frame to the response writer.
func WriteMJPEGFrame(w http.ResponseWriter, frame []byte, flusher http.Flusher) error {
	parts := [][]byte{
		[]byte("--frame\r\n"),
		[]byte("Content-Type: image/jpeg\r\n"),
		[]byte(fmt.Sprintf("Content-Length: %d\r\n\r\n", len(frame))),
		frame,
		[]byte("\r\n"),
	}

	for _, part := range parts {
		if _, err := w.Write(part); err != nil {
			return err
		}
	}

	flusher.Flush()
	return nil
}

// MustWriteMJPEGFrame is similar to WriteMJPEGFrame but panics if an error occurs.
func MustWriteMJPEGFrame(w http.ResponseWriter, frame []byte, flusher http.Flusher) {
	err := WriteMJPEGFrame(w, frame, flusher)
	E(err)
}
