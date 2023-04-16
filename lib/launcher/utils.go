package launcher

import (
	"crypto"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"net/url"

	"github.com/go-rod/rod/lib/utils"
)

var inContainer = utils.InContainer

func toHTTP(u url.URL) *url.URL {
	newURL := u
	if newURL.Scheme == "ws" {
		newURL.Scheme = "http"
	} else if newURL.Scheme == "wss" {
		newURL.Scheme = "https"
	}
	return &newURL
}

func toWS(u url.URL) *url.URL {
	newURL := u
	if newURL.Scheme == "http" {
		newURL.Scheme = "ws"
	} else if newURL.Scheme == "https" {
		newURL.Scheme = "wss"
	}
	return &newURL
}

// certSPKI generates the SPKI of a certificate public key
// https://blog.afoolishmanifesto.com/posts/golang-self-signed-and-pinned-certs/
func certSPKI(pk crypto.PublicKey) ([]byte, error) {
	pubDER, err := x509.MarshalPKIXPublicKey(pk)
	if err != nil {
		return nil, fmt.Errorf("x509.MarshalPKIXPublicKey: %w", err)
	}

	sum := sha256.Sum256(pubDER)
	pin := make([]byte, base64.StdEncoding.EncodedLen(len(sum)))
	base64.StdEncoding.Encode(pin, sum[:])

	return pin, nil
}
