package defaults

import (
	"testing"
	"time"

	"github.com/ysmood/got"
)

func TestBasic(t *testing.T) {
	as := got.New(t)

	Show = true
	Devtools = true
	URL = "test"
	Monitor = "test"

	ResetWithEnv("")
	parse("")
	as.False(Show)
	as.False(Devtools)
	as.Eq("", Monitor)
	as.Eq("", URL)
	as.Eq(2978, Lock)
	as.Regex(`rod[\\/]user-data`, Dir)

	parse("show,devtools,trace,slow=2s,port=8080,dir=tmp," +
		"url=http://test.com,cdp,monitor,bin=/path/to/chrome," +
		"proxy=localhost:8080,lock=9981",
	)

	as.True(Show)
	as.True(Devtools)
	as.True(Trace)
	as.Eq(2*time.Second, Slow)
	as.Eq("8080", Port)
	as.Eq("/path/to/chrome", Bin)
	as.Eq("tmp", Dir)
	as.Eq("http://test.com", URL)
	as.NotNil(CDP.Println)
	as.Eq(":0", Monitor)
	as.Eq("localhost:8080", Proxy)
	as.Eq(9981, Lock)

	parse("monitor=:1234")
	as.Eq(":1234", Monitor)

	as.Panic(func() {
		parse("a")
	})

	as.Eq(try(func() { parse("slow=1") }), "invalid value for \"slow\": time: missing unit in duration \"1\" (learn format from https://golang.org/pkg/time/#ParseDuration)")
}

func TestDotFile(t *testing.T) {
	as := got.New(t)

	ResetWithEnv("")
	parse(`

show

 port=9999
dir=path =to file

	`)

	as.True(Show)
	as.Eq("9999", Port)
	as.Eq("path =to file", Dir)
}

func try(fn func()) (err interface{}) {
	defer func() {
		err = recover()
	}()

	fn()

	return err
}
