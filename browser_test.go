package rod_test

import (
	"testing"
	"time"

	"github.com/ysmood/kit"
	"github.com/ysmood/rod"
	"github.com/ysmood/rod/lib/cdp"
)

func (s *S) TestBrowserPages() {
	page := s.browser.Page(srcFile("fixtures/click.html"))
	defer page.Close()

	page.Element("button")
	pages := s.browser.Pages()

	s.Len(pages, 3)
}

func (s *S) TestBrowserContext() {
	b := s.browser.Timeout(time.Minute).CancelTimeout().Cancel()
	_, err := b.CallE(&cdp.Request{})
	s.Error(err)
}

func (s *S) TestIncognito() {
	file := srcFile("fixtures/click.html")
	k := kit.RandString(8)

	b := s.browser.Incognito()
	page := b.Page(file)
	page.Eval(`k => localStorage[k] = 1`, k)

	s.Nil(s.page.Navigate(file).Eval(`k => localStorage[k]`, k).Value())
	s.EqualValues(1, page.Eval(`k => localStorage[k]`, k).Int())
}

func (s *S) TestBrowserWaitEvent() {
	wait := s.browser.WaitEvent("Page.frameNavigated")
	s.page.Navigate(srcFile("fixtures/click.html"))
	wait()
}

func (s *S) TestBrowserCall() {
	v := s.browser.Call("Browser.getVersion", nil)

	s.Regexp("HeadlessChrome", v.Get("product").String())
}

// It's obvious that, the v8 will take more time to parse long function.
// For BenchmarkCache and BenchmarkNoCache, the difference is nearly 12% which is too much to ignore.
func BenchmarkCacheOff(b *testing.B) {
	p := rod.New().Connect().Page(srcFile("fixtures/click.html"))

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		p.Eval(`(time) => {
			// won't call this function, it's used to make the declaration longer
			function foo (id, left, top, width, height, msg) {
				var div = document.createElement('div')
				var msgDiv = document.createElement('div')
				div.id = id
				div.style = 'position: fixed; z-index:2147483647; border: 2px dashed red;'
					+ 'border-radius: 3px; box-shadow: #5f3232 0 0 3px; pointer-events: none;'
					+ 'box-sizing: border-box;'
					+ 'left:' + left + 'px;'
					+ 'top:' + top + 'px;'
					+ 'height:' + height + 'px;'
					+ 'width:' + width + 'px;'
		
				if (height === 0) {
					div.style.border = 'none'
				}
			
				msgDiv.style = 'position: absolute; color: #cc26d6; font-size: 12px; background: #ffffffeb;'
					+ 'box-shadow: #333 0 0 3px; padding: 2px 5px; border-radius: 3px; white-space: nowrap;'
					+ 'top:' + height + 'px; '
			
				msgDiv.innerHTML = msg
			
				div.appendChild(msgDiv)
				document.body.appendChild(div)
			}
			return time
		}`, time.Now().UnixNano())
	}
}

func BenchmarkCache(b *testing.B) {
	p := rod.New().Connect().Page(srcFile("fixtures/click.html"))

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		p.Eval(`(time) => {
			return time
		}`, time.Now().UnixNano())
	}
}
