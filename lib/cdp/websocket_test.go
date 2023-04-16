package cdp_test

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/go-rod/rod/lib/cdp"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/utils"
	"github.com/ysmood/got"
	"github.com/ysmood/gson"
)

func TestWebSocketLargePayload(t *testing.T) {
	g := setup(t)

	ctx := g.Context()
	client, id := newPage(ctx, g)

	res, err := client.Call(ctx, id, "Runtime.evaluate", map[string]interface{}{
		"expression":    fmt.Sprintf(`"%s"`, strings.Repeat("a", 2*1024*1024)),
		"returnByValue": true,
	})
	g.E(err)
	g.Gt(res, 2*1024*1024) // 2MB
}

func ConcurrentCall(t *testing.T) {
	g := setup(t)

	ctx := g.Context()
	client, id := newPage(ctx, g)

	wg := sync.WaitGroup{}
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func() {
			res, err := client.Call(ctx, id, "Runtime.evaluate", map[string]interface{}{
				"expression": `10`,
			})
			g.Nil(err)
			g.Eq(string(res), "{\"result\":{\"type\":\"number\",\"value\":10,\"description\":\"10\"}}")
			wg.Done()
		}()
	}
	wg.Wait()
}

func TestWebSocketHeader(t *testing.T) {
	g := setup(t)

	s := g.Serve()

	wait := make(chan struct{})
	s.Mux.HandleFunc("/a", func(rw http.ResponseWriter, r *http.Request) {
		g.Eq(r.Header.Get("Test"), "header")
		g.Eq(r.Host, "test.com")
		g.Eq(r.URL.Query().Get("q"), "ok")
		close(wait)
	})

	ws := cdp.WebSocket{}
	err := ws.Connect(g.Context(), s.URL("/a?q=ok"), http.Header{
		"Host": {"test.com"},
		"Test": {"header"},
	})
	<-wait

	g.Eq(err.Error(), "websocket bad handshake: 200 OK. ")
}

func newPage(ctx context.Context, g got.G) (*cdp.Client, string) {
	l := launcher.New()
	g.Cleanup(l.Kill)

	client := cdp.New().Start(cdp.MustConnectWS(l.MustLaunch()))

	go func() {
		for range client.Event() {
			utils.Noop()
		}
	}()

	file, err := filepath.Abs(filepath.FromSlash("fixtures/basic.html"))
	g.E(err)

	res, err := client.Call(ctx, "", "Target.createTarget", map[string]interface{}{
		"url": "file://" + file,
	})
	g.E(err)

	targetID := gson.New(res).Get("targetId").String()

	res, err = client.Call(ctx, "", "Target.attachToTarget", map[string]interface{}{
		"targetId": targetID,
		"flatten":  true,
	})
	g.E(err)

	sessionID := gson.New(res).Get("sessionId").String()

	return client, sessionID
}

func TestDuplicatedConnectErr(t *testing.T) {
	g := setup(t)

	l := launcher.New()
	g.Cleanup(l.Kill)

	u := l.MustLaunch()

	ws := &cdp.WebSocket{}
	g.E(ws.Connect(g.Context(), u, nil))

	g.Panic(func() {
		_ = ws.Connect(g.Context(), u, nil)
	})
}
