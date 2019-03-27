package reconciler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

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

var log = logf.Log.WithName("reconciler")

type Reconciler struct {
	client.Client
	config  *config.ControllerConfig
	handler handler.Handler
}

func NewReconciler(config *config.ControllerConfig) (*Reconciler, error) {
	h, err := exec.NewHandler(config.Reconciler.Exec)
	if err != nil {
		return nil, err
	}

	r := &Reconciler{
		config:  config,
		handler: h,
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
		log.Error(err, "Failed to get a resource", "namespace", namespace, "name", name)
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

	dependents := []unstructured.Unstructured{}
	for _, dep := range r.config.Dependents {
		dependentList := &unstructured.UnstructuredList{}
		dependentList.SetGroupVersionKind(dep)

		err := r.List(context.TODO(), dependentList, client.InNamespace(namespace))
		if err != nil {
			log.Error(err, "Failed to get a list for dependent resource", "namespace", namespace, "name", name)
			return reconcile.Result{}, err
		}

		for _, item := range dependentList.Items {
			ownerRefs := item.GetOwnerReferences()
			for _, ownerRef := range ownerRefs {
				if !reflect.DeepEqual(ownerRef, instanceRef) {
					continue
				}
				dependents = append(dependents, item)
			}
		}
	}

	state := NewState(instance, dependents)
	buf, err := json.Marshal(state)
	if err != nil {
		log.Error(err, "Failed to encode state", "namespace", namespace, "name", name)
		return reconcile.Result{}, err
	}

	out, err := r.handler.Run(buf)
	if err != nil {
		log.Error(err, "Handler error", "namespace", namespace, "name", name)
		return reconcile.Result{}, err
	}

	newState := &State{}
	err = json.Unmarshal(out, newState)
	if err != nil {
		log.Error(err, "Failed to decode new state", "namespace", namespace, "name", name)
		return reconcile.Result{}, err
	}

	err = r.validateState(state, newState)
	if err != nil {
		log.Error(err, "Ignored due to the new state is invalid", "namespace", namespace, "name", name)
		return reconcile.Result{}, nil
	}

	created, updated, deleted := state.Diff(newState)

	for _, res := range created {
		log.Info("Creating resource", "kind", res.GetKind(), "namespace", res.GetNamespace(), "name", res.GetName())

		res.SetOwnerReferences([]metav1.OwnerReference{instanceRef})
		err = r.Create(context.TODO(), &res)
		if err != nil {
			log.Error(err, "Failed to create a resource", "namespace", res.GetNamespace(), "name", res.GetName())
			return reconcile.Result{}, err
		}
	}

	for _, res := range updated {
		if res.GetSelfLink() != instance.GetSelfLink() {
			res.SetOwnerReferences([]metav1.OwnerReference{instanceRef})
		}

		log.Info("Updating resource", "kind", res.GetKind(), "namespace", res.GetNamespace(), "name", res.GetName())

		err = r.Update(context.TODO(), &res)
		if err != nil {
			log.Error(err, "Failed to update a resource", "namespace", res.GetNamespace(), "name", res.GetName())
			return reconcile.Result{}, err
		}

	}

	for _, res := range deleted {
		log.Info("Deleting resource", "kind", res.GetKind(), "namespace", res.GetNamespace(), "name", res.GetName())

		err = r.Delete(context.TODO(), &res)
		if err != nil {
			log.Error(err, "Failed to delete a resource", "namespace", res.GetNamespace(), "name", res.GetName())
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, nil
}

func (r *Reconciler) validateState(current, new *State) error {
	if new.Resource != nil {
		namespace := new.Resource.GetNamespace()

		if !reflect.DeepEqual(new.Resource.GroupVersionKind(), r.config.Resource) {
			return errors.New("resource: group/version/kind does not match")
		}
		if namespace != current.Resource.GetNamespace() {
			return errors.New("resource: namespace does not match")
		}
		if new.Resource.GetName() != current.Resource.GetName() {
			return errors.New("resource: name does not match")
		}

		for i, dep := range new.Dependents {
			if dep.GetNamespace() != namespace {
				return fmt.Errorf("dependents[%d]: namespace does not match", i)
			}
		}
	}

	for i, dep := range new.Dependents {
		if len(r.config.Dependents) == 0 {
			return errors.New("no dependents specified in the configuration")
		}

		matched := false
		for _, gvk := range r.config.Dependents {
			if reflect.DeepEqual(dep.GroupVersionKind(), gvk) {
				matched = true
				break
			}
		}

		if !matched {
			return fmt.Errorf("dependents[%d]: unexpected group/version/kind", i)
		}
	}

	return nil
}
