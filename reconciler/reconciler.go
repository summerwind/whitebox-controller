package reconciler

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/summerwind/whitebox-controller/config"
	"github.com/summerwind/whitebox-controller/handler"
	"github.com/summerwind/whitebox-controller/handler/exec"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

const (
	LabelHandlerID       = "whitebox.summerwind.github.io/hanlderID"
	HandlerContainerName = "handler"
)

type Reconciler struct {
	client.Client
	config *config.Config
	log    logr.Logger
}

func NewReconciler(config *config.Config) (*Reconciler, error) {
	r := &Reconciler{
		config: config,
		log:    logf.Log.WithName("reconciler"),
	}

	return r, nil
}

func (r *Reconciler) InjectClient(c client.Client) error {
	r.Client = c
	return nil
}

func (r *Reconciler) Reconcile(req reconcile.Request) (reconcile.Result, error) {
	var (
		phase string
		ok    bool
	)

	namespace := req.NamespacedName.Namespace
	name := req.NamespacedName.Name

	instance := &unstructured.Unstructured{}
	instance.SetGroupVersionKind(r.config.Resource)

	err := r.Get(context.TODO(), req.NamespacedName, instance)
	if err != nil {
		r.log.Error(err, "Failed to get a resource", "namespace", namespace, "name", name)
		return reconcile.Result{}, err
	}

	status, ok := instance.Object["status"].(map[string]interface{})
	if ok {
		phase, _ = status["phase"].(string)
	}

	if phase == "" {
		phase = "new"
	}

	handlerConfig, ok := r.config.Handlers[phase]
	if !ok {
		r.log.Info("No handler for the phase", "namespace", namespace, "name", name, "phase", phase)
		return reconcile.Result{}, nil
	}

	phaseHandler, err := exec.NewHandler(handlerConfig.Exec)
	if err != nil {
		r.log.Error(err, "Failed to create a handler", "namespace", namespace, "name", name)
		return reconcile.Result{}, err
	}

	hreq := &handler.Request{Resource: instance}
	hres, err := phaseHandler.Run(hreq)
	if err != nil {
		r.log.Error(err, "Handler error", "namespace", namespace, "name", name, "phase", phase)
		return reconcile.Result{}, err
	}

	next := hres.Resource
	next.SetGroupVersionKind(r.config.Resource)

	err = r.Update(context.TODO(), next)
	if err != nil {
		r.log.Error(err, "Failed to update a resource", "namespace", namespace, "name", name)
		return reconcile.Result{}, err
	}

	r.log.Info("Resource updated", "namespace", namespace, "name", name, "phase", phase)

	return reconcile.Result{}, nil
}
