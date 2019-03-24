package reconciler

import (
	"context"
	"reflect"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"

	"github.com/summerwind/whitebox-controller/config"
	"github.com/summerwind/whitebox-controller/handler"
	"github.com/summerwind/whitebox-controller/handler/exec"
)

type Reconciler struct {
	client.Client
	config  *config.ControllerConfig
	handler handler.Handler
	log     logr.Logger
}

func NewReconciler(config *config.ControllerConfig) (*Reconciler, error) {
	h, err := exec.NewHandler(config.Reconciler.Exec)
	if err != nil {
		return nil, err
	}

	r := &Reconciler{
		config:  config,
		handler: h,
		log:     logf.Log.WithName("reconciler"),
	}

	return r, nil
}

func (r *Reconciler) InjectClient(c client.Client) error {
	r.Client = c
	return nil
}

func (r *Reconciler) Reconcile(req reconcile.Request) (reconcile.Result, error) {
	namespace := req.NamespacedName.Namespace
	name := req.NamespacedName.Name

	instance := &unstructured.Unstructured{}
	instance.SetGroupVersionKind(r.config.Resource)

	err := r.Get(context.TODO(), req.NamespacedName, instance)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		r.log.Error(err, "Failed to get a resource", "namespace", namespace, "name", name)
		return reconcile.Result{}, err
	}

	state := handler.NewState(handler.ActionReconcile, instance)
	newState, err := r.handler.Run(state)
	if err != nil {
		r.log.Error(err, "Handler error", "namespace", namespace, "name", name)
		return reconcile.Result{}, err
	}

	resource := newState.Resource
	resource.SetGroupVersionKind(r.config.Resource)

	if !reflect.DeepEqual(instance, resource) {
		err = r.Update(context.TODO(), resource)
		if err != nil {
			r.log.Error(err, "Failed to update a resource", "namespace", namespace, "name", name)
			return reconcile.Result{}, err
		}

		r.log.Info("Resource updated", "namespace", namespace, "name", name)
	}

	return reconcile.Result{}, nil
}
