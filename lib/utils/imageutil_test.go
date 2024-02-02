package utils

import (
	"bytes"
	"image"
	"testing"

	"github.com/go-rod/rod/lib/proto"
	"github.com/ysmood/got"
)

var setup = got.Setup(nil)

func TestSplicePngVertical(t *testing.T) {
	g := setup(t)
	a := image.NewRGBA(image.Rect(0, 0, 1000, 200))
	b := image.NewRGBA(image.Rect(0, 0, 1000, 300))

	t.Run("jpeg", func(t *testing.T) {
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
		}, format, nil)
		g.E(err)

		img, err := processor.Decode(bytes.NewBuffer(bs))
		g.E(err)

		g.Eq(img.Bounds().Dy(), 500)
		g.Eq(img.Bounds().Dx(), 1000)
	})
	t.Run("jpegWithOptions", func(t *testing.T) {
		format := proto.PageCaptureScreenshotFormatJpeg
		processor, err := NewImgProcessor(format)
		g.E(err)

		aBs, _ := processor.Encode(a, nil)
		bBs, _ := processor.Encode(b, nil)

		bs, err := SplicePngVertical([]ImgWithBox{
			{Img: aBs},
			{Img: bBs},
		}, format, &ImgOption{
			Quality: 10,
		})
		g.E(err)

		img, err := processor.Decode(bytes.NewBuffer(bs))
		g.E(err)

		g.Eq(img.Bounds().Dy(), 500)
		g.Eq(img.Bounds().Dx(), 1000)
	})
	t.Run("jpegWithBox", func(t *testing.T) {
		format := proto.PageCaptureScreenshotFormatJpeg
		processor, err := NewImgProcessor(format)
		g.E(err)

		aBs, _ := processor.Encode(a, nil)
		bBs, _ := processor.Encode(b, nil)

		bs, err := SplicePngVertical([]ImgWithBox{
			{
				Img: aBs,
				Box: &image.Rectangle{
					Max: image.Point{
						X: a.Bounds().Dx(),
						Y: 100,
					},
				},
			},
			{Img: bBs},
		}, format, nil)
		g.E(err)

		img, err := processor.Decode(bytes.NewBuffer(bs))
		g.E(err)

		g.Eq(img.Bounds().Dy(), 400)
		g.Eq(img.Bounds().Dx(), 1000)
	})
	t.Run("errorEncode", func(t *testing.T) {
		format := proto.PageCaptureScreenshotFormatPng
		processor, err := NewImgProcessor(format)
		g.E(err)

		aBs, _ := processor.Encode(a, nil)
		bBs, _ := processor.Encode(b, nil)

		_, err = SplicePngVertical([]ImgWithBox{
			{
				Img: aBs,
				Box: &image.Rectangle{},
			},
			{
				Img: bBs,
				Box: &image.Rectangle{},
			},
		}, format, nil)
		// invalid image size: 0x0
		g.Err(err)
	})
	t.Run("noFile", func(t *testing.T) {
		_, err := SplicePngVertical(nil, "", nil)
		g.E(err)
	})
	t.Run("oneFile", func(t *testing.T) {
		bs, err := SplicePngVertical([]ImgWithBox{
			{Img: []byte{1}},
		}, "", nil)
		g.E(err)
		g.Eq(1, len(bs))
	})
	t.Run("unsupportedFormat", func(t *testing.T) {
		_, err := SplicePngVertical([]ImgWithBox{
			{Img: []byte{1}},
			{Img: []byte{1}},
		}, "gif", nil)
		g.Err(err)
	})
	t.Run("errorFile", func(t *testing.T) {
		_, err := SplicePngVertical([]ImgWithBox{
			{Img: []byte{1}},
			{Img: []byte{1}},
		}, "", nil)
		g.Err(err)
	})
}

func TestNewImgProcessor(t *testing.T) {
	g := setup(t)
	type args struct {
		format proto.PageCaptureScreenshotFormat
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "jpeg",
			args: args{
				format: proto.PageCaptureScreenshotFormatJpeg,
			},
			wantErr: false,
		},
		{
			name: "default",
			args: args{
				format: "",
			},
			wantErr: false,
		},
		{
			name: "png",
			args: args{
				format: proto.PageCaptureScreenshotFormatPng,
			},
			wantErr: false,
		},
		{
			name: "webP",
			args: args{
				/* cspell: disable-next-line */
				format: proto.PageCaptureScreenshotFormatWebp,
			},
			wantErr: true,
		},
	}

	a := image.NewRGBA(image.Rect(0, 0, 1000, 200))
	// errImg := image.NewRGBA(image.Rect(0, 0, 0, 0))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor, err := NewImgProcessor(tt.args.format)
			if tt.wantErr {
				g.Eq(err != nil, tt.wantErr)
			}
			if err != nil {
				return
			}
			buf, err := processor.Encode(a, nil)
			if err != nil {
				g.Err(err)
			}
			img, err := processor.Decode(bytes.NewBuffer(buf))
			if err != nil {
				g.Err(err)
			}

			g.Eq(1000, img.Bounds().Dx())
			g.Eq(200, img.Bounds().Dy())

			_, err = processor.Decode(bytes.NewBuffer(nil))
			g.Err(err)
		})
	}
}
