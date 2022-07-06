package rod_test

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

//GODEBUG="tracebackancestors=1000" go test -timeout 60s -run ^TestPageScreenCastAvi$ github.com/go-rod/rod -v -count=1 -parallel=1
func TestPageScreenCastAvi(t *testing.T) {
	g := setup(t)

	{
		browser := rod.New().MustConnect()
		page := browser.MustPage("http://www.google.com").MustWaitLoad()

		videoFrames := []rod.VideoFrame{}
		fps := 100

		// ScreenCastRecord listen PageScreenCastFrame and save data into videoFrames
		aviWriter, err := page.ScreenCastRecordAvi("sample.avi", &videoFrames, fps) // Only support .avi video file & frame per second
		if err != nil {
			g.Fatal(err)
		}

		// ScreenCastStart start listening ScreenCastRecord
		err = page.ScreenCastStart(50) // Image quality
		if err != nil {
			g.Fatal(err)
		}

		fmt.Println("sleep 10 seconds start: ", time.Now())
		time.Sleep(6 * time.Second)

		page.Navigate("https://dayspedia.com/time/online/")
		page.MustWaitNavigation()
		page.MustWaitLoad()
		time.Sleep(4 * time.Second)

		page.Navigate("http://www.google.com")
		page.MustWaitNavigation()
		page.MustWaitLoad()
		time.Sleep(4 * time.Second)

		// ScreenCastStop stop listening ScreenCastRecord and convert the videoFrames data into avi file
		err = page.ScreenCastStopAvi(aviWriter, &videoFrames, fps)
		if err != nil {
			g.Fatal(err)
		}

		page.MustClose()
		browser.MustClose()
	}
}

//Direct put data from screen cast event to ffmpeg stdin,
//but the result is not good, the video is not smooth, sometimes fast, sometimes slow, sometimes it is more than 10 seconds
//and we dun know the frame rate from the screen cast event, so the video duration may be longer / shorter than we expect, we may need to change the -r argument in ffmpeg
//and screen cast event didn't send data on sequence, it cause the video sometimes shift backward & forward a little bit
//but using pipe can save memory
//GODEBUG="tracebackancestors=1000" go test -timeout 60s -run ^TestPageScreenCastDirectPipeMp4$ github.com/go-rod/rod -v -count=1 -parallel=1
func TestPageScreenCastDirectPipeMp4(t *testing.T) {
	g := setup(t)

	{
		browser := rod.New().MustConnect()

		page := browser.MustPage("https://dayspedia.com/time/online/").MustWaitLoad()

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
			"-r", "60",
			"-vf", "format=yuv420p",
			"-movflags", "+faststart",
			"output.mp4", // output
		)
		cmd.Stderr = os.Stderr // bind log stream to stderr
		cmd.Stdin = pr

		err := cmd.Start() // Start a process on another goroutine
		if err != nil {
			g.Fatal(err)
		}

		everyNthFrame := 1
		qty := 50
		proto.PageStartScreencast{
			Format:        proto.PageStartScreencastFormatJpeg,
			Quality:       &qty,
			EveryNthFrame: &everyNthFrame,
		}.Call(page)

		time.Sleep(10 * time.Second)

		err = proto.PageStopScreencast{}.Call(page)
		if err != nil {
			g.Fatal(err)
		}

		err = pw.Close()
		if err != nil {
			g.Fatal(err)
		}
		err = pr.Close() // close the stdin, or ffmpeg will wait forever
		if err != nil {
			g.Fatal(err)
		}

		err = cmd.Wait() // wait until ffmpeg finish
		if err != nil {
			g.Fatal(err)
		}

		page.MustClose()
		browser.MustClose()
	}
}

