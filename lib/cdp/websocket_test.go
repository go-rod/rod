package cdp_test

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-rod/rod/lib/cdp"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/ysmood/gson"
)

func (t T) WebSocketLargePayload() {
	ctx := t.Context()
	client, id := t.newPage(ctx)

	res, err := client.Call(ctx, id, "Runtime.evaluate", map[string]interface{}{
		"expression":    fmt.Sprintf(`"%s"`, strings.Repeat("a", 2*1024*1024)),
		"returnByValue": true,
	})
	t.E(err)
	t.Gt(res, 2*1024*1024) // 2MB
}

func (t T) WebSocketHeader() {
	s := t.Serve()

	wait := make(chan struct{})
	s.Mux.HandleFunc("/a", func(rw http.ResponseWriter, r *http.Request) {
		t.Eq(r.Header.Get("Test"), "header")
		t.Eq(r.Host, "test.com")
		t.Eq(r.URL.Query().Get("q"), "ok")
		close(wait)
	})

	ws := cdp.WebSocket{}
	err := ws.Connect(t.Context(), s.URL("/a?q=ok"), http.Header{
		"Host": {"test.com"},
		"Test": {"header"},
	})
	<-wait

	t.Eq(err.Error(), "websocket bad handshake: 200 OK. ")
}

func (t T) newPage(ctx context.Context) (*cdp.Client, string) {
	l := launcher.New()
	t.Cleanup(l.Kill)

	client := cdp.New(l.MustLaunch()).MustConnect(ctx)

	go func() {
		for range client.Event() {
		}
	}()

	file, err := filepath.Abs(filepath.FromSlash("fixtures/basic.html"))
	t.E(err)

	res, err := client.Call(ctx, "", "Target.createTarget", map[string]interface{}{
		"url": "file://" + file,
	})
	t.E(err)

	targetID := gson.New(res).Get("targetId").String()

	res, err = client.Call(ctx, "", "Target.attachToTarget", map[string]interface{}{
		"targetId": targetID,
		"flatten":  true,
	})
	t.E(err)

	sessionID := gson.New(res).Get("sessionId").String()

	return client, sessionID
}
