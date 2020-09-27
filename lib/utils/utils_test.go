package utils_test

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-rod/rod/lib/utils"
	"github.com/stretchr/testify/assert"
)

type T = testing.T

func TestErr(t *T) {
	utils.E(nil)

	assert.Panics(t, func() {
		utils.E(errors.New("err"))
	})
}

func TestDump(t *T) {
	assert.Equal(t, "{\n  \"a\": \"<b>\"\n}", utils.SDump(map[string]string{"a": "<b>"}))
	assert.Equal(t, "10", utils.SDump(json.RawMessage("10")))
	utils.Dump("")
}

func TestSTemplate(t *T) {
	out := utils.S(
		"{{.a}} {{.b}} {{.c.A}} {{d}}",
		"a", "<value>",
		"b", 10,
		"c", struct{ A string }{"ok"},
		"d", func() string {
			return "ok"
		},
	)
	assert.Equal(t, "<value> 10 ok ok", out)
}

func TestGenerateRandomString(t *T) {
	v := utils.RandString(10)
	raw, _ := hex.DecodeString(v)
	assert.Len(t, raw, 10)
}

func TestMkdir(t *testing.T) {
	p := filepath.Join(t.TempDir(), "t")
	utils.E(utils.Mkdir(p))
}

func TestOutputString(t *testing.T) {
	p := "tmp/" + utils.RandString(10)

	_ = utils.OutputFile(p, p)

	c, err := utils.ReadString(p)

	if err != nil {
		panic(err)
	}

	assert.Equal(t, c, p)
}

func TestOutputBytes(t *testing.T) {
	p := "tmp/" + utils.RandString(10)

	_ = utils.OutputFile(p, []byte("test"))

	c, err := utils.ReadString(p)

	if err != nil {
		panic(err)
	}

	assert.Equal(t, c, "test")
}

func TestOutputStream(t *testing.T) {
	p := "tmp/" + utils.RandString(10)
	b := bytes.NewBufferString("test")

	_ = utils.OutputFile(p, b)

	c, err := utils.ReadString(p)

	if err != nil {
		panic(err)
	}

	assert.Equal(t, "test", c)
}

func TestOutputJSONErr(t *testing.T) {
	p := "tmp/" + utils.RandString(10)

	assert.Panics(t, func() {
		_ = utils.OutputFile(p, make(chan struct{}))
	})
}

func TestSleep(t *T) {
	utils.Sleep(0.01)
}

func TestAll(t *T) {
	utils.All(func() {
		fmt.Println("one")
	}, func() {
		fmt.Println("two")
	})()
}

func TestPause(t *T) {
	go utils.Pause()
}

func TestBackoffSleeperWakeNow(t *T) {
	utils.E(utils.BackoffSleeper(0, 0, nil)(context.Background()))
}

func TestRetry(t *T) {
	count := 0
	s1 := utils.BackoffSleeper(1, 5, nil)

	err := utils.Retry(context.Background(), s1, func() (bool, error) {
		if count > 5 {
			return true, io.EOF
		}
		count++
		return false, nil
	})

	assert.EqualError(t, err, io.EOF.Error())
}

func TestRetryCancel(t *T) {
	ctx, cancel := context.WithCancel(context.Background())
	go cancel()
	s := utils.BackoffSleeper(time.Second, time.Second, nil)

	err := utils.Retry(ctx, s, func() (bool, error) {
		return false, nil
	})

	assert.EqualError(t, err, context.Canceled.Error())
}

func TestCountSleeperErr(t *T) {
	ctx := context.Background()
	s := utils.CountSleeper(5)
	for i := 0; i < 5; i++ {
		_ = s(ctx)
	}
	assert.Errorf(t, s(ctx), "max sleep count")
}

func TestCountSleeperCancel(t *T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s := utils.CountSleeper(5)
	assert.Errorf(t, s(ctx), context.Canceled.Error())
}

func TestMustToJSON(t *T) {
	assert.Equal(t, `{"a":1}`, utils.MustToJSON(map[string]int{"a": 1}))
}

func TestFileExists(t *T) {
	assert.Equal(t, false, utils.FileExists("."))
	assert.Equal(t, true, utils.FileExists("utils.go"))
	assert.Equal(t, false, utils.FileExists(utils.RandString(8)))
}

func TestExec(t *T) {
	utils.Exec("echo")
}

func TestServe(t *T) {
	u, mux, close := utils.Serve("")
	defer close()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		panic("err")
	})

	res, err := http.Get(u)
	utils.E(err)

	assert.Equal(t, http.StatusBadRequest, res.StatusCode)
}

type errReader struct {
	err error
}

func (r *errReader) Read(p []byte) (n int, err error) {
	return 0, r.err
}

func TestReader(t *T) {
	utils.MustReadJSON(bytes.NewBufferString(""))

	_, err := utils.ReadJSON(&errReader{err: errors.New("err")})
	assert.Error(t, err)

	utils.MustReadString(bytes.NewBufferString(""))

	_, err = utils.ReadJSONPathAsString(bytes.NewBufferString("{}"), "")
	assert.Nil(t, err)

	_, err = utils.ReadJSONPathAsString(&errReader{err: errors.New("err")}, "")
	assert.Error(t, err)
}

func TestEscapeGoString(t *testing.T) {
	assert.Equal(t, "`` + \"`\" + `test` + \"`\" + ``", utils.EscapeGoString("`test`"))
}

func TestIdleCounter(t *testing.T) {
	utils.All(func() {
		c := utils.NewIdleCounter(100 * time.Millisecond)

		go func() {
			c.Add()
			time.Sleep(300 * time.Millisecond)
			c.Done()
		}()

		ctx, cancel := context.WithCancel(context.Background())

		start := time.Now()
		c.Wait(ctx)
		d := time.Since(start)
		assert.Greater(t, d, 400*time.Millisecond)
		assert.Less(t, d, 450*time.Millisecond)

		assert.Panics(t, func() {
			c.Done()
		})

		cancel()
		c.Wait(ctx)
	}, func() {
		c := utils.NewIdleCounter(100 * time.Millisecond)
		start := time.Now()
		c.Wait(context.Background())
		assert.Less(t, time.Since(start), 150*time.Millisecond)
	}, func() {
		c := utils.NewIdleCounter(0)
		start := time.Now()
		c.Wait(context.Background())
		assert.Less(t, time.Since(start), 10*time.Millisecond)
	})()
}
