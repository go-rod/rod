package utils_test

import (
	"bytes"
	"encoding/hex"
	"errors"
	"image"
	"image/jpeg"
	"image/png"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-rod/rod/lib/utils"
	"github.com/ysmood/got"
)

type T struct {
	got.G
}

func Test(t *testing.T) {
	got.Each(t, T{})
}

func (t T) TestLog() {
	var res []interface{}
	lg := utils.Log(func(msg ...interface{}) { res = append(res, msg[0]) })
	lg.Println("ok")
	t.Eq(res[0], "ok")

	utils.LoggerQuiet.Println()

	utils.MultiLogger(lg, lg).Println("ok")
	t.Eq(res, []interface{}{"ok", "ok", "ok"})
}

func (t T) TestE() {
	utils.E(nil)

	t.Panic(func() {
		utils.E(errors.New("err"))
	})
}

func (t T) STemplate() {
	out := utils.S(
		"{{.a}} {{.b}} {{.c.A}} {{d}}",
		"a", "<value>",
		"b", 10,
		"c", struct{ A string }{"ok"},
		"d", func() string {
			return "ok"
		},
	)
	t.Eq("<value> 10 ok ok", out)
}

func (t T) GenerateRandomString() {
	v := utils.RandString(10)
	raw, _ := hex.DecodeString(v)
	t.Len(raw, 10)
}

func (t T) Mkdir() {
	p := filepath.Join(t.Testable.(*testing.T).TempDir(), "t")
	t.E(utils.Mkdir(p))
}

func (t T) OutputString() {
	p := "tmp/" + t.Srand(16)

	_ = utils.OutputFile(p, p)

	s, err := utils.ReadString(p)

	if err != nil {
		panic(err)
	}

	t.Eq(s, p)
}

func (t T) OutputBytes() {
	p := "tmp/" + t.Srand(16)

	_ = utils.OutputFile(p, []byte("test"))

	s, err := utils.ReadString(p)

	if err != nil {
		panic(err)
	}

	t.Eq(s, "test")
}

func (t T) OutputStream() {
	p := "tmp/" + t.Srand(16)
	b := bytes.NewBufferString("test")

	_ = utils.OutputFile(p, b)

	s, err := utils.ReadString(p)

	if err != nil {
		panic(err)
	}

	t.Eq("test", s)
}

func (t T) OutputJSONErr() {
	p := "tmp/" + t.Srand(16)

	t.Panic(func() {
		_ = utils.OutputFile(p, make(chan struct{}))
	})
}

func (t T) Sleep() {
	utils.Sleep(0.01)
}

func (t T) All() {
	c := t.Count(3)
	utils.All(c, c, c)()
}

func (t T) Pause() {
	go utils.Pause()
}

func (t T) MustToJSON() {
	t.Eq(utils.Dump("a", 10), `"a" 10`)
	t.Eq(`{"a":1}`, utils.MustToJSON(map[string]int{"a": 1}))
}

func (t T) FileExists() {
	t.Eq(false, utils.FileExists("."))
	t.Eq(true, utils.FileExists("utils.go"))
	t.Eq(false, utils.FileExists(t.Srand(16)))
}

func (t T) ExecErr() {
	t.Panic(func() {
		utils.ExecLine("")
	})
}

func (t T) EscapeGoString() {
	t.Eq("`` + \"`\" + `test` + \"`\" + ``", utils.EscapeGoString("`test`"))
}

func (t T) IdleCounter() {
	utils.All(func() {
		ct := utils.NewIdleCounter(100 * time.Millisecond)

		ct.Add()
		go func() {
			ct.Add()
			time.Sleep(300 * time.Millisecond)
			ct.Done()
			ct.Done()
		}()

		ctx := t.Context()

		start := time.Now()
		ct.Wait(ctx)
		d := time.Since(start)
		t.Gt(d, 400*time.Millisecond)
		t.Lt(d, 450*time.Millisecond)

		t.Panic(func() {
			ct.Done()
		})

		ctx.Cancel()
		ct.Wait(ctx)
	}, func() {
		ct := utils.NewIdleCounter(100 * time.Millisecond)
		start := time.Now()
		ct.Wait(t.Context())
		t.Lt(time.Since(start), 150*time.Millisecond)
	}, func() {
		ct := utils.NewIdleCounter(0)
		start := time.Now()
		ct.Wait(t.Context())
		t.Lt(time.Since(start), 10*time.Millisecond)
	})()
}

func (t T) CropImage() {
	img := image.NewNRGBA(image.Rect(0, 0, 100, 100))

	t.Err(utils.CropImage(nil, 0, 0, 0, 0, 0))

	bin := bytes.NewBuffer(nil)
	t.E(png.Encode(bin, img))
	t.E(utils.CropImage(bin.Bytes(), 0, 10, 10, 30, 30))

	bin = bytes.NewBuffer(nil)
	t.E(jpeg.Encode(bin, img, &jpeg.Options{Quality: 80}))
	t.E(utils.CropImage(bin.Bytes(), 0, 10, 10, 30, 30))
}
