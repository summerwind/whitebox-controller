package webhook

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/manager"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/summerwind/whitebox-controller/config"
	"github.com/summerwind/whitebox-controller/handler/exec"
)

var (
	timeout = 30 * time.Second
	log     = logf.Log.WithName("webhook")
)

type Server struct {
	handler http.Handler
	config  *config.WebhookConfig
}

func NewServer(c *config.WebhookConfig, mgr manager.Manager) (*Server, error) {
	mux := http.NewServeMux()

	for _, hc := range c.Handlers {
		if hc.Validator != nil {
			hook, err := newValidationHook(hc.Validator)
			if err != nil {
				return nil, err
			}

			res := hc.Resource
			p := fmt.Sprintf("/%s.%s/%s/validate", strings.ToLower(res.Kind), res.Group, res.Version)

			log.Info("Adding validation hook", "path", p)
			mux.Handle(p, hook)
		}

		if hc.Mutator != nil {
			hook, err := newMutationHook(hc.Mutator)
			if err != nil {
				return nil, err
			}

			res := hc.Resource
			p := fmt.Sprintf("/%s.%s/%s/mutate", strings.ToLower(res.Kind), res.Group, res.Version)

			log.Info("Adding mutation hook", "path", p)
			mux.Handle(p, hook)
		}
	}

	s := &Server{
		handler: wrap(mux),
		config:  c,
	}

	return s, mgr.Add(s)
}

func (s *Server) Start(stop <-chan struct{}) error {
	cert, err := tls.LoadX509KeyPair(s.config.TLS.CertFile, s.config.TLS.KeyFile)
	if err != nil {
		return err
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	listener, err := tls.Listen("tcp", addr, tlsConfig)
	if err != nil {
		return err
	}

	server := &http.Server{
		Handler: s.handler,
	}

	shutdown := make(chan struct{})
	go func() {
		<-stop

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		err := server.Shutdown(ctx)
		if err != nil {
			log.Error(err, "Failed to gracefully shutdown")
		}

		close(shutdown)
	}()

	log.Info("Starting webhook server", "address", addr)
	err = server.Serve(listener)
	if err != nil && err != http.ErrServerClosed {
		return err
	}

	<-shutdown
	return nil
}

func wrap(h http.Handler) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		reqPath := req.URL.Path
		start := time.Now()

		defer func() {
			d := time.Now().Sub(start).Seconds()
			log.Info("Requesting webhook handler", "path", reqPath, "duration", d)
		}()

		h.ServeHTTP(resp, req)
	})
}

func newValidationHook(hc *config.HandlerConfig) (http.Handler, error) {
	h, err := exec.NewHandler(hc.Exec)
	if err != nil {
		return nil, err
	}

	validator := func(ctx context.Context, req admission.Request) admission.Response {
		buf, err := json.Marshal(req)
		if err != nil {
			return admission.ValidationResponse(false, fmt.Sprintf("invalid request: %v", err))
		}

		out, err := h.Run(buf)
		if err != nil {
			return admission.ValidationResponse(false, fmt.Sprintf("handler error: %v", err))
		}

		res := admission.Response{}
		err = json.Unmarshal(out, &res)
		if err != nil {
			return admission.ValidationResponse(false, fmt.Sprintf("handler error: %v", err))
		}

		return res
	}

	hook := &admission.Webhook{Handler: admission.HandlerFunc(validator)}
	hook.InjectLogger(log)

	return hook, nil
}

func newMutationHook(hc *config.HandlerConfig) (http.Handler, error) {
	h, err := exec.NewHandler(hc.Exec)
	if err != nil {
		return nil, err
	}

	mutator := func(ctx context.Context, req admission.Request) admission.Response {
		buf, err := json.Marshal(req)
		if err != nil {
			return admission.Errored(http.StatusInternalServerError, fmt.Errorf("invalid request: %v", err))
		}

		out, err := h.Run(buf)
		if err != nil {
			return admission.Errored(http.StatusInternalServerError, fmt.Errorf("handler error: %v", err))
		}

		res := admission.Response{}
		err = json.Unmarshal(out, &res)
		if err != nil {
			return admission.Errored(http.StatusInternalServerError, fmt.Errorf("handler error: %v", err))
		}

		return res
	}

	hook := &admission.Webhook{Handler: admission.HandlerFunc(mutator)}
	hook.InjectLogger(log)

	return hook, nil
}
