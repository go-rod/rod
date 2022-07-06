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
		b := rod.New().MustConnect()
		p := b.MustPage("https://dayspedia.com/time/online/").MustWaitLoad()

		videoFrames := []rod.VideoFrame{}
		fps := 50

		// ScreenCastRecord listen PageScreenCastFrame and save data into videoFrames
		aviWriter, err := p.ScreenCastRecordAvi("sample.avi", &videoFrames, fps) // Only support .avi video file & frame per second
		if err != nil {
			g.Fatal(err)
		}

		// ScreenCastStart start listening ScreenCastRecord
		err = p.ScreenCastStart(50) // Image quality
		if err != nil {
			g.Fatal(err)
		}

		fmt.Println("sleep 10 seconds start: ", time.Now())
		time.Sleep(10 * time.Second)

		// ScreenCastStop stop listening ScreenCastRecord and convert the videoFrames data into avi file
		err = p.ScreenCastStopAvi(aviWriter, &videoFrames, fps)
		if err != nil {
			g.Fatal(err)
		}

		p.MustClose()
		b.MustClose()
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
//But I dunno why I cannot change frame rate to 50 fps, it still make use 25 fps, so I speed the video up using setpts=0.5*PTS in ffmpeg in order to achieve 50 fps
//So it hardcode 50 fps, please help to modify if u know why
//It sort the data from screen cast event, and insert frames base on 50 fps
//GODEBUG="tracebackancestors=1000" go test -timeout 60s -run ^TestPageScreenCastMp4$ github.com/go-rod/rod -v -count=1 -parallel=1
func TestPageScreenCastMp4(t *testing.T) {
	g := setup(t)

	{
		browser := rod.New().MustConnect()

		page := browser.MustPage("https://dayspedia.com/time/online/").MustWaitLoad()

		videoFrames := []rod.VideoFrame{}

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

		fmt.Println("sleep 10 seconds")
		time.Sleep(10 * time.Second)

		// ScreenCastStop stop listening ScreenCastRecord and convert the videoFrames data into mp4 file
		err = page.ScreenCastStopMp4(&videoFrames, "output.mp4")
		if err != nil {
			g.Fatal(err)
		}

		page.MustClose()
		browser.MustClose()
	}
}
