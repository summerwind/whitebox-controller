package webhook

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/dgrijalva/jwt-go"
	"github.com/summerwind/whitebox-controller/config"
	"github.com/summerwind/whitebox-controller/handler/common"
	"github.com/summerwind/whitebox-controller/webhook/injection"
)

var (
	timeout = 30 * time.Second
	log     = logf.Log.WithName("webhook")
)

type Server struct {
	client.Client
	config  *config.ServerConfig
	mux     *http.ServeMux
	handler http.Handler
}

func NewServer(c *config.ServerConfig, mgr manager.Manager) (*Server, error) {
	mux := http.NewServeMux()

	if c == nil {
		return nil, fmt.Errorf("configuration must be specified")
	}
	if c.TLS == nil {
		return nil, fmt.Errorf("TLS configuration must be specified")
	}

	s := &Server{
		config:  c,
		mux:     mux,
		handler: wrap(mux),
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

	port := s.config.Port
	if port == 0 {
		port = 443
	}
	addr := fmt.Sprintf("%s:%d", s.config.Host, port)

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

func (s *Server) AddValidator(c *config.ResourceConfig) error {
	hook, err := newValidationHook(c.Validator)
	if err != nil {
		return err
	}

	p := fmt.Sprintf("%s/validate", getBasePath(c.GroupVersionKind))
	log.Info("Adding validation hook", "path", p)
	s.mux.Handle(p, hook)

	return nil
}

func (s *Server) AddMutator(c *config.ResourceConfig) error {
	hook, err := newMutationHook(c.Mutator)
	if err != nil {
		return err
	}

	p := fmt.Sprintf("%s/mutate", getBasePath(c.GroupVersionKind))
	log.Info("Adding mutation hook", "path", p)
	s.mux.Handle(p, hook)

	return nil
}

func (s *Server) AddInjector(c *config.ResourceConfig) error {
	hook, err := newInjectionHook(c.Injector, s.Client)
	if err != nil {
		return err
	}

	p := fmt.Sprintf("%s/inject", getBasePath(c.GroupVersionKind))
	log.Info("Adding injection hook", "path", p)
	s.mux.Handle(p, hook)

	return nil
}

func (s *Server) InjectClient(c client.Client) error {
	s.Client = c
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

func getBasePath(gvk schema.GroupVersionKind) string {
	return fmt.Sprintf("/%s/%s/%s", gvk.Group, gvk.Version, strings.ToLower(gvk.Kind))
}

func newValidationHook(hc *config.HandlerConfig) (http.Handler, error) {
	h, err := common.NewAdmissionRequestHandler(hc)
	if err != nil {
		return nil, err
	}

	validator := func(ctx context.Context, req admission.Request) admission.Response {
		res, err := h.HandleAdmissionRequest(req)
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
	h, err := common.NewAdmissionRequestHandler(hc)
	if err != nil {
		return nil, err
	}

	mutator := func(ctx context.Context, req admission.Request) admission.Response {
		res, err := h.HandleAdmissionRequest(req)
		if err != nil {
			return admission.Errored(http.StatusInternalServerError, fmt.Errorf("handler error: %v", err))
		}

		return res
	}

	hook := &admission.Webhook{Handler: admission.HandlerFunc(mutator)}
	hook.InjectLogger(log)

	return hook, nil
}

func newInjectionHook(ic *config.InjectorConfig, client client.Client) (http.Handler, error) {
	var (
		key interface{}
		err error
	)

	buf, err := ioutil.ReadFile(ic.VerifyKeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read verification key: %v", err)
	}

	key, err = jwt.ParseRSAPublicKeyFromPEM(buf)
	if err != nil {
		key, err = jwt.ParseECPublicKeyFromPEM(buf)
		if err != nil {
			return nil, errors.New("unsupported verification key type")
		}
	}

	keyHandler := func(token *jwt.Token) (interface{}, error) {
		switch token.Method.Alg() {
		case "ES256":
			return key, nil
		case "RS256":
			return key, nil
		}

		return nil, errors.New("unsupported signing key type")
	}

	h, err := common.NewInjectionRequestHandler(&ic.HandlerConfig)
	if err != nil {
		return nil, err
	}

	handler := func(ctx context.Context, req injection.Request) (injection.Response, error) {
		res, err := h.HandleInjectionRequest(req)
		if err != nil {
			return res, errors.New("Handler error")
		}

		return res, nil
	}

	hook := &injection.Webhook{
		Handler:    injection.HandlerFunc(handler),
		KeyHandler: keyHandler,
	}
	hook.InjectClient(client)
	hook.InjectLogger(log)

	return hook, nil
}

func Unpack(r runtime.RawExtension, v interface{}) error {
	err := json.Unmarshal(r.Raw, v)
	if err != nil {
		return fmt.Errorf("failed to unpack: %v", err)
	}

	return nil
}