//Best approach I found to capture mp4
//It sort the data from screen cast event, and insert frames base on the input fps in ScreenCastStopMp4
//GODEBUG="tracebackancestors=1000" go test -timeout 60s -run ^TestPageScreenCastMp4$ github.com/go-rod/rod -v -count=1 -parallel=1
func TestPageScreenCastMp4(t *testing.T) {
	g := setup(t)

	{
		browser := rod.New().MustConnect()
		page := browser.MustPage("http://www.google.com").MustWaitLoad()

		videoFrames := []rod.VideoFrame{}
		fps := 25

		// ScreenCastRecord listen PageScreenCastFrame and save data into videoFrames
		err := page.ScreenCastRecordMp4(&videoFrames)
		if err != nil {
			g.Fatal(err)
		}

		// ScreenCastStart start listening ScreenCastRecord
		err = page.ScreenCastStart(100) // Image quality & frame per second
		if err != nil {
			g.Fatal(err)
		}

		fmt.Println("sleep 10 seconds start: ", time.Now())
		time.Sleep(6 * time.Second)

		page.Navigate("https://dayspedia.com/time/online/")
		page.MustWaitNavigation()
		page.MustWaitLoad()
		time.Sleep(4 * time.Second)

		page.Navigate("http://www.google.com")
		page.MustWaitNavigation()
		page.MustWaitLoad()
		time.Sleep(4 * time.Second)

		// ScreenCastStop stop listening ScreenCastRecord and convert the videoFrames data into mp4 file
		err = page.ScreenCastStopMp4(&videoFrames, "output.mp4", fps)
		if err != nil {
			g.Fatal(err)
		}

		page.MustClose()
		browser.MustClose()
	}
}

//Test concurrent capture mp4 from several browser
//GODEBUG="tracebackancestors=1000" go test -timeout 60s -run ^TestConcurrentCaptureMp4$ github.com/go-rod/rod -v -count=1 -parallel=1
func TestConcurrentCaptureMp4(t *testing.T) {
	g := setup(t)

	{
		browser := rod.New().MustConnect()

		type PageScreenCastInfo struct {
			Page        *rod.Page
			VideoFrames *[]rod.VideoFrame
		}

		pageMap := map[string]PageScreenCastInfo{}

		page1 := browser.MustPage("https://www.timeanddate.com/worldclock/hong-kong/hong-kong").MustWaitLoad()
		page2 := browser.MustPage("https://www.timeanddate.com/worldclock/japan/tokyo").MustWaitLoad()

		pageMap["1"] = PageScreenCastInfo{
			Page: page1,
			VideoFrames: &[]rod.VideoFrame{},
		}
		pageMap["2"] = PageScreenCastInfo{
			Page: page2,
			VideoFrames: &[]rod.VideoFrame{},
		}

		fps := 25

		var err error

		// ScreenCastRecord listen PageScreenCastFrame and save data into videoFrames
		err = pageMap["1"].Page.ScreenCastRecordMp4(pageMap["1"].VideoFrames)
		if err != nil {
			g.Fatal(err)
		}
		err = pageMap["2"].Page.ScreenCastRecordMp4(pageMap["2"].VideoFrames)
		if err != nil {
			g.Fatal(err)
		}

		// ScreenCastStart start listening ScreenCastRecord
		err = pageMap["1"].Page.ScreenCastStart(100)
		if err != nil {
			g.Fatal(err)
		}
		err = pageMap["2"].Page.ScreenCastStart(100)
		if err != nil {
			g.Fatal(err)
		}

		fmt.Println("sleep 10 seconds start: ", time.Now())
		time.Sleep(6 * time.Second)

		pageMap["1"].Page.Navigate("https://dayspedia.com/time/online/")
		pageMap["2"].Page.Navigate("https://dayspedia.com/time/online/")
		pageMap["1"].Page.MustWaitNavigation()
		pageMap["2"].Page.MustWaitNavigation()
		pageMap["1"].Page.MustWaitLoad()
		pageMap["2"].Page.MustWaitLoad()
		time.Sleep(4 * time.Second)

		pageMap["1"].Page.Navigate("https://www.timeanddate.com/worldclock/hong-kong/hong-kong")
		pageMap["2"].Page.Navigate("https://www.timeanddate.com/worldclock/japan/tokyo")
		pageMap["1"].Page.MustWaitNavigation()
		pageMap["2"].Page.MustWaitNavigation()
		pageMap["1"].Page.MustWaitLoad()
		pageMap["2"].Page.MustWaitLoad()
		time.Sleep(4 * time.Second)

		// ScreenCastStop stop listening ScreenCastRecord and convert the videoFrames data into mp4 file
		err = pageMap["1"].Page.ScreenCastStopMp4(pageMap["1"].VideoFrames, "output_1.mp4", fps)
		if err != nil {
			g.Fatal(err)
		}
		err = pageMap["2"].Page.ScreenCastStopMp4(pageMap["2"].VideoFrames, "output_2.mp4", fps)
		if err != nil {
			g.Fatal(err)
		}

		pageMap["1"].Page.MustClose()
		pageMap["2"].Page.MustClose()
		browser.MustClose()
	}
}
