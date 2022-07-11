package rod

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"time"

	"github.com/go-rod/rod/lib/proto"
	"github.com/icza/mjpeg"
)

//MaxVideoFrame prevent video non stop recording, memory may not enough
//if fps = 50, 20000 max frame mean you can capture 400s second video
const MaxVideoFrame = 20000

//VideoFrame store the data from screen cast event
type VideoFrame struct {
	Data                  []byte
	Timestamp             time.Time
	DurationInSecond      float64
	AccumDurationInSecond float64
	FrameCnt              float64
	FrameCntRemaining     float64
}

//ScreenCastRecordAvi listen PageScreenCastFrame and convert it directly into AVI Movie
func (p *Page) ScreenCastRecordAvi(videoAVIPath string, videoFrames *[]VideoFrame, fps int) (*mjpeg.AviWriter, error) {
	browserBound, err := p.GetWindow()
	if err != nil {
		return nil, err
	}

	aviWriter, err := mjpeg.New(videoAVIPath, int32(*browserBound.Width), int32(*browserBound.Height), int32(fps))
	if err != nil {
		return nil, err
	}

	go p.EachEvent(func(e *proto.PageScreencastFrame) {
		if len(*videoFrames) >= MaxVideoFrame {
			fmt.Println("Max video frames reach")
			return
		}

		err := proto.PageScreencastFrameAck{
			SessionID: e.SessionID,
		}.Call(p)
		if err != nil {
			fmt.Println("ScreencastFrameAck err:", err)
		}

		*videoFrames = append(*videoFrames, VideoFrame{
			Data:      e.Data,
			Timestamp: e.Metadata.Timestamp.Time(),
		})
	})()

	return &aviWriter, nil
}

//ScreenCastStart start screen cast event for video recording
func (p *Page) ScreenCastStart(JPEGQuality int) error {
	everyNthFrame := 1
	return proto.PageStartScreencast{
		Format:        proto.PageStartScreencastFormatJpeg,
		Quality:       &JPEGQuality,
		EveryNthFrame: &everyNthFrame,
	}.Call(p)
}

//ScreenCastStopAvi stop screen cast event and save videoframes data in an avi video file
func (p *Page) ScreenCastStopAvi(aviWriter *mjpeg.AviWriter, videoFrames *[]VideoFrame, fps int) error {
	err := proto.PageStopScreencast{}.Call(p)
	if err != nil {
		return err
	}

	vfs := *videoFrames

	sort.Slice(vfs, func(i int, y int) bool {
		return vfs[i].Timestamp.Before(vfs[y].Timestamp)
	})

	//since screen cast event will not trigger if the page didn't change
	//So I need to append a stop frame so that it can copy the last frame data to fill the time
	vfs = append(vfs, VideoFrame{
		Timestamp: time.Now(),
	})

	//screen cast frames may not has the same fps, so convert to our fps
	for i, vf := range vfs {
		//fmt.Printf("frame %d, data %d, timestamp %v\n", i, len(vf.Data), vf.Timestamp)
		if i > 0 {
			dur := float64(vf.Timestamp.Sub(vfs[i-1].Timestamp).Nanoseconds())/float64(time.Second) + vfs[i-1].AccumDurationInSecond
			fc := (dur * float64(fps)) + vfs[i-1].FrameCntRemaining
			fci := float64(int64(fc))
			vfs[i-1].DurationInSecond = dur
			vfs[i-1].FrameCnt = fci

			// if frame count = 0, save the duration to current frame's AccumDurationInSecond
			if fci == float64(0) {
				vfs[i].AccumDurationInSecond += dur
				continue
			}

			fcr := fc - fci
			// save the remaining frame count portion to current frame
			if fcr > 0 {
				vfs[i].FrameCntRemaining += fcr
			}
		}
	}

	total := 0
	aw := *aviWriter
	for _, vf := range vfs {
		//fmt.Printf("frame %d, data %d, duration %v,\t\tdurationAcc %v,\t\tframeCnt %v\t\tframeCntR %v\n", j, len(vf.Data), vf.DurationInSecond, vf.AccumDurationInSecond, vf.FrameCnt, vf.FrameCntRemaining)
		if vf.FrameCnt > 0 {
			for i := int64(0); i < int64(vf.FrameCnt); i++ {
				err = aw.AddFrame(vf.Data)
				if err != nil {
					return err
				}
				total++
			}
		}
	}
	err = aw.Close()
	if err != nil {
		return err
	}

	fmt.Println("totalFrameCnt: ", total)

	return nil
}

