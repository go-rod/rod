package rod

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"time"

	"github.com/TommyLeng/go-rod/lib/proto"
	"github.com/icza/mjpeg"
)

type VideoFrame struct {
	Data                  []byte
	Timestamp             time.Time
	DurationInSecond      float64
	AccumDurationInSecond float64
	FrameCnt              float64
	FrameCntRemaining     float64
}

// ScreenCastRecord listen PageScreenCastFrame and convert it directly into AVI Movie
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

	/*
		workingDirectory, errorWD := os.Getwd()
		if errorWD != nil {
			return nil, errorWD
		}

		matches, errorGlob := filepath.Glob(workingDirectory + "/*.idx_")
		if errorGlob != nil {
			return nil, errorGlob
		}

		for _, name := range matches {
			errRemove := os.Remove(name)
			if errRemove != nil {
				return nil, errRemove
			}
		}
	*/

	return &aviWriter, nil
}

func (p *Page) ScreenCastStart(JPEGQuality int) error {
	everyNthFrame := 1
	return proto.PageStartScreencast{
		Format:        proto.PageStartScreencastFormatJpeg,
		Quality:       &JPEGQuality,
		EveryNthFrame: &everyNthFrame,
	}.Call(p)
}

func (p *Page) ScreenCastStopAvi(aviWriter *mjpeg.AviWriter, videoFrames *[]VideoFrame, fps int) error {
	err := proto.PageStopScreencast{}.Call(p)
	if err != nil {
		return err
	}

	vfs := *videoFrames

	sort.Slice(vfs, func(i int, y int) bool {
		return vfs[i].Timestamp.Before(vfs[y].Timestamp)
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

	total := 0
	aw := *aviWriter
	for j, vf := range vfs {
		fmt.Printf("frame %d, data %d, duration %v,\t\tdurationAcc %v,\t\tframeCnt %v\t\tframeCntR %v\n", j, len(vf.Data), vf.DurationInSecond, vf.AccumDurationInSecond, vf.FrameCnt, vf.FrameCntRemaining)
		if vf.FrameCnt > 0 {
			for i := int64(0); i < int64(vf.FrameCnt); i++ {
				aw.AddFrame(vf.Data)
				total += 1
			}
		}
	}
	aw.Close()

	fmt.Println("totalFrameCnt: ", total)

	return nil
}

// ScreenCastRecord listen PageScreenCastFrame and convert it directly into MP4 using ffmpeg
func (p *Page) ScreenCastRecordMp4(videoFrames *[]VideoFrame) error {
	go p.EachEvent(func(e *proto.PageScreencastFrame) {
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

// use ffmpeg to create mp4 directly using pipe
func (p *Page) ScreenCastStopMp4UsingPipe(videoFrames *[]VideoFrame, outputFile string, fps int) error {
	err := proto.PageStopScreencast{}.Call(p)
	if err != nil {
		return err
	}

	vfs := *videoFrames

	sort.Slice(vfs, func(i int, y int) bool {
		return vfs[i].Timestamp.Before(vfs[y].Timestamp)
	})

	pr, pw := io.Pipe()

	//cat $(find . -maxdepth 1 -name '*.png' -print | sort | tail -10) | ffmpeg -framerate 25 -i - -vf format=yuv420p -movflags +faststart output.mp4

	cmd := exec.Command("ffmpeg", "-y", // Yes to all
		"-i", "pipe:0", // take stdin as input
		//"-filter:v", "fps=25",
		"-vf", "format=yuv420p",
		//"-movflags", "+faststart",
		outputFile, // output
	)
	cmd.Stderr = os.Stderr // bind log stream to stderr
	cmd.Stdin = pr

	err = cmd.Start() // Start a process on another goroutine
	if err != nil {
		return err
	}

	for _, v := range vfs {
		pw.Write(v.Data)
	}

	err = pw.Close()
	if err != nil {
		return err
	}
	err = pr.Close() // close the stdin, or ffmpeg will wait forever
	if err != nil {
		return err
	}

	err = cmd.Wait() // wait until ffmpeg finish
	if err != nil {
		return err
	}

	return nil
}

// use ffmpeg to create mp4
func (p *Page) ScreenCastStopMp4(videoFrames *[]VideoFrame, outputFile string) error {
	err := proto.PageStopScreencast{}.Call(p)
	if err != nil {
		return err
	}

	vfs := *videoFrames
	fps := 50
	fpsStr := strconv.FormatInt(int64(fps), 10)

	sort.Slice(vfs, func(i int, y int) bool {
		return vfs[i].Timestamp.Before(vfs[y].Timestamp)
	})

	//screen cast frames may not has the same fps, so convert to 50 fps, we feel better when make use 50 fps
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

	// I dunno why I cannot change frame rate to 50 fps, it still make use 25 fps, so I speed the video up using setpts=0.5*PTS in order to achieve 50 fps
	cmd := exec.Command("ffmpeg", "-y", // Yes to all
		"-i", "pipe:0", // take stdin as input
		//"-framerate", fpsStr,
		"-r", fpsStr,
		"-filter:v", "setpts=0.5*PTS",
		//"-f", "image2pipe", //not working
		//"-f", "rawvideo",
		//"-filter:v", "fps=" + fpsStr,
		//"-c:v", "libx264",
		//"-vf", "format=yuv420p",
		//"-movflags", "+faststart",
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
	for j, vf := range vfs {
		fmt.Printf("frame %d, data %d, duration %v,\t\tdurationAcc %v,\t\tframeCnt %v\t\tframeCntR %v\n", j, len(vf.Data), vf.DurationInSecond, vf.AccumDurationInSecond, vf.FrameCnt, vf.FrameCntRemaining)
		if vf.FrameCnt > 0 {
			for i := int64(0); i < int64(vf.FrameCnt); i++ {
				_, err = stdin.Write(vf.Data)
				if err != nil {
					return err
				}
				total += 1
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
