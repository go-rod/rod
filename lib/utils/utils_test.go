package utils_test

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-rod/rod/lib/utils"
	"github.com/ysmood/got"
)

type C struct {
	got.Assertion
}

func Test(t *testing.T) {
	got.Each(t, C{})
}

func (c C) TestLog() {
	utils.Log(func(msg ...interface{}) {}).Println()
	utils.LoggerQuiet.Println()
}

func (c C) TestE() {
	utils.E(nil)

	c.Panic(func() {
		utils.E(errors.New("err"))
	})
}

func (c C) STemplate() {
	out := utils.S(
		"{{.a}} {{.b}} {{.c.A}} {{d}}",
		"a", "<value>",
		"b", 10,
		"c", struct{ A string }{"ok"},
		"d", func() string {
			return "ok"
		},
	)
	c.Eq("<value> 10 ok ok", out)
}

func (c C) GenerateRandomString() {
	v := utils.RandString(10)
	raw, _ := hex.DecodeString(v)
	c.Len(raw, 10)
}

func (c C) Mkdir() {
	p := filepath.Join(c.Testable.(*testing.T).TempDir(), "t")
	c.E(utils.Mkdir(p))
}

func (c C) OutputString() {
	p := "tmp/" + utils.RandString(10)

	_ = utils.OutputFile(p, p)

	s, err := utils.ReadString(p)

	if err != nil {
		panic(err)
	}

	c.Eq(s, p)
}

func (c C) OutputBytes() {
	p := "tmp/" + utils.RandString(10)

	_ = utils.OutputFile(p, []byte("test"))

	s, err := utils.ReadString(p)

	if err != nil {
		panic(err)
	}

	c.Eq(s, "test")
}

func (c C) OutputStream() {
	p := "tmp/" + utils.RandString(10)
	b := bytes.NewBufferString("test")

	_ = utils.OutputFile(p, b)

	s, err := utils.ReadString(p)

	if err != nil {
		panic(err)
	}

	c.Eq("test", s)
}

func (c C) OutputJSONErr() {
	p := "tmp/" + utils.RandString(10)

	c.Panic(func() {
		_ = utils.OutputFile(p, make(chan struct{}))
	})
}

func (c C) Sleep() {
	utils.Sleep(0.01)
}

func (c C) All() {
	utils.All(func() {
		fmt.Println("one")
	}, func() {
		fmt.Println("two")
	})()
}

func (c C) Pause() {
	go utils.Pause()
}

func (c C) BackoffSleeperWakeNow() {
	c.E(utils.BackoffSleeper(0, 0, nil)(context.Background()))
}

func (c C) Retry() {
	count := 0
	s1 := utils.BackoffSleeper(1, 5, nil)

	err := utils.Retry(context.Background(), s1, func() (bool, error) {
		if count > 5 {
			return true, io.EOF
		}
		count++
		return false, nil
	})

	c.Eq(err.Error(), io.EOF.Error())
}

func (c C) RetryCancel() {
	ctx, cancel := context.WithCancel(context.Background())
	go cancel()
	s := utils.BackoffSleeper(time.Second, time.Second, nil)

	err := utils.Retry(ctx, s, func() (bool, error) {
		return false, nil
	})

	c.Eq(err.Error(), context.Canceled.Error())
}

func (c C) CountSleeperErr() {
	ctx := context.Background()
	s := utils.CountSleeper(5)
	for i := 0; i < 5; i++ {
		_ = s(ctx)
	}
	c.Err(s(ctx))
}

func (c C) CountSleeperCancel() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s := utils.CountSleeper(5)
	c.Eq(s(ctx).Error(), context.Canceled.Error())
}

func (c C) MustToJSON() {
	c.Eq(utils.Dump("a", 10), `"a" 10`)
	c.Eq(`{"a":1}`, utils.MustToJSON(map[string]int{"a": 1}))
}

func (c C) FileExists() {
	c.Eq(false, utils.FileExists("."))
	c.Eq(true, utils.FileExists("utils.go"))
	c.Eq(false, utils.FileExists(utils.RandString(8)))
}

func (c C) Exec() {
	utils.Exec("echo")
}

func (c C) Serve() {
	u, mux, close := utils.Serve("")
	defer close()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		panic("err")
	})

	res, err := http.Get(u)
	c.E(err)

	c.Eq(http.StatusBadRequest, res.StatusCode)
}

type errReader struct {
	err error
}

func (r *errReader) Read(p []byte) (n int, err error) {
	return 0, r.err
}

func (c C) EscapeGoString() {
	c.Eq("`` + \"`\" + `test` + \"`\" + ``", utils.EscapeGoString("`test`"))
}

func (c C) IdleCounter() {
	utils.All(func() {
		ct := utils.NewIdleCounter(100 * time.Millisecond)

		ct.Add()
		go func() {
			ct.Add()
			time.Sleep(300 * time.Millisecond)
			ct.Done()
			ct.Done()
		}()

		ctx, cancel := context.WithCancel(context.Background())

		start := time.Now()
		ct.Wait(ctx)
		d := time.Since(start)
		c.Gt(d, 400*time.Millisecond)
		c.Lt(d, 450*time.Millisecond)

		c.Panic(func() {
			ct.Done()
		})

		cancel()
		ct.Wait(ctx)
	}, func() {
		ct := utils.NewIdleCounter(100 * time.Millisecond)
		start := time.Now()
		ct.Wait(context.Background())
		c.Lt(time.Since(start), 150*time.Millisecond)
	}, func() {
		ct := utils.NewIdleCounter(0)
		start := time.Now()
		ct.Wait(context.Background())
		c.Lt(time.Since(start), 10*time.Millisecond)
	})()
}
