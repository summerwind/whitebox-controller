package reconciler

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
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
	"github.com/summerwind/whitebox-controller/reconciler/state"
)

var log = logf.Log.WithName("reconciler")

type Reconciler struct {
	client.Client
	config       *config.ControllerConfig
	handler      handler.Handler
	finalizer    handler.Handler
	recorder     record.EventRecorder
	requeueAfter *time.Duration
}

func New(c *config.ControllerConfig, rec record.EventRecorder) (*Reconciler, error) {
	h, err := common.NewHandler(&c.Reconciler.HandlerConfig)
	if err != nil {
		return nil, err
	}

	r := &Reconciler{
		config:   c,
		handler:  h,
		recorder: rec,
	}

	if c.Reconciler.RequeueAfter != "" {
		ra, err := time.ParseDuration(c.Reconciler.RequeueAfter)
		if err != nil {
			return nil, err
		}
		r.requeueAfter = &ra
	}

	if c.Finalizer != nil {
		fh, err := common.NewHandler(&c.Reconciler.HandlerConfig)
		if err != nil {
			return nil, err
		}
		r.finalizer = fh
	}

	return r, nil
}

func (r *Reconciler) InjectClient(c client.Client) error {
	r.Client = c
	return nil
}

func (r *Reconciler) Reconcile(req reconcile.Request) (reconcile.Result, error) {
	var (
		newState  *state.State
		err       error
		finalized bool
	)

	if r.IsObserver() {
		return r.Observe(req)
	}

	namespace := req.Namespace
	name := req.Name

	instance := &unstructured.Unstructured{}
	instance.SetGroupVersionKind(r.config.Resource)

	err = r.Get(context.TODO(), req.NamespacedName, instance)
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

	s := state.New(instance, dependents, refs)

	if isDeleting(instance) && r.finalizer != nil {
		log.Info("Starting finalizer", "namespace", namespace, "name", name)
		newState, err = r.finalizer.Finalize(s)
		finalized = true
	} else {
		newState, err = r.handler.Reconcile(s)
	}
	if err != nil {
		log.Error(err, "Handler error", "namespace", namespace, "name", name)
		return reconcile.Result{}, err
	}

	err = r.validateState(s, newState)
	if err != nil {
		log.Error(err, "Ignored due to the new state is invalid", "namespace", namespace, "name", name)
		return reconcile.Result{}, nil
	}

	if finalized {
		err := r.unsetFinalizer(newState.Object)
		if err != nil {
			log.Error(err, "Failed to unset finalizer from object metadata", "namespace", namespace, "name", name)
			return reconcile.Result{}, err
		}
	} else if r.finalizer != nil {
		err := r.setFinalizer(newState.Object)
		if err != nil {
			log.Error(err, "Failed to set finalizer from object metadata", "namespace", namespace, "name", name)
			return reconcile.Result{}, err
		}
	}

	r.setOwnerReference(newState, ownerRef)

	created, updated, deleted := s.Diff(newState)

	for _, res := range created {
		log.Info("Creating resource", "kind", res.GetKind(), "namespace", res.GetNamespace(), "name", res.GetName())

		err = r.Create(context.TODO(), res)
		if err != nil {
			log.Error(err, "Failed to create a resource", "namespace", res.GetNamespace(), "name", res.GetName())
			return reconcile.Result{}, err
		}
	}

	for _, res := range updated {
		log.Info("Updating resource", "kind", res.GetKind(), "namespace", res.GetNamespace(), "name", res.GetName())

		err = r.Update(context.TODO(), res)
		if err != nil {
			log.Error(err, "Failed to update a resource", "namespace", res.GetNamespace(), "name", res.GetName())
			return reconcile.Result{}, err
		}

	}

	for _, res := range deleted {
		log.Info("Deleting resource", "kind", res.GetKind(), "namespace", res.GetNamespace(), "name", res.GetName())

		err = r.Delete(context.TODO(), res)
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

	result := reconcile.Result{}
	if r.requeueAfter != nil {
		result.RequeueAfter = *r.requeueAfter
	}

	return result, nil
}

func (r *Reconciler) Observe(req reconcile.Request) (reconcile.Result, error) {
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

	s := state.State{
		Object: instance,
	}

	_, err = r.handler.Reconcile(&s)
	if err != nil {
		log.Error(err, "Handler error", "namespace", namespace, "name", name)
		return reconcile.Result{}, nil
	}

	return reconcile.Result{}, nil
}

func (r *Reconciler) IsObserver() bool {
	return r.config.Reconciler.Observe
}

// getDependents returns a list of dependent resources with
// an specified owner reference.
func (r *Reconciler) getDependents(res *unstructured.Unstructured, ownerRef metav1.OwnerReference) (map[string][]*unstructured.Unstructured, error) {
	dependents := map[string][]*unstructured.Unstructured{}

	for _, dep := range r.config.Dependents {
		key := getKindArg(dep.GroupVersionKind)
		dependents[key] = []*unstructured.Unstructured{}

		dependentList := &unstructured.UnstructuredList{}
		dependentList.SetGroupVersionKind(dep.GroupVersionKind)

		err := r.List(context.TODO(), dependentList, client.InNamespace(res.GetNamespace()))
		if err != nil {
			return nil, fmt.Errorf("Failed to get a list for dependent resource: %v", err)
		}

		for i := range dependentList.Items {
			depOwnerRefs := dependentList.Items[i].GetOwnerReferences()
			for _, ref := range depOwnerRefs {
				if !reflect.DeepEqual(ref, ownerRef) {
					continue
				}
				dependents[key] = append(dependents[key], &dependentList.Items[i])
			}
		}
	}

	return dependents, nil
}

// getReferences returns a list of reference resources based on
// spcified field path.
func (r *Reconciler) getReferences(res *unstructured.Unstructured) (map[string][]*unstructured.Unstructured, error) {
	refs := map[string][]*unstructured.Unstructured{}

	for _, ref := range r.config.References {
		if ref.NameFieldPath == "" {
			continue
		}

		key := getKindArg(ref.GroupVersionKind)
		refs[key] = []*unstructured.Unstructured{}

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

			refs[key] = append(refs[key], refRes)
		}
	}

	return refs, nil
}