//ScreenCastRecordMp4 listen PageScreenCastFrame and convert it directly into MP4 using ffmpeg
func (p *Page) ScreenCastRecordMp4(videoFrames *[]VideoFrame) error {
	go p.EachEvent(func(e *proto.PageScreencastFrame) {
		if len(*videoFrames) >= MaxVideoFrame {
			fmt.Println("Max video frames reach")
			return
		}

		err := proto.PageScreencastFrameAck{
			SessionID: e.SessionID,
		}.Call(p)
		if err != nil {
			fmt.Println("ScreencastFrameAck err:", err)
		}

		*videoFrames = append(*videoFrames, VideoFrame{
			Data:      e.Data,
			Timestamp: e.Metadata.Timestamp.Time(),
		})
	})()

	return nil
}

//ScreenCastStopMp4 stop screen cast event and use ffmpeg to create mp4 from videoFrame data
func (p *Page) ScreenCastStopMp4(videoFrames *[]VideoFrame, outputFile string, fps int) error {
	err := proto.PageStopScreencast{}.Call(p)
	if err != nil {
		return err
	}

	vfs := *videoFrames

	sort.Slice(vfs, func(i int, y int) bool {
		return vfs[i].Timestamp.Before(vfs[y].Timestamp)
	})

	//since screen cast event will not trigger if the page didn't change
	//So I need to append a stop frame so that it can copy the last frame data to fill the time
	vfs = append(vfs, VideoFrame{
		Timestamp: time.Now(),
	})

	//screen cast frames may not has the same fps, so convert to our fps
	for i, vf := range vfs {
		if i > 0 {
			dur := float64(vf.Timestamp.Sub(vfs[i-1].Timestamp).Nanoseconds())/float64(time.Second) + vfs[i-1].AccumDurationInSecond
			fc := (dur * float64(fps)) + vfs[i-1].FrameCntRemaining
			fci := float64(int64(fc))
			vfs[i-1].DurationInSecond = dur
			vfs[i-1].FrameCnt = fci

			// if frame count = 0, save the duration to current frame's AccumDurationInSecond
			if fci == float64(0) {
				vfs[i].AccumDurationInSecond += dur
				continue
			}

			fcr := fc - fci
			// save the remaining frame count portion to current frame
			if fcr > 0 {
				vfs[i].FrameCntRemaining += fcr
			}
		}
	}

	//cat $(find . -maxdepth 1 -name '*.png' -print | sort | tail -10) | ffmpeg -framerate 25 -i - -vf format=yuv420p -movflags +faststart output.mp4

	cmd := exec.Command("ffmpeg",
		"-y", // Yes to all
		"-f", "image2pipe",
		"-r", strconv.Itoa(fps),
		"-i", "pipe:0", // take stdin as input
		"-an",
		"-vf", "format=yuv420p",
		"-vsync", "1",
		"-movflags", "+faststart",
		outputFile, // output
	)
	cmd.Stderr = os.Stderr // bind log stream to stderr

	stdin, err := cmd.StdinPipe() // Open stdin pip
	if err != nil {
		return err
	}

	err = cmd.Start() // Start a process on another goroutine
	if err != nil {
		return err
	}

	total := 0
	for _, vf := range vfs {
		//fmt.Printf("frame %d, data %d, duration %v,\t\tdurationAcc %v,\t\tframeCnt %v\t\tframeCntR %v\n", j, len(vf.Data), vf.DurationInSecond, vf.AccumDurationInSecond, vf.FrameCnt, vf.FrameCntRemaining)
		if vf.FrameCnt > 0 {
			for i := int64(0); i < int64(vf.FrameCnt); i++ {
				_, err = stdin.Write(vf.Data)
				if err != nil {
					return err
				}
				total++
			}
		}
	}
	fmt.Printf("totalFrameCnt: %v\n\n", total)

	err = stdin.Close()
	if err != nil {
		return err
	}

	err = cmd.Wait() // wait until ffmpeg finish
	if err != nil {
		return err
	}

	return nil
}
