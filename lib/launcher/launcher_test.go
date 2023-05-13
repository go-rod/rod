package launcher_test

import (
	"archive/zip"
	"bytes"
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-rod/rod/lib/defaults"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/launcher/flags"
	"github.com/go-rod/rod/lib/utils"
	"github.com/ysmood/got"
)

var setup = got.Setup(nil)

func TestDownloadHosts(t *testing.T) {
	g := setup(t)

	g.Has(launcher.HostGoogle(launcher.RevisionDefault), "https://storage.googleapis.com/chromium-browser-snapshots")
	g.Has(launcher.HostNPM(launcher.RevisionDefault), "https://registry.npmmirror.com/-/binary/chromium-browser-snapshots")
	g.Has(launcher.HostPlaywright(launcher.RevisionDefault), "https://playwright.azureedge.net/")
}

func TestDownload(t *testing.T) {
	g := got.T(t)

	buf := bytes.NewBuffer(nil)
	z := zip.NewWriter(buf)
	f, _ := z.Create(filepath.FromSlash("a/b/c.txt"))
	_, _ = f.Write([]byte(g.RandStr(500 * 1024)))
	_ = z.Close()

	s := g.Serve()
	s.Route("/", ".zip", buf.Bytes())

	b := launcher.NewBrowser()
	b.Revision = 1
	b.Logger = utils.LoggerQuiet
	b.Hosts = []launcher.Host{func(_ int) string {
		return s.URL("/a.zip")
	}}

	g.Cleanup(func() { _ = os.RemoveAll(b.Dir()) })

	b.MustGet()

	g.PathExists(b.Dir())
}

func TestLaunch(t *testing.T) {
	g := setup(t)

	defaults.Proxy = "test.com"
	defer func() { defaults.ResetWith("") }()

	l := launcher.New()
	defer l.Kill()

	u := l.MustLaunch()
	g.Regex(`\Aws://.+\z`, u)

	parsed, _ := url.Parse(u)

	{ // test GetWebSocketDebuggerURL
		for _, prefix := range []string{"", ":", "127.0.0.1:", "ws://127.0.0.1:"} {
			u2 := launcher.MustResolveURL(prefix + parsed.Port())
			g.Regex(u, u2)
		}

		_, err := launcher.ResolveURL("")
		g.Err(err)
	}

	{
		_, err := launcher.NewManaged("")
		g.Err(err)

		_, err = launcher.NewManaged("1://")
		g.Err(err)

		_, err = launcher.NewManaged("ws://not-exists")
		g.Err(err)
	}

	{
		g.Panic(func() { launcher.New().Set("a=b") })
	}
}

func TestLaunchUserMode(t *testing.T) {
	g := setup(t)

	l := launcher.NewUserMode()
	defer l.Kill()

	l.Kill() // empty kill should do nothing

	has := l.Has("not-exists")
	g.False(has)

	l.Append("test-append", "a")
	f := l.Get("test-append")
	g.Eq("a", f)

	dir := l.Get(flags.UserDataDir)
	port := 58472

	l = l.Context(g.Context()).Delete("test").Bin("").
		Revision(launcher.RevisionDefault).
		Logger(ioutil.Discard).
		Leakless(false).Leakless(true).
		Headless(false).Headless(true).RemoteDebuggingPort(port).
		NoSandbox(true).NoSandbox(false).
		Devtools(true).Devtools(false).
		StartURL("about:blank").
		Proxy("test.com").
		UserDataDir("test").UserDataDir(dir).
		WorkingDir("").
		Env(append(os.Environ(), "TZ=Asia/Tokyo")...)

	g.Eq(l.FormatArgs(), []string /* len=6 cap=8 */ {
		"--headless",
		`--no-startup-window`,           /* len=19 */
		`--proxy-server=test.com`,       /* len=23 */
		`--remote-debugging-port=58472`, /* len=29 */
		"--test-append=a",
		"about:blank",
	})

	url := l.MustLaunch()

	g.Eq(url, launcher.NewUserMode().RemoteDebuggingPort(port).MustLaunch())
}

func TestUserModeErr(t *testing.T) {
	g := setup(t)

	_, err := launcher.NewUserMode().RemoteDebuggingPort(48277).Bin("not-exists").Launch()
	g.Err(err)

	_, err = launcher.NewUserMode().RemoteDebuggingPort(58217).Bin("echo").Launch()
	g.Err(err)
}

func TestAppMode(t *testing.T) {
	g := setup(t)

	l := launcher.NewAppMode("http://example.com")

	g.Eq(l.Get(flags.App), "http://example.com")
}

func TestGetWebSocketDebuggerURLErr(t *testing.T) {
	g := setup(t)

	_, err := launcher.ResolveURL("1://")
	g.Err(err)
}

