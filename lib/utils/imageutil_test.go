package utils

import (
	"bytes"
	"github.com/go-rod/rod/lib/proto"
	"github.com/ysmood/got"
	"image"
	"testing"
)

var setup = got.Setup(nil)

func TestSplicePngVertical(t *testing.T) {
	g := setup(t)
	a := image.NewRGBA(image.Rect(0, 0, 1000, 200))
	b := image.NewRGBA(image.Rect(0, 0, 1000, 300))

	format := proto.PageCaptureScreenshotFormatJpeg

	processor, err := NewImgProcessor(format)
	if err != nil {
		g.Err(err)
	}
	aBs, _ := processor.Encode(a, nil)
	bBs, _ := processor.Encode(b, nil)

	bs, err := SplicePngVertical([]ImgWithBox{
		{Img: aBs},
		{Img: bBs},
	}, proto.PageCaptureScreenshotFormatJpeg, nil)
	if err != nil {
		g.Err(err)
	}

	img, err := processor.Decode(bytes.NewBuffer(bs))
	if err != nil {
		g.Err(err)
	}

	g.Eq(img.Bounds().Dy(), 500)
	g.Eq(img.Bounds().Dx(), 1000)
}
