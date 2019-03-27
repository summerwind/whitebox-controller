package observer

import (
	"context"
	"encoding/json"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"

	"github.com/summerwind/whitebox-controller/config"
	"github.com/summerwind/whitebox-controller/handler"
	"github.com/summerwind/whitebox-controller/handler/exec"
)

var log = logf.Log.WithName("reconciler")

type Observer struct {
	client.Client
	config  *config.ControllerConfig
	handler handler.Handler
}

func New(config *config.ControllerConfig) (*Observer, error) {
	h, err := exec.NewHandler(config.Observer.Exec)
	if err != nil {
		return nil, err
	}

	r := &Observer{
		config:  config,
		handler: h,
	}

	return r, nil
}

func (r *Observer) InjectClient(c client.Client) error {
	r.Client = c
	return nil
}

func (r *Observer) Reconcile(req reconcile.Request) (reconcile.Result, error) {
	namespace := req.Namespace
	name := req.Name

	instance := &unstructured.Unstructured{}
	instance.SetGroupVersionKind(r.config.Resource)

	err := r.Get(context.TODO(), req.NamespacedName, instance)
	if err != nil && !apierrors.IsNotFound(err) {
		log.Error(err, "Failed to get a resource", "namespace", namespace, "name", name)
		return reconcile.Result{}, nil
	}

	// This allows determination of deleted resources
	instance.SetNamespace(namespace)
	instance.SetName(name)

	buf, err := json.Marshal(instance)
	if err != nil {
		log.Error(err, "Failed to encode resource", "namespace", namespace, "name", name)
		return reconcile.Result{}, nil
	}

	_, err = r.handler.Run(buf)
	if err != nil {
		log.Error(err, "Handler error", "namespace", namespace, "name", name)
		return reconcile.Result{}, nil
	}

	return reconcile.Result{}, nil
}