func TestLaunchErr(t *testing.T) {
	g := setup(t)

	g.Panic(func() {
		launcher.New().Bin("not-exists").MustLaunch()
	})
	g.Panic(func() {
		launcher.New().Headless(false).Bin("not-exists").MustLaunch()
	})
	g.Panic(func() {
		launcher.New().ClientHeader()
	})
	{
		l := launcher.New().XVFB()
		_, _ = l.Launch()
		l.Kill()
	}
}

var testProfileDir = flag.Bool("test-profile-dir", false, "set it to test profile dir")

func TestProfileDir(t *testing.T) {
	g := setup(t)

	url := launcher.New().Headless(false).
		ProfileDir("").ProfileDir("test-profile-dir")

	if !*testProfileDir {
		g.Skip("It's not CI friendly, so we skip it!")
	}

	url.MustLaunch()

	userDataDir := url.Get(flags.UserDataDir)
	file, err := os.Stat(filepath.Join(userDataDir, "test-profile-dir"))

	g.E(err)
	g.True(file.IsDir())
}

func TestBrowserValid(t *testing.T) {
	g := setup(t)

	b := launcher.NewBrowser()
	b.Revision = 0
	g.Err(b.Validate())

	g.E(utils.Mkdir(filepath.Dir(b.BinPath())))
	g.Cleanup(func() { _ = os.RemoveAll(b.Dir()) })

	g.E(exec.Command("go", "build", "-o", b.BinPath(), "./fixtures/chrome-exit-err").CombinedOutput())
	g.Has(b.Validate().Error(), "failed to run the browser")

	g.E(exec.Command("go", "build", "-o", b.BinPath(), "./fixtures/chrome-empty").CombinedOutput())
	g.Eq(b.Validate().Error(), "the browser executable doesn't support headless mode")

	g.E(exec.Command("go", "build", "-o", b.BinPath(), "./fixtures/chrome-lib-missing").CombinedOutput())
	g.Nil(b.Validate())
}

func TestIgnoreCerts(t *testing.T) {
	g := setup(t)

	// https://travistidwell.com/jsencrypt/demo/
	testData := []string{
		`-----BEGIN PUBLIC KEY-----
MIGeMA0GCSqGSIb3DQEBAQUAA4GMADCBiAKBgF9pr2zok5bivQIEUN7Y58a9uB1o
sroMt3hxNfzOh/G+sXgYPPoEl2/Ys/2zbvym7Ze0eGbb6FrV8aueg89TPTNWAKlN
N49q6S3zLG1WmI2rVYz4LtPgpg1YR9FQRIg4Ll0C02daufXgvUBGjIARH19FTw6P
61kEhnEQxUHhdAqbAgMBAAE=
-----END PUBLIC KEY-----
		`,
		`-----BEGIN PUBLIC KEY-----
MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQCvBTz/TOYc66qB97OyYenSHk4T
hAUKX5RUWZ/80o0zyJoo1dfrrwW9PlT5o4DlGMs0NSbtJ8RMQRTLZwL/zxXjiEMv
dKFs2OrefYKANTc0e2XAtQAm3Is5Ro8AF1S4Fk+eZXr2yZtBRKXvhJ/A2bilVoSn
fmQnyBe7dVU43NXfrQIDAQAB
-----END PUBLIC KEY-----
		`,
	}

	keys := make([]crypto.PublicKey, 0, len(testData))

	for _, pubPEM := range testData {
		block, _ := pem.Decode([]byte(pubPEM))
		if block == nil {
			g.Fatal("failed to parse PEM block containing the public key")
			return // no-op because g.Fatal calls t.FailNow() but `staticcheck` doesn't know it
		}

		pub, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			g.Fatalf("failed to parse DER encoded public key: " + err.Error())
		}

		keys = append(keys, pub)
	}

	l := launcher.New()

	err := l.IgnoreCerts(keys)
	if err != nil {
		g.Fatalf("IgnoreCerts: %s", err)
	}

	expected := "--ignore-certificate-errors-spki-list=" + strings.Join([]string{
		"+ZqfrXb+V/36nZecO59bghHlNhiHTzImjYLnNWGUd1I=",
		"llpTCSqZ2/IKsMg4tz+o1mCkXIOdKcM6sKu9kC6o7S4=",
	}, ",")

	g.Has(l.FormatArgs(), expected)
}

func TestIgnoreCerts_InvalidCert(t *testing.T) {
	g := setup(t)

	l := launcher.New()

	err := l.IgnoreCerts([]crypto.PublicKey{nil})
	if err == nil {
		g.Fatalf("IgnoreCerts: %s", err)
	}
}

func TestBrowserDownloadErr(t *testing.T) {
	g := setup(t)
	b := launcher.NewBrowser()
	b.Logger = utils.LoggerQuiet
	b.HTTPClient = http.DefaultClient
	b.Hosts = []launcher.Host{}
	g.Err(b.Download())

	s := g.Serve()
	s.Route("/download", ".txt", "ok")

	b = launcher.NewBrowser()
	b.Hosts = []launcher.Host{func(_ int) string {
		return s.URL("/download/file")
	}}
	g.Err(b.Download())
}
