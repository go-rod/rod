package utils

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"

	"github.com/go-rod/rod/lib/proto"
)

// ImgWithBox is a image with a box, if the box is nil, it means the whole image.
type ImgWithBox struct {
	Img []byte
	Box *image.Rectangle
}

// ImgOption is the option for image processing.
type ImgOption struct {
	Quality int
}

// ImgProcessor is the interface for image processing.
type ImgProcessor interface {
	Encode(img image.Image, opt *ImgOption) ([]byte, error)
	Decode(file io.Reader) (image.Image, error)
}

type jpegProcessor struct{}

func (p jpegProcessor) Encode(img image.Image, opt *ImgOption) ([]byte, error) {
	var buf bytes.Buffer
	var jpegOpt *jpeg.Options
	if opt != nil {
		jpegOpt = &jpeg.Options{Quality: opt.Quality}
	}
	err := jpeg.Encode(&buf, img, jpegOpt)
	return buf.Bytes(), err
}

func (p jpegProcessor) Decode(file io.Reader) (image.Image, error) {
	return jpeg.Decode(file)
}

type pngProcessor struct{}

func (p pngProcessor) Encode(img image.Image, _ *ImgOption) ([]byte, error) {
	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	return buf.Bytes(), err
}

func (p pngProcessor) Decode(file io.Reader) (image.Image, error) {
	return png.Decode(file)
}

// NewImgProcessor create a ImgProcessor by the format.
func NewImgProcessor(format proto.PageCaptureScreenshotFormat) (ImgProcessor, error) {
	switch format {
	case proto.PageCaptureScreenshotFormatJpeg:
		return &jpegProcessor{}, nil
	case "", proto.PageCaptureScreenshotFormatPng:
		return &pngProcessor{}, nil
	default:
		return nil, fmt.Errorf("not support format: %v", format)
	}
}

// SplicePngVertical splice png vertically, if there is only one image, it will return the image directly.
// Only support png and jpeg format yet, webP is not supported because no suitable processing
// library was found in golang.
func SplicePngVertical(files []ImgWithBox, format proto.PageCaptureScreenshotFormat, opt *ImgOption) ([]byte, error) {
	if len(files) == 0 {
		return nil, nil
	}
	if len(files) == 1 {
		return files[0].Img, nil
	}

	var width, height int

	processor, err := NewImgProcessor(format)
	if err != nil {
		return nil, err
	}

	var images []image.Image
	for _, file := range files {
		img, err := processor.Decode(bytes.NewReader(file.Img))
		if err != nil {
			return nil, err
		}

		images = append(images, img)
		if file.Box != nil {
			width = file.Box.Dx()
			height += file.Box.Dy()
		} else {
			width = img.Bounds().Dx()
			height += img.Bounds().Dy()
		}
	}

	spliceImg := image.NewRGBA(image.Rect(0, 0, width, height))

	var destY int
	for i, file := range files {
		img := images[i]
		bounds := img.Bounds()

		if file.Box != nil {
			bounds = *file.Box
		}
		start := bounds.Min
		end := bounds.Max
		for y := start.Y; y < end.Y; y++ {
			for x := start.X; x < end.X; x++ {
				color := img.At(x, y)
				spliceImg.Set(x, y-start.Y+destY, color)
			}
		}

		destY += bounds.Dy()
	}

	bs, err := processor.Encode(spliceImg, opt)
	if err != nil {
		return nil, err
	}

	return bs, nil
}