// setFinalizer adds it's finalizer name to resource's metadata.
func (r *Reconciler) setFinalizer(res *unstructured.Unstructured) error {
	finalizers, ok, err := unstructured.NestedStringSlice(res.UnstructuredContent(), "metadata", "finalizers")
	if err != nil {
		return err
	}

	name := r.getFinalizerName()

	if ok {
		exist := false
		for i := range finalizers {
			if finalizers[i] == name {
				exist = true
			}
		}
		if exist {
			return nil
		}

		finalizers = append(finalizers, name)
	} else {
		finalizers = []string{name}
	}

	err = unstructured.SetNestedStringSlice(res.UnstructuredContent(), finalizers, "metadata", "finalizers")
	if err != nil {
		return err
	}

	return nil
}

// unsetFinalizer removes it's finalizer name from resource's metadata.
func (r *Reconciler) unsetFinalizer(res *unstructured.Unstructured) error {
	finalizers, ok, err := unstructured.NestedStringSlice(res.UnstructuredContent(), "metadata", "finalizers")
	if err != nil {
		return err
	}

	if !ok {
		return nil
	}

	new := []string{}
	name := r.getFinalizerName()
	exist := false

	for i := range finalizers {
		if finalizers[i] == name {
			exist = true
			continue
		}
		new = append(new, finalizers[i])
	}

	if !exist {
		return nil
	}

	err = unstructured.SetNestedStringSlice(res.UnstructuredContent(), new, "metadata", "finalizers")
	if err != nil {
		return err
	}

	return nil
}

// getFinalizerName returns controller's finalizer name.
func (r *Reconciler) getFinalizerName() string {
	return fmt.Sprintf("%s.%s", r.config.Name, r.config.Resource.Group)
}

// validateState validates a new state.
func (r *Reconciler) validateState(old *state.State, new *state.State) error {
	if new.Object != nil {
		namespace := new.Object.GetNamespace()

		if !reflect.DeepEqual(new.Object.GroupVersionKind(), old.Object.GroupVersionKind()) {
			return errors.New("resource: group/version/kind does not match")
		}
		if namespace != old.Object.GetNamespace() {
			return errors.New("resource: namespace does not match")
		}
		if new.Object.GetName() != old.Object.GetName() {
			return errors.New("resource: name does not match")
		}

		for key := range new.Dependents {
			for i, dep := range new.Dependents[key] {
				if dep.GetNamespace() != namespace {
					return fmt.Errorf("dependents[%s][%d]: namespace does not match", key, i)
				}
			}
		}
	}

	for key := range new.Dependents {
		if len(r.config.Dependents) == 0 {
			return errors.New("no dependents specified in the configuration")
		}

		matched := false
		for _, res := range r.config.Dependents {
			if key == getKindArg(res.GroupVersionKind) {
				matched = true
				break
			}
		}

		if !matched {
			return fmt.Errorf("dependents[%s]: unexpected group/version/kind", key)
		}
	}

	return nil
}

// setOwnerReference returns a state with specified owner reference.
func (r *Reconciler) setOwnerReference(s *state.State, ownerRef metav1.OwnerReference) *state.State {
	orphans := map[string]bool{}

	for _, dep := range r.config.Dependents {
		orphans[getKindArg(dep.GroupVersionKind)] = dep.Orphan
	}

	for key, deps := range s.Dependents {
		if orphans[key] {
			continue
		}

		for _, dep := range deps {
			dep.SetOwnerReferences([]metav1.OwnerReference{ownerRef})
		}
	}

	return s
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

// isDeleting returns whether the specified resource is being deleted.
func isDeleting(res *unstructured.Unstructured) bool {
	_, ok, err := unstructured.NestedString(res.UnstructuredContent(), "metadata", "deletionTimestamp")
	if err != nil {
		return false
	}

	return ok
}

// getKindArg returns string representation of GVK.
func getKindArg(gvk schema.GroupVersionKind) string {
	if gvk.Group == "" {
		return strings.ToLower(fmt.Sprintf("%s.%s", gvk.Kind, gvk.Version))
	}

	return strings.ToLower(fmt.Sprintf("%s.%s.%s", gvk.Kind, gvk.Version, gvk.Group))
}
