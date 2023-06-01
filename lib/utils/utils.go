// Package utils ...
package utils

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/ysmood/gson"
)

// TestEnvs for testing
var TestEnvs = map[string]string{
	"GODEBUG": "tracebackancestors=100",
}

// InContainer will be true if is inside container environment, such as docker
var InContainer = FileExists("/.dockerenv") || FileExists("/.containerenv")

// Noop does nothing
func Noop() {}

// Logger interface
type Logger interface {
	// Same as fmt.Printf
	Println(...interface{})
}

// Log type for Println
type Log func(msg ...interface{})

// Println interface
func (l Log) Println(msg ...interface{}) {
	l(msg...)
}

// LoggerQuiet does nothing
var LoggerQuiet Logger = Log(func(_ ...interface{}) {})

// MultiLogger is similar to https://golang.org/pkg/io/#MultiWriter
func MultiLogger(list ...Logger) Log {
	return Log(func(msg ...interface{}) {
		for _, lg := range list {
			lg.Println(msg...)
		}
	})
}

// Panic is the same as the built-in panic
var Panic = func(v interface{}) { panic(v) }

// E if the last arg is error, panic it
func E(args ...interface{}) []interface{} {
	err, ok := args[len(args)-1].(error)
	if ok {
		Panic(err)
	}
	return args
}

// S Template render, the params is key-value pairs
func S(tpl string, params ...interface{}) string {
	var out bytes.Buffer

	dict := map[string]interface{}{}
	fnDict := template.FuncMap{}

	l := len(params)
	for i := 0; i < l-1; i += 2 {
		k := params[i].(string)
		v := params[i+1]
		if reflect.TypeOf(v).Kind() == reflect.Func {
			fnDict[k] = v
		} else {
			dict[k] = v
		}
	}

	t := template.Must(template.New("").Funcs(fnDict).Parse(tpl))
	E(t.Execute(&out, dict))

	return out.String()
}

// RandString generate random string with specified string length
func RandString(len int) string {
	b := make([]byte, len)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// Mkdir makes dir recursively
func Mkdir(path string) error {
	return os.MkdirAll(path, 0o775)
}

// AbsolutePaths returns absolute paths of files in current working directory
func AbsolutePaths(paths []string) []string {
	absPaths := []string{}
	for _, p := range paths {
		absPath, err := filepath.Abs(p)
		E(err)
		absPaths = append(absPaths, absPath)
	}
	return absPaths
}

// OutputFile auto creates file if not exists, it will try to detect the data type and
// auto output binary, string or json
func OutputFile(p string, data interface{}) error {
	dir := filepath.Dir(p)
	_ = Mkdir(dir)

	var bin []byte

	switch t := data.(type) {
	case []byte:
		bin = t
	case string:
		bin = []byte(t)
	case io.Reader:
		f, _ := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o664)
		_, err := io.Copy(f, t)
		return err
	default:
		bin = MustToJSONBytes(data)
	}

	return ioutil.WriteFile(p, bin, 0o664)
}

// ReadString reads file as string
func ReadString(p string) (string, error) {
	bin, err := ioutil.ReadFile(p)
	return string(bin), err
}

// All runs all actions concurrently, returns the wait function for all actions.
func All(actions ...func()) func() {
	wg := &sync.WaitGroup{}

	wg.Add(len(actions))

	runner := func(action func()) {
		defer wg.Done()
		action()
	}

	for _, action := range actions {
		go runner(action)
	}

	return wg.Wait
}

// IdleCounter is similar to sync.WaitGroup but it only resolves if no jobs for specified duration.
type IdleCounter struct {
	lock     *sync.Mutex
	job      int
	duration time.Duration
	tmr      *time.Timer
}

// NewIdleCounter ...
func NewIdleCounter(d time.Duration) *IdleCounter {
	tmr := time.NewTimer(time.Hour)
	tmr.Stop()

	return &IdleCounter{
		lock:     &sync.Mutex{},
		duration: d,
		tmr:      tmr,
	}
}

