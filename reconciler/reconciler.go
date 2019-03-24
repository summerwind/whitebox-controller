package reconciler

import (
	"context"
	"reflect"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	namespace := req.Namespace
	name := req.Name

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

	enabled := true
	instanceRef := metav1.OwnerReference{
		APIVersion:         instance.GetAPIVersion(),
		Kind:               instance.GetKind(),
		Name:               instance.GetName(),
		UID:                instance.GetUID(),
		Controller:         &enabled,
		BlockOwnerDeletion: &enabled,
	}
	state := handler.NewState(handler.ActionReconcile, instance)

	for _, dep := range r.config.Dependents {
		dependentList := &unstructured.UnstructuredList{}
		dependentList.SetGroupVersionKind(dep)

		err := r.List(context.TODO(), &client.ListOptions{Namespace: namespace}, dependentList)
		if err != nil {
			r.log.Error(err, "Failed to get a list for dependent resource", "namespace", namespace, "name", name)
			return reconcile.Result{}, err
		}

		for _, item := range dependentList.Items {
			ownerRefs := item.GetOwnerReferences()
			for _, ownerRef := range ownerRefs {
				if !reflect.DeepEqual(ownerRef, instanceRef) {
					continue
				}
				state.Dependents = append(state.Dependents, item)
			}
		}
	}

	newState, err := r.handler.Run(state)
	if err != nil {
		r.log.Error(err, "Handler error", "namespace", namespace, "name", name)
		return reconcile.Result{}, err
	}

	created, updated, deleted := state.Diff(newState)

	for _, res := range created {
		res.SetOwnerReferences([]metav1.OwnerReference{instanceRef})
		err = r.Create(context.TODO(), &res)
		if err != nil {
			r.log.Error(err, "Failed to create a resource", "namespace", res.GetNamespace(), "name", res.GetName())
			return reconcile.Result{}, err
		}

		r.log.Info("Resource created", "kind", res.GetKind(), "namespace", res.GetNamespace(), "name", res.GetName())
	}

	for _, res := range updated {
		if res.GetSelfLink() != instance.GetSelfLink() {
			res.SetOwnerReferences([]metav1.OwnerReference{instanceRef})
		}

		err = r.Update(context.TODO(), &res)
		if err != nil {
			r.log.Error(err, "Failed to update a resource", "namespace", res.GetNamespace(), "name", res.GetName())
			return reconcile.Result{}, err
		}

		r.log.Info("Resource updated", "kind", res.GetKind(), "namespace", res.GetNamespace(), "name", res.GetName())
	}

	for _, res := range deleted {
		err = r.Delete(context.TODO(), &res)
		if err != nil {
			r.log.Error(err, "Failed to delete a resource", "namespace", res.GetNamespace(), "name", res.GetName())
			return reconcile.Result{}, err
		}

		r.log.Info("Resource deleted", "kind", res.GetKind(), "namespace", res.GetNamespace(), "name", res.GetName())
	}

	return reconcile.Result{}, nil
}
