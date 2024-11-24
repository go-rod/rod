package main

import (
	"log"
	"net/http"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/utils"
	"github.com/gorilla/websocket"
)

var (
	// URLs to cycle through in the demo
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

	// WebSocket connection handling
	upgrader = websocket.Upgrader{}
	clients  = make(map[*websocket.Conn]bool)

	// Channels for frame distribution
	broadcast = make(chan []byte, 100)
	mjpegChan = make(chan []byte, 100)
)

func main() {
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

	// Distribute frames to WebSocket and MJPEG clients
	go func() {
		for frame := range frames {
			broadcast <- frame
			mjpegChan <- frame
		}
	}()

	// Set up HTTP handlers
	http.HandleFunc("/", serveHTML)
	http.HandleFunc("/ws", handleWebSocket)
	http.HandleFunc("/mjpeg", serveMJPEG)

	// Start broadcasting frames to WebSocket clients
	go handleBroadcast()

	// Start HTTP server
	go func() {
		log.Printf("Starting server at http://localhost:8282")
		if err := http.ListenAndServe(":8282", nil); err != nil {
			log.Fatal(err)
		}
	}()

	// Navigate between pages and do some scrolling to demonstrate the screencast
	for {
		for _, url := range urls {
			log.Printf("Navigating to: %s", url)
			page.MustNavigate(url)
			time.Sleep(time.Second)
			page.Mouse.MustScroll(0, 500)
			time.Sleep(time.Millisecond * 500)
			page.Mouse.MustScroll(0, -500)
			time.Sleep(time.Millisecond * 500)
		}
	}
}

func serveHTML(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Rod Screencast Demo</title>
    <style>
        body { max-width: 95%; margin: 0 auto; padding: 20px; }
        .streams { display: flex; gap: 2rem; }
        .stream { flex: 1; }
        img { max-width: 100%; border: 1px solid #ccc; }
        h1, h2 { color: #333; }
    </style>
</head>
<body>
    <h1>Rod Screencast Demo</h1>
    <div class="streams">
        <div class="stream">
            <h2>WebSocket Stream</h2>
            <img id="ws-stream" alt="WebSocket Stream" />
        </div>
        <div class="stream">
            <h2>MJPEG Stream</h2>
            <img src="/mjpeg" alt="MJPEG Stream" />
        </div>
    </div>
    <script>
        const ws = new WebSocket("ws://" + location.host + "/ws");
        const img = document.getElementById('ws-stream');
        
        ws.onmessage = function(event) {
            img.src = URL.createObjectURL(event.data);
        };
        
        ws.onerror = function(event) {
            console.error("WebSocket error:", event);
        };
    </script>
</body>
</html>`
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	clients[conn] = true
	defer delete(clients, conn)

	// Keep connection alive until client disconnects
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}

func serveMJPEG(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "multipart/x-mixed-replace; boundary=frame")

	for frame := range mjpegChan {
		if f, ok := w.(http.Flusher); ok {
			// Rod internal utility function to write a MJPEG frame
			utils.WriteMJPEGFrame(w, frame, f)
		}
	}
}

func handleBroadcast() {
	for frame := range broadcast {
		for client := range clients {
			if err := client.WriteMessage(websocket.BinaryMessage, frame); err != nil {
				client.Close()
				delete(clients, client)
			}
		}
	}
}
