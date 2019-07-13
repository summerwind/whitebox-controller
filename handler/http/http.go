package http

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/summerwind/whitebox-controller/config"
	"github.com/summerwind/whitebox-controller/reconciler/state"
	"github.com/summerwind/whitebox-controller/webhook/injection"
)

var defaultTimeout = 60 * time.Second

type HTTPHandler struct {
	client *http.Client
	url    string
	debug  bool
}

func New(c *config.HTTPHandlerConfig) (*HTTPHandler, error) {
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

func (h *HTTPHandler) HandleState(s *state.State) error {
	in, err := json.Marshal(s)
	if err != nil {
		return err
	}

	out, err := h.run(in)
	if err != nil {
		return err
	}

	if len(out) == 0 {
		return nil
	}

	err = json.Unmarshal(out, s)
	if err != nil {
		return err
	}

	return nil
}

func (h *HTTPHandler) HandleAdmissionRequest(req admission.Request) (admission.Response, error) {
	res := admission.Response{}

	in, err := json.Marshal(&req)
	if err != nil {
		return res, err
	}

	out, err := h.run(in)
	if err != nil {
		return res, err
	}

	err = json.Unmarshal(out, &res)
	if err != nil {
		return res, err
	}

	return res, nil
}

func (h *HTTPHandler) HandleInjectionRequest(req injection.Request) (injection.Response, error) {
	res := injection.Response{}

	in, err := json.Marshal(&req)
	if err != nil {
		return res, err
	}

	out, err := h.run(in)
	if err != nil {
		return res, err
	}

	err = json.Unmarshal(out, &res)
	if err != nil {
		return res, err
	}

	return res, nil
}

func (h *HTTPHandler) run(buf []byte) ([]byte, error) {
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
