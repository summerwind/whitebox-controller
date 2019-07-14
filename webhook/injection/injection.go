package injection

import (
	"context"
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/dgrijalva/jwt-go"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Request struct {
	Headers http.Header `json:"headers"`
	Body    string      `json:"body"`
}

type Response struct {
	Object *unstructured.Unstructured `json:"object"`
}

type HandlerFunc func(context.Context, Request) (Response, error)

type Webhook struct {
	client.Client

	Handler    HandlerFunc
	KeyHandler jwt.Keyfunc
	log        logr.Logger
}

func (wh *Webhook) InjectClient(c client.Client) error {
	wh.Client = c
	return nil
}

func (wh *Webhook) InjectLogger(l logr.Logger) error {
	wh.log = l
	return nil
}

func (wh *Webhook) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL == nil {
		wh.error(w, "Unexpected URL", 500)
		return
	}

	tokenStr := r.URL.Query().Get("token")
	token, err := jwt.Parse(tokenStr, wh.KeyHandler)
	if err != nil {
		wh.error(w, "Invalid token", 400)
		return
	}

	if !token.Valid {
		wh.error(w, "Invalid token", 400)
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		wh.error(w, "Invalid token claims", 400)
		return
	}

	namespace := claims["namespace"].(string)
	if namespace == "" {
		wh.error(w, "Invalid namespace", 400)
		return
	}

	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		wh.error(w, "Failed to read request body", 500)
		return
	}
	defer r.Body.Close()

	req := Request{
		Headers: r.Header,
		Body:    string(buf),
	}

	res, err := wh.Handler(context.TODO(), req)
	if err != nil {
		wh.error(w, err.Error(), 500)
		return
	}

	if res.Object == nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	res.Object.SetNamespace(namespace)

	err = wh.Create(context.TODO(), res.Object)
	if err != nil {
		msg := "Failed to create a resource"
		wh.log.Error(err, msg, "namespace", res.Object.GetNamespace(), "name", res.Object.GetName())
		http.Error(w, msg, 500)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (wh *Webhook) error(w http.ResponseWriter, err string, code int) {
	wh.log.Error(errors.New(err), "injection request error", "code", code)
	http.Error(w, err, code)
}
