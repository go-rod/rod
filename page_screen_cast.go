package rod

import (
	"fmt"
	"sort"
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
	browserBound, errorBrowserBound := p.GetWindow()

	if errorBrowserBound != nil {
		return nil, errorBrowserBound
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

func (p *Page) ScreenCastStartAvi(JPEGQuality int) error {
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