// Add ...
func (de *IdleCounter) Add() {
	de.lock.Lock()
	defer de.lock.Unlock()

	de.tmr.Stop()
	de.job++
}

// Done ...
func (de *IdleCounter) Done() {
	de.lock.Lock()
	defer de.lock.Unlock()

	de.job--
	if de.job == 0 {
		de.tmr.Reset(de.duration)
	}
	if de.job < 0 {
		panic("all jobs are already done")
	}
}

// Wait ...
func (de *IdleCounter) Wait(ctx context.Context) {
	de.lock.Lock()
	if de.job == 0 {
		de.tmr.Reset(de.duration)
	}
	de.lock.Unlock()

	select {
	case <-ctx.Done():
		de.tmr.Stop()
	case <-de.tmr.C:
	}
}

var chPause = make(chan struct{})

// Pause the goroutine forever
func Pause() {
	<-chPause
}

// Dump values for debugging
func Dump(list ...interface{}) string {
	out := []string{}
	for _, el := range list {
		out = append(out, gson.New(el).JSON("", "  "))
	}
	return strings.Join(out, " ")
}

// MustToJSONBytes encode data to json bytes
func MustToJSONBytes(data interface{}) []byte {
	buf := bytes.NewBuffer(nil)
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	E(enc.Encode(data))
	b := buf.Bytes()
	return b[:len(b)-1]
}

// MustToJSON encode data to json string
func MustToJSON(data interface{}) string {
	return string(MustToJSONBytes(data))
}

// FileExists checks if file exists, only for file, not for dir
func FileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	if info.IsDir() {
		return false
	}

	return true
}

var regSpace = regexp.MustCompile(`\s`)

// Exec command
func Exec(line string, rest ...string) string {
	return ExecLine(true, line, rest...)
}

var execLogger = log.New(os.Stdout, "[exec] ", 0)

// ExecLine of command
func ExecLine(std bool, line string, rest ...string) string {
	args := rest
	if line != "" {
		args = append(regSpace.Split(line, -1), rest...)
	}

	execLogger.Println(FormatCLIArgs(args))

	buf := bytes.NewBuffer(nil)

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stderr = buf
	cmd.Stdout = buf

	if std {
		cmd.Stdin = os.Stdin
		cmd.Stderr = io.MultiWriter(buf, os.Stderr)
		cmd.Stdout = io.MultiWriter(buf, os.Stdout)
	}

	if err := cmd.Run(); err != nil {
		if std {
			panic(err)
		}
		panic(fmt.Sprintf("%v\n%v", err, buf.String()))
	}

	return buf.String()
}

// FormatCLIArgs into one line string
func FormatCLIArgs(args []string) string {
	list := []string{}
	for _, arg := range args {
		if regSpace.MatchString(arg) {
			list = append(list, fmt.Sprintf("%#v", arg))
		} else {
			list = append(list, arg)
		}
	}
	return strings.Join(list, " ")
}

// EscapeGoString not using encoding like base64 or gzip because of they will make git diff every large for small change
func EscapeGoString(s string) string {
	return "`" + strings.ReplaceAll(s, "`", "` + \"`\" + `") + "`"
}

// CropImage by the specified box, quality is only for jpeg bin.
func CropImage(bin []byte, quality, x, y, width, height int) ([]byte, error) {
	img, typ, err := image.Decode(bytes.NewBuffer(bin))
	if err != nil {
		return nil, err
	}

	cropped := bytes.NewBuffer(nil)

	switch typ {
	case "png":
		img = img.(*image.NRGBA).SubImage(image.Rect(
			x, y, x+width, y+height,
		))

		err = png.Encode(cropped, img)
	case "jpeg":
		img = img.(*image.YCbCr).SubImage(image.Rect(
			x, y, x+width, y+height,
		))

		if quality == 0 {
			quality = 80
		}

		err = jpeg.Encode(cropped, img, &jpeg.Options{Quality: quality})
	}

	return cropped.Bytes(), err
}
