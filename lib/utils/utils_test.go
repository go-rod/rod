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

var setup = got.Setup(nil)

func TestTestLog(t *testing.T) {
	g := setup(t)

	var res []interface{}
	lg := utils.Log(func(msg ...interface{}) { res = append(res, msg[0]) })
	lg.Println("ok")
	g.Eq(res[0], "ok")

	utils.LoggerQuiet.Println()

	utils.MultiLogger(lg, lg).Println("ok")
	g.Eq(res, []interface{}{"ok", "ok", "ok"})
}

func TestTestE(t *testing.T) {
	g := setup(t)

	utils.E(nil)

	g.Panic(func() {
		utils.E(errors.New("err"))
	})
}

func TestSTemplate(t *testing.T) {
	g := setup(t)

	out := utils.S(
		"{{.a}} {{.b}} {{.c.A}} {{d}}",
		"a", "<value>",
		"b", 10,
		"c", struct{ A string }{"ok"},
		"d", func() string {
			return "ok"
		},
	)
	g.Eq("<value> 10 ok ok", out)
}

func TestGenerateRandomString(t *testing.T) {
	g := setup(t)

	v := utils.RandString(10)
	raw, _ := hex.DecodeString(v)
	g.Len(raw, 10)
}

func TestMkdir(t *testing.T) {
	g := setup(t)

	p := filepath.Join(g.Testable.(*testing.T).TempDir(), "t")
	g.E(utils.Mkdir(p))
}

func TestAbsolutePaths(t *testing.T) {
	g := setup(t)

	p := utils.AbsolutePaths([]string{"utils.go"})
	g.Has(p[0], filepath.FromSlash("/utils.go"))
}

func TestOutputString(t *testing.T) {
	g := setup(t)

	p := "tmp/" + g.RandStr(16)

	_ = utils.OutputFile(p, p)

	s, err := utils.ReadString(p)
	if err != nil {
		panic(err)
	}

	g.Eq(s, p)
}

func TestOutputBytes(t *testing.T) {
	g := setup(t)

	p := "tmp/" + g.RandStr(16)

	_ = utils.OutputFile(p, []byte("test"))

	s, err := utils.ReadString(p)
	if err != nil {
		panic(err)
	}

	g.Eq(s, "test")
}

func TestOutputStream(t *testing.T) {
	g := setup(t)

	p := "tmp/" + g.RandStr(16)
	b := bytes.NewBufferString("test")

	_ = utils.OutputFile(p, b)

	s, err := utils.ReadString(p)
	if err != nil {
		panic(err)
	}

	g.Eq("test", s)
}

func TestOutputJSONErr(t *testing.T) {
	g := setup(t)

	p := "tmp/" + g.RandStr(16)

	g.Panic(func() {
		_ = utils.OutputFile(p, make(chan struct{}))
	})
}

func TestSleep(_ *testing.T) {
	utils.Sleep(0.01)
}

func TestAll(t *testing.T) {
	g := setup(t)

	c := g.Count(3)
	utils.All(c, c, c)()
}

func TestPause(_ *testing.T) {
	go utils.Pause()
}

func TestMustToJSON(t *testing.T) {
	g := setup(t)

	g.Eq(utils.Dump("a", 10), `"a" 10`)
	g.Eq(`{"a":1}`, utils.MustToJSON(map[string]int{"a": 1}))
}

func TestFileExists(t *testing.T) {
	g := setup(t)

	g.Eq(false, utils.FileExists("."))
	g.Eq(true, utils.FileExists("utils.go"))
	g.Eq(false, utils.FileExists(g.RandStr(16)))
}

func TestExec(t *testing.T) {
	g := setup(t)

	g.Has(utils.Exec("go version"), "go version")
}

func TestExecErr(t *testing.T) {
	g := setup(t)

	g.Panic(func() {
		utils.Exec("")
	})
	g.Panic(func() {
		utils.Exec(g.RandStr(16))
	})
	g.Panic(func() {
		utils.ExecLine(false, "", "")
	})
}

func TestFormatCLIArgs(t *testing.T) {
	g := setup(t)

	g.Eq(utils.FormatCLIArgs([]string{"ab c", "abc"}), `"ab c" abc`)
}

func TestEscapeGoString(t *testing.T) {
	g := setup(t)

	g.Eq("`` + \"`\" + `test` + \"`\" + ``", utils.EscapeGoString("`test`"))
}

func TestIdleCounter(t *testing.T) {
	g := setup(t)

	utils.All(func() {
		ct := utils.NewIdleCounter(100 * time.Millisecond)

		ct.Add()
		go func() {
			ct.Add()
			time.Sleep(300 * time.Millisecond)
			ct.Done()
			ct.Done()
		}()

		ctx := g.Context()

		start := time.Now()
		ct.Wait(ctx)
		d := time.Since(start)
		g.Gt(d, 400*time.Millisecond)
		g.Lt(d, 450*time.Millisecond)

		g.Panic(func() {
			ct.Done()
		})

		ctx.Cancel()
		ct.Wait(ctx)
	}, func() {
		ct := utils.NewIdleCounter(100 * time.Millisecond)
		start := time.Now()
		ct.Wait(g.Context())
		g.Lt(time.Since(start), 150*time.Millisecond)
	}, func() {
		ct := utils.NewIdleCounter(0)
		start := time.Now()
		ct.Wait(g.Context())
		g.Lt(time.Since(start), 10*time.Millisecond)
	})()
}

func TestCropImage(t *testing.T) {
	g := setup(t)

	img := image.NewNRGBA(image.Rect(0, 0, 100, 100))

	g.Err(utils.CropImage(nil, 0, 0, 0, 0, 0))

	bin := bytes.NewBuffer(nil)
	g.E(png.Encode(bin, img))
	g.E(utils.CropImage(bin.Bytes(), 0, 10, 10, 30, 30))

	bin = bytes.NewBuffer(nil)
	g.E(jpeg.Encode(bin, img, &jpeg.Options{Quality: 80}))
	g.E(utils.CropImage(bin.Bytes(), 0, 10, 10, 30, 30))
}
