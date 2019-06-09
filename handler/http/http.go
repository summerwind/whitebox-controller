package http

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/summerwind/whitebox-controller/config"
)

var defaultTimeout = 60 * time.Second

type HTTPHandler struct {
	client *http.Client
	url    string
	debug  bool
}

func NewHandler(c *config.HTTPHandlerConfig) (*HTTPHandler, error) {
	var (
		timeout time.Duration
		err     error
	)

	if c.Timeout != "" {
		timeout, err = time.ParseDuration(c.Timeout)
		if err != nil {
			return nil, err
		}
	} else {
		timeout = defaultTimeout
	}

	tlsConfig := &tls.Config{}

	if c.TLS != nil {
		if c.TLS.KeyFile != "" && c.TLS.CertFile != "" {
			cert, err := tls.LoadX509KeyPair(c.TLS.CertFile, c.TLS.KeyFile)
			if err != nil {
				return nil, err
			}

			tlsConfig.Certificates = []tls.Certificate{cert}
			tlsConfig.BuildNameToCertificate()
		}

		if c.TLS.CACertFile != "" {
			caCert, err := ioutil.ReadFile(c.TLS.CACertFile)
			if err != nil {
				return nil, err
			}

			caCertPool := x509.NewCertPool()
			caCertPool.AppendCertsFromPEM(caCert)

			tlsConfig.RootCAs = caCertPool
		}
	}

	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	return &HTTPHandler{
		client: client,
		url:    c.URL,
		debug:  c.Debug,
	}, nil
}

func (h *HTTPHandler) Run(buf []byte) ([]byte, error) {
	reqBody := bytes.NewBuffer(buf)

	req, err := http.NewRequest("POST", h.url, reqBody)
	if err != nil {
		return nil, err
	}

	if h.debug {
		log("request", string(buf))
	}

	req.Header.Set("Content-Type", "application/json")
	res, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("invalid status: %s", res.Status)
	}

	resBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if h.debug {
		log("response", string(resBody))
	}

	return resBody, nil
}

func log(stream, msg string) {
	fmt.Fprintf(os.Stderr, "[http] %s: %s\n", stream, msg)
}
