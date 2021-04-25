package launcher_test

import (
	"context"
	"math/rand"
	"sync"
	"testing"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/utils"
	"github.com/ysmood/got"
)

func BenchmarkRemoteLauncher(b *testing.B) {
	const concurrent = 30 // how many browsers will run at the same time
	const num = 300       // how many browsers we will launch

	limiter := make(chan int, concurrent)

	s := got.New(b).Serve()

	// docker run --rm -p 9222:9222 ghcr.io/go-rod/rod
	s.HostURL.Host = "host.docker.internal"

	s.Route("/", ".html", `<html><body>
		ok
	</body><script>
		function wait() {
			return new Promise(r => setTimeout(r, 1000 * Math.random()))
		}
	</script></html>`)

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
			page := browser.MustPage()
			wait := page.MustWaitNavigation()
			page.MustNavigate(s.URL())
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
