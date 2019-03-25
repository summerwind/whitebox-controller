package webhook

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/summerwind/whitebox-controller/config"
	"github.com/summerwind/whitebox-controller/handler/exec"
)

var timeout = 30 * time.Second

type Server struct {
	mux      *http.ServeMux
	handlers map[string]http.Handler
	config   *config.WebhookConfig
	log      logr.Logger
}

func NewServer(c *config.WebhookConfig, mgr manager.Manager) (*Server, error) {
	mux := http.NewServeMux()
	log := logf.Log.WithName("webhook")

	for _, hc := range c.Handlers {
		if hc.Validator.Exec == nil {
			continue
		}

		h, err := exec.NewHandler(hc.Validator.Exec)
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

		res := hc.Resource
		path := fmt.Sprintf("/%s.%s/%s/validate", strings.ToLower(res.Kind), res.Group, res.Version)

		log.Info("Adding validation hook", "path", path)
		mux.Handle(path, hook)
	}

	s := &Server{
		mux:    mux,
		config: c,
		log:    log,
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
		Handler: s.mux,
	}

	shutdown := make(chan struct{})
	go func() {
		<-stop

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		err := server.Shutdown(ctx)
		if err != nil {
			s.log.Error(err, "Failed to gracefully shutdown")
		}

		close(shutdown)
	}()

	s.log.Info("Starting webhook server", "address", addr)
	err = server.Serve(listener)
	if err != nil && err != http.ErrServerClosed {
		return err
	}

	<-shutdown
	return nil
}
