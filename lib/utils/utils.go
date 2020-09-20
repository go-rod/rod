package utils

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	mr "math/rand"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/tidwall/gjson"
)

// Nil type
type Nil struct{}

// E if the last arg is error, panic it
func E(args ...interface{}) []interface{} {
	err, ok := args[len(args)-1].(error)
	if ok {
		panic(err)
	}
	return args
}

// SDump a value
func SDump(v interface{}) string {
	buf := bytes.NewBuffer(nil)
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	E(enc.Encode(v))
	return strings.TrimRight(string(buf.Bytes()), "\n")
}

// Dump values to logger
func Dump(list ...interface{}) {
	out := []string{}
	for _, v := range list {
		out = append(out, SDump(v))
	}
	log.Println(strings.Join(out, " "))
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
	return os.MkdirAll(path, 0775)
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
	default:
		bin = MustToJSONBytes(data)
	}

	return ioutil.WriteFile(p, bin, 0664)
}

// ReadString reads file as string
func ReadString(p string) (string, error) {
	bin, err := ioutil.ReadFile(p)
	return string(bin), err
}

// All run all actions concurrently, returns the wait function for all actions.
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

// Sleep the goroutine for specified seconds, such as 2.3 seconds
func Sleep(seconds float64) {
	d := time.Duration(seconds * float64(time.Second))
	time.Sleep(d)
}

// Sleeper sleeps for sometime, returns the reason to wake, if ctx is done release resource
type Sleeper func(context.Context) error

// CountSleeper wake when counts to max and return
func CountSleeper(max int) Sleeper {
	count := 0
	return func(ctx context.Context) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if count == max {
			return errors.New("max sleep count")
		}
		count++
		return nil
	}
}

// DefaultBackoff algorithm: A(n) = A(n-1) * random[1.9, 2.1)
func DefaultBackoff(interval time.Duration) time.Duration {
	scale := 2 + (mr.Float64()-0.5)*0.2
	return time.Duration(float64(interval) * scale)
}

// BackoffSleeper returns a sleeper that sleeps in a backoff manner every time get called.
// If algorithm is nil, DefaultBackoff will be used.
// Set interval and maxInterval to the same value to make it a constant interval sleeper.
// If maxInterval is not greater than 0, it will wake immediately.
func BackoffSleeper(init, maxInterval time.Duration, algorithm func(time.Duration) time.Duration) Sleeper {
	if algorithm == nil {
		algorithm = DefaultBackoff
	}

	return func(ctx context.Context) error {
		// wake immediately
		if maxInterval <= 0 {
			return nil
		}

		var interval time.Duration
		if init < maxInterval {
			interval = algorithm(init)
		} else {
			interval = maxInterval
		}

		t := time.NewTicker(interval)
		defer t.Stop()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
			init = interval
		}

		return nil
	}
}

// Retry fn and sleeper until fn returns true or s returns error
func Retry(ctx context.Context, s Sleeper, fn func() (stop bool, err error)) error {
	for {
		stop, err := fn()
		if stop {
			return err
		}
		err = s(ctx)
		if err != nil {
			return err
		}
	}
}

var chPause = make(chan struct{})

// Pause the goroutine forever
func Pause() {
	<-chPause
}

// IsSyncMapEmpty helper
func IsSyncMapEmpty(s *sync.Map) bool {
	isEmpty := true
	s.Range(func(key, value interface{}) bool {
		isEmpty = false
		return false
	})
	return isEmpty
}

// SyncMapToMap convertor
func SyncMapToMap(s *sync.Map) map[string]interface{} {
	m := map[string]interface{}{}
	s.Range(func(key, value interface{}) bool {
		m[fmt.Sprintf("%v", key)] = value
		return false
	})
	return m
}

// MustToJSONBytes encode data to json bytes
func MustToJSONBytes(data interface{}) []byte {
	bytes, err := json.Marshal(data)
	E(err)
	return bytes
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

// Exec command
func Exec(name string, args ...string) {
	cmd := exec.Command(name, args...)
	SetCmdStdPipe(cmd)
	E(cmd.Run())
}

// SetCmdStdPipe command
func SetCmdStdPipe(cmd *exec.Cmd) {
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
}

type errMuxWrapper struct {
	mux *http.ServeMux
}

// ServeHTTP interface
func (h *errMuxWrapper) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			E(w.Write([]byte(fmt.Sprint(err))))
		}
	}()

	h.mux.ServeHTTP(w, r)
}

// Serve a port, if host is empty a random port will be used.
func Serve(host string) (string, *http.ServeMux, func()) {
	if host == "" {
		host = "127.0.0.1:0"
	}

	mux := http.NewServeMux()
	srv := &http.Server{Handler: &errMuxWrapper{mux}}

	l, err := net.Listen("tcp", host)
	E(err)

	go func() { _ = srv.Serve(l) }()

	url := "http://" + l.Addr().String()

	return url, mux, func() {
		E(srv.Close())
	}
}

// ReadJSON from reader
func ReadJSON(r io.Reader) (gjson.Result, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return gjson.Result{}, err
	}
	return gjson.ParseBytes(b), nil
}

// ReadJSONPathAsString from reader
func ReadJSONPathAsString(r io.Reader, path string) (string, error) {
	obj, err := ReadJSON(r)
	if err != nil {
		return "", err
	}

	return obj.Get(path).String(), nil
}

// MustReadJSON from reader
func MustReadJSON(r io.Reader) gjson.Result {
	j, err := ReadJSON(r)
	E(err)
	return j
}

// MustReadBytes from reader
func MustReadBytes(r io.Reader) []byte {
	b, err := ioutil.ReadAll(r)
	E(err)
	return b
}

// MustReadString from reader
func MustReadString(r io.Reader) string {
	return string(MustReadBytes(r))
}

// EscapeGoString not using encoding like base64 or gzip because of they will make git diff every large for small change
func EscapeGoString(s string) string {
	return "`" + strings.ReplaceAll(s, "`", "` + \"`\" + `") + "`"
}
