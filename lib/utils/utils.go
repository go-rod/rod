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
	mr "math/rand"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
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
func SDump(s interface{}) string {
	raw, ok := s.(json.RawMessage)
	if ok {
		var val interface{}
		err := json.Unmarshal(raw, &val)
		E(err)
		d, err := json.MarshalIndent(val, "", " ")
		E(err)
		return string(d)
	}

	d, err := json.MarshalIndent(s, "", " ")
	E(err)
	return string(d)
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

// OutputFileOptions ...
type OutputFileOptions struct {
	DirPerm    os.FileMode
	FilePerm   os.FileMode
	JSONPrefix string
	JSONIndent string
}

// MkdirOptions ...
type MkdirOptions struct {
	Perm os.FileMode
}

// Mkdir makes dir recursively
func Mkdir(path string, options *MkdirOptions) error {
	if options == nil {
		options = &MkdirOptions{
			Perm: 0775,
		}
	}

	return os.MkdirAll(path, options.Perm)
}

// OutputFile auto creates file if not exists, it will try to detect the data type and
// auto output binary, string or json
func OutputFile(p string, data interface{}, options *OutputFileOptions) error {
	if options == nil {
		options = &OutputFileOptions{0775, 0664, "", "    "}
	}

	dir := filepath.Dir(p)
	_ = Mkdir(dir, &MkdirOptions{Perm: options.DirPerm})

	var bin []byte

	switch t := data.(type) {
	case []byte:
		bin = t
	case string:
		bin = []byte(t)
	default:
		var err error
		bin, err = json.MarshalIndent(data, options.JSONPrefix, options.JSONIndent)

		if err != nil {
			return err
		}
	}

	return ioutil.WriteFile(p, bin, options.FilePerm)
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
type Sleeper func(ctx context.Context) error

// ErrMaxSleepCount ...
var ErrMaxSleepCount = errors.New("max sleep count")

// CountSleeper wake when counts to max and return
func CountSleeper(max int) Sleeper {
	count := 0
	return func(ctx context.Context) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if count == max {
			return ErrMaxSleepCount
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
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	E(cmd.Run())
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

	return url, mux, func() { E(l.Close()) }
}

// ReadJSON from reader
func ReadJSON(r io.Reader) (gjson.Result, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return gjson.Result{}, err
	}
	return gjson.ParseBytes(b), nil
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
