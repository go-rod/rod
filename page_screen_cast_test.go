package rod_test

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/TommyLeng/go-rod"
	"github.com/TommyLeng/go-rod/lib/proto"
)

//GODEBUG="tracebackancestors=1000" go test -timeout 30s -run ^TestPageScreenCastAvi$ github.com/TommyLeng/go-rod -v -count=1 -parallel=1
func TestPageScreenCastAvi(t *testing.T) {
	//g := setup(t)

	{
		b := rod.New().MustConnect()
		p := b.MustPage("https://www.google.com").MustWaitLoad()

		time.Sleep(10 * time.Second)
		fmt.Println("slept 10 seconds")

		videoFrames := []rod.VideoFrame{}
		fps := 25

		// ScreenCastRecord listen PageScreenCastFrame and save data into videoFrames
		aviWriter, err := p.ScreenCastRecordAvi("sample.avi", &videoFrames, fps) // Only support .avi video file & frame per second
		if err != nil {
			t.Fatal(err)
		}

		// ScreenCastStart start listening ScreenCastRecord
		err = p.ScreenCastStart(100) // Image quality
		if err != nil {
			t.Fatal(err)
		}

		fmt.Println("sleep 10 seconds")
		time.Sleep(3 * time.Second)

		p.Navigate("https://www.youtube.com")

		time.Sleep(7 * time.Second)

		// ScreenCastStop stop listening ScreenCastRecord and convert the videoFrames data into avi file
		err = p.ScreenCastStopAvi(aviWriter, &videoFrames, fps)
		if err != nil {
			t.Fatal(err)
		}

		p.MustClose()
		b.MustClose()
	}
}

//GODEBUG="tracebackancestors=1000" go test -timeout 60s -run ^TestPageScreenCastAvi2$ github.com/TommyLeng/go-rod -v -count=1
func TestPageScreenCastAvi2(t *testing.T) {
	//g := setup(t)

	browser := rod.New().MustConnect()

	page := browser.MustPage("https://member.bbtb.dev")

	GoToTestPage(browser, page)

	//page.MustScreenshot("a.png")

	videoFrames := []rod.VideoFrame{}
	fps := 50

	// ScreenCastRecord listen PageScreenCastFrame and save data into videoFrames
	aviWriter, err := page.ScreenCastRecordAvi("sample.avi", &videoFrames, fps) // Only support .avi video file & frame per second
	if err != nil {
		t.Fatal(err)
	}

	// ScreenCastStart start listening ScreenCastRecord
	err = page.ScreenCastStart(100) // Image quality
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println("sleep 10 seconds")
	time.Sleep(10 * time.Second)

	// ScreenCastStop stop listening ScreenCastRecord and convert the videoFrames data into avi file
	err = page.ScreenCastStopAvi(aviWriter, &videoFrames, fps)
	if err != nil {
		t.Fatal(err)
	}

	page.MustClose()
	browser.MustClose()
}

//Direct put data from screen cast event to ffmpeg stdin, but the result is not good, the video is not smooth and it is more than 10 seconds
//GODEBUG="tracebackancestors=1000" go test -timeout 60s -run ^TestPageScreenCastDirectMp4$ github.com/TommyLeng/go-rod -v -count=1
func TestPageScreenCastDirectMp4(t *testing.T) {
	//g := setup(t)

	browser := rod.New().MustConnect()

	page := browser.MustPage("https://member.bbtb.dev")

	GoToTestPage(browser, page)

	pr, pw := io.Pipe()

	go page.EachEvent(func(e *proto.PageScreencastFrame) {
		err := proto.PageScreencastFrameAck{
			SessionID: e.SessionID,
		}.Call(page)
		if err != nil {
			fmt.Println("ScreencastFrameAck err:", err)
		}
		pw.Write(e.Data)
	})()

	//cat $(find . -maxdepth 1 -name '*.png' -print | sort | tail -10) | ffmpeg -framerate 25 -i - -vf format=yuv420p -movflags +faststart output.mp4

	cmd := exec.Command("ffmpeg", "-y", // Yes to all
		"-i", "pipe:0", // take stdin as input
		"-vf", "format=yuv420p",
		"-movflags", "+faststart",
		"output.mp4", // output
	)
	cmd.Stderr = os.Stderr // bind log stream to stderr
	cmd.Stdin = pr

	err := cmd.Start() // Start a process on another goroutine
	if err != nil {
		t.Fatal(err)
	}

	everyNthFrame := 1
	qty := 100
	proto.PageStartScreencast{
		Format:        proto.PageStartScreencastFormatJpeg,
		Quality:       &qty,
		EveryNthFrame: &everyNthFrame,
	}.Call(page)

	time.Sleep(10 * time.Second)

	err = proto.PageStopScreencast{}.Call(page)
	if err != nil {
		t.Fatal(err)
	}

	err = pw.Close()
	if err != nil {
		t.Fatal(err)
	}
	err = pr.Close() // close the stdin, or ffmpeg will wait forever
	if err != nil {
		t.Fatal(err)
	}

	err = cmd.Wait() // wait until ffmpeg finish
	if err != nil {
		t.Fatal(err)
	}

	page.MustClose()
	browser.MustClose()
}

//GODEBUG="tracebackancestors=1000" go test -timeout 60s -run ^TestPageScreenCastMp4UsingPipe$ github.com/TommyLeng/go-rod -v -count=1
func TestPageScreenCastMp4UsingPipe(t *testing.T) {
	//g := setup(t)

	browser := rod.New().MustConnect()

	page := browser.MustPage("https://member.bbtb.dev")

	GoToTestPage(browser, page)

	videoFrames := []rod.VideoFrame{}

	// ScreenCastRecord listen PageScreenCastFrame and save data into videoFrames
	err := page.ScreenCastRecordMp4(&videoFrames)
	if err != nil {
		t.Fatal(err)
	}

	// ScreenCastStart start listening ScreenCastRecord
	err = page.ScreenCastStart(100) // Image quality & frame per second
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println("sleep 10 seconds")
	time.Sleep(10 * time.Second)

	// ScreenCastStop stop listening ScreenCastRecord and convert the videoFrames data into mp4 file
	err = page.ScreenCastStopMp4UsingPipe(&videoFrames, "output.mp4", 25)
	if err != nil {
		t.Fatal(err)
	}

	page.MustClose()
	browser.MustClose()
}

//GODEBUG="tracebackancestors=1000" go test -timeout 60s -run ^TestPageScreenCastMp4$ github.com/TommyLeng/go-rod -v -count=1
func TestPageScreenCastMp4(t *testing.T) {
	//g := setup(t)

	browser := rod.New().MustConnect()

	page := browser.MustPage("https://member.bbtb.dev")

	GoToTestPage(browser, page)

	videoFrames := []rod.VideoFrame{}

	// ScreenCastRecord listen PageScreenCastFrame and save data into videoFrames
	err := page.ScreenCastRecordMp4(&videoFrames)
	if err != nil {
		t.Fatal(err)
	}

	// ScreenCastStart start listening ScreenCastRecord
	err = page.ScreenCastStart(100) // Image quality & frame per second
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println("sleep 10 seconds")
	time.Sleep(10 * time.Second)

	// ScreenCastStop stop listening ScreenCastRecord and convert the videoFrames data into mp4 file
	err = page.ScreenCastStopMp4(&videoFrames, "output.mp4")
	if err != nil {
		t.Fatal(err)
	}

	page.MustClose()
	browser.MustClose()
}

func GoToTestPage(browser *rod.Browser, page *rod.Page) {
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
}
