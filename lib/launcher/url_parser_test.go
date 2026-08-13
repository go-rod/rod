package launcher

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestResolveURLUsingDefaultProxy(t *testing.T) {
	setProxy("http://127.0.0.1:7890")

	u, err := ResolveURL("37712")
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(u, "<nil>") || strings.Contains(u, `%3Cnil%3E`) {
		t.Fatal(u)
	}

	t.Log(u)
}

func setProxy(proxy string) error {
	proxyURL, err := url.Parse(proxy)
	if err != nil {
		return err
	}

	// set http default client to use proxy
	http.DefaultClient.Transport = &http.Transport{
		Proxy:           http.ProxyURL(proxyURL),
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	return nil
}
