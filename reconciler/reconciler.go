package reconciler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/third_party/forked/golang/template"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/jsonpath"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"

	"github.com/summerwind/whitebox-controller/config"
	"github.com/summerwind/whitebox-controller/handler"
	"github.com/summerwind/whitebox-controller/handler/common"
)

var log = logf.Log.WithName("reconciler")

type Reconciler struct {
	client.Client
	config   *config.ControllerConfig
	handler  handler.Handler
	recorder record.EventRecorder
}

func New(c *config.ControllerConfig, rec record.EventRecorder) (*Reconciler, error) {
	h, err := common.NewHandler(c.Reconciler)
	if err != nil {
		return nil, err
	}

	r := &Reconciler{
		config:   c,
		handler:  h,
		recorder: rec,
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

	ownerRef := newOwnerReference(instance)
	dependents, err := r.getDependents(instance, ownerRef)
	if err != nil {
		log.Error(err, "Failed to get dependent resources", "namespace", namespace, "name", name)
		return reconcile.Result{}, err
	}

	refs, err := r.getReferences(instance)
	if err != nil {
		log.Error(err, "Failed to get reference resources", "namespace", namespace, "name", name)
		return reconcile.Result{}, err
	}

	state := NewState(instance, dependents, refs)
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

	if len(out) == 0 {
		err := errors.New("empty state")
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

		res.SetOwnerReferences([]metav1.OwnerReference{ownerRef})
		err = r.Create(context.TODO(), &res)
		if err != nil {
			log.Error(err, "Failed to create a resource", "namespace", res.GetNamespace(), "name", res.GetName())
			return reconcile.Result{}, err
		}
	}

	for _, res := range updated {
		if res.GetSelfLink() != instance.GetSelfLink() {
			res.SetOwnerReferences([]metav1.OwnerReference{ownerRef})
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

	for _, ev := range newState.Events {
		if ev.Empty() {
			log.Info("Ignored event due to the event is invalid", "namespace", namespace, "name", name, "event", ev)
			continue
		}
		r.recorder.Event(instance, ev.Type, ev.Reason, ev.Message)
	}

	return reconcile.Result{}, nil
}

// getDependents returns a list of dependent resources with
// an specified owner reference.
func (r *Reconciler) getDependents(res *unstructured.Unstructured, ownerRef metav1.OwnerReference) ([]unstructured.Unstructured, error) {
	dependents := []unstructured.Unstructured{}

	for _, dep := range r.config.Dependents {
		dependentList := &unstructured.UnstructuredList{}
		dependentList.SetGroupVersionKind(dep)

		err := r.List(context.TODO(), dependentList, client.InNamespace(res.GetNamespace()))
		if err != nil {
			return nil, fmt.Errorf("Failed to get a list for dependent resource: %v", err)
		}

		for _, item := range dependentList.Items {
			depOwnerRefs := item.GetOwnerReferences()
			for _, ref := range depOwnerRefs {
				if !reflect.DeepEqual(ref, ownerRef) {
					continue
				}
				dependents = append(dependents, item)
			}
		}
	}

	return dependents, nil
}

// getReferences returns a list of reference resources based on
// spcified field path.
func (r *Reconciler) getReferences(res *unstructured.Unstructured) ([]unstructured.Unstructured, error) {
	refs := []unstructured.Unstructured{}

	for _, ref := range r.config.References {
		if ref.NameFieldPath == "" {
			continue
		}

		refNames, err := getNamesFromField(ref.NameFieldPath, res)
		if err != nil {
			return nil, fmt.Errorf("Failed to get reference name list: %v", err)
		}

		if len(refNames) == 0 {
			continue
		}

		for i := range refNames {
			refRes := &unstructured.Unstructured{}
			refRes.SetGroupVersionKind(ref.GroupVersionKind)

			nn := types.NamespacedName{
				Namespace: res.GetNamespace(),
				Name:      refNames[i],
			}
			err = r.Get(context.TODO(), nn, refRes)
			if err != nil {
				if apierrors.IsNotFound(err) {
					continue
				}
				return nil, fmt.Errorf("Failed to get a resource '%s/%s': %v", res.GetNamespace(), refNames[i], err)
			}

			refs = append(refs, *refRes)
		}
	}

	return refs, nil
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

// newOwnerReference creates and returns an OwnerReference based on
// specified resource's GroupVersionKind.
func newOwnerReference(res *unstructured.Unstructured) metav1.OwnerReference {
	enabled := true

	return metav1.OwnerReference{
		APIVersion:         res.GetAPIVersion(),
		Kind:               res.GetKind(),
		Name:               res.GetName(),
		UID:                res.GetUID(),
		Controller:         &enabled,
		BlockOwnerDeletion: &enabled,
	}
}

// getNamesFromField returns a list of reference resource names based
// on JSON Path and resource.
func getNamesFromField(namePath string, res *unstructured.Unstructured) ([]string, error) {
	j := jsonpath.New("reference")
	j.AllowMissingKeys(true)

	err := j.Parse(fmt.Sprintf("{%s}", namePath))
	if err != nil {
		return []string{}, err
	}

	results, err := j.FindResults(res.Object)
	if err != nil {
		return []string{}, err
	}

	nameMap := map[string]bool{}
	for x := range results {
		for _, v := range results[x] {
			val, ok := template.PrintableValue(v)
			if !ok {
				return nil, fmt.Errorf("can't print type %s", v.Type())
			}

			var buf bytes.Buffer
			fmt.Fprint(&buf, val)
			nameMap[buf.String()] = true
		}
	}

	names := []string{}
	for key, _ := range nameMap {
		names = append(names, key)
	}

	return names, nil
}
