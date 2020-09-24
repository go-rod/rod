package launcher_test

import (
	"context"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/utils"
)

func TestRemoteLauncherUnderLoad(t *testing.T) {
	if _, has := os.LookupEnv("loadtest"); !has {
		t.Skip("use env loadtest to not skip")
	}

	const concurrent = 30 // how many browsers will run at the same time
	const num = 300       // how many browsers we will launch

	limiter := make(chan int, concurrent)

	u, mux, close := utils.Serve("")
	defer close()

	// docker run --rm -p 9222:9222 rodorg/rod
	u = strings.ReplaceAll(u, "127.0.0.1", "host.docker.internal")

	mux.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Add("Content-Type", "text/html; charset=utf-8")
		utils.E(rw.Write([]byte(`<html><body>
			ok
		</body><script>
			function wait() {
				return new Promise(r => setTimeout(r, 1000 * Math.random()))
			}
		</script></html>`)))
	})

	wg := &sync.WaitGroup{}
	wg.Add(num)
	for i := 0; i < num; i++ {
		limiter <- 0

		go func() {
			utils.Sleep(rand.Float64())

			ctx, cancel := context.WithCancel(context.Background())
			defer func() {
				go func() {
					utils.Sleep(2)
					cancel()
				}()
			}()

			l := launcher.MustNewRemote("http://127.0.0.1:9222")
			client := l.Client()
			browser := rod.New().Context(ctx).Client(client).MustConnect()
			page := browser.MustPage("")
			wait := page.MustWaitNavigation()
			page.MustNavigate(u)
			wait()
			page.MustEval(`wait()`)

			if rand.Int()%10 == 0 {
				// 10% we will drop the websocket connection without call the api to gracefully close the browser
				cancel()
			} else {
				browser.MustClose()
			}

			wg.Done()
			<-limiter
		}()
	}
	wg.Wait()
}
