package rod_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/TommyLeng/go-rod"
)

//GODEBUG="tracebackancestors=1000" go test -timeout 30s -run ^TestPageScreenCastAvi$ github.com/TommyLeng/go-rod -v -count=1 -parallel=1
func TestPageScreenCastAvi(t *testing.T) {
	g := setup(t)

	{
		b := g.browser.MustConnect()
		defer b.MustClose()

		p := b.MustPage("https://www.google.com").MustWaitLoad()

		time.Sleep(10 * time.Second)
		fmt.Println("slept 10 seconds")

		videoFrames := []rod.VideoFrame{}
		fps := 25

		// ScreenCastRecord listen PageScreenCastFrame and convert it directly into AVI Movie
		aviWriter, errorRecord := p.ScreenCastRecordAvi("sample.avi", &videoFrames, fps) // Only support .avi video file & frame per second
		if errorRecord != nil {
			t.Fatal(errorRecord)
		}

		// ScreenCastStart start listening ScreenCastRecord
		errorStart := p.ScreenCastStartAvi(100) // Image quality & frame per second
		if errorStart != nil {
			t.Fatal(errorStart)
		}

		fmt.Println("sleep 10 seconds")
		time.Sleep(10 * time.Second)

		p.Navigate("https://member.bbtb.dev")

		// ScreenCastStop stop listening ScreenCastRecord
		errorStop := p.ScreenCastStopAvi(aviWriter, &videoFrames, fps)
		if errorStop != nil {
			t.Fatal(errorStop)
		}

		p.MustClose()
	}
}

//GODEBUG="tracebackancestors=1000" go test -timeout 60s -run ^TestPageScreenCastAvi2$ github.com/TommyLeng/go-rod -v -count=1
func TestPageScreenCastAvi2(t *testing.T) {
	g := setup(t)

	{
		browser := g.browser.MustConnect()

		page := browser.MustPage("https://member.bbtb.dev")

		page.MustEval(`k => localStorage[k] = true`, "hasEnterBefore")

		//page.WaitElementsMoreThan(".TermAndConditionCheckbox-agree-tnc-checkbox-bhCdR", 0)
		l := page.MustElement(`.TermAndConditionCheckbox-agree-tnc-checkbox-bhCdR`)

		time.Sleep(2 * time.Second)
		fmt.Println("1")

		l.MustElement("label").MustWaitStable().MustClick()
		fmt.Println("2")

		page.MustElement(".LoginForm-visitor-button-uLUAT").MustClick()
		time.Sleep(2 * time.Second)
		fmt.Println("3")

		page.MustElement("#captcha").MustInput("11111")
		fmt.Println("4")

		time.Sleep(2 * time.Second)
		page.MustElement(".CaptchaModal-button-group-YP5iw button").MustClick()
		fmt.Println("5")

		time.Sleep(3 * time.Second)
		page.MustElement(".BetTable-game-instance-OBAbO").MustClick()
		time.Sleep(3 * time.Second)
		fmt.Println("6")

		//page.MustScreenshot("a.png")

		videoFrames := []rod.VideoFrame{}
		fps := 50

		// ScreenCastRecord listen PageScreenCastFrame and convert it directly into AVI Movie
		aviWriter, errorRecord := page.ScreenCastRecordAvi("sample.avi", &videoFrames, fps) // Only support .avi video file & frame per second
		if errorRecord != nil {
			g.Fatal(errorRecord)
		}
		/*
			errorRecord := page.ScreenCastJPGRecord()
			if errorRecord != nil {
				t.Fatal(errorRecord)
			}
		*/

		// ScreenCastStart start listening ScreenCastRecord
		errorStart := page.ScreenCastStartAvi(100) // Image quality & frame per second
		if errorStart != nil {
			g.Fatal(errorStart)
		}

		fmt.Println("sleep 10 seconds")
		time.Sleep(10 * time.Second)

		// ScreenCastStop stop listening ScreenCastRecord
		errorStop := page.ScreenCastStopAvi(aviWriter, &videoFrames, fps)
		if errorStop != nil {
			g.Fatal(errorStop)
		}

		page.MustClose()
		browser.MustClose()
	}
}
