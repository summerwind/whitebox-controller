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

// Reconciler represents a reconciler of controller.
type Reconciler struct {
	client.Client
	config       *config.ResourceConfig
	handler      handler.StateHandler
	finalizer    handler.StateHandler
	recorder     record.EventRecorder
	requeueAfter *time.Duration
}

// New returns a new reconciler.
func New(c *config.ResourceConfig, rec record.EventRecorder) (*Reconciler, error) {
	h, err := common.NewStateHandler(&c.Reconciler.HandlerConfig)
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
			return nil, errors.New("invalid requeue after")
		}
		r.requeueAfter = &ra
	}

	if c.Finalizer != nil {
		fh, err := common.NewStateHandler(c.Finalizer)
		if err != nil {
			return nil, err
		}
		r.finalizer = fh
	}

	return r, nil
}

// InjectClient implements inject.Client interface.
func (r *Reconciler) InjectClient(c client.Client) error {
	r.Client = c
	return nil
}

// Reconcile reconciles specified object.
func (r *Reconciler) Reconcile(req reconcile.Request) (reconcile.Result, error) {
	var (
		err       error
		finalized bool
	)

	if r.IsObserver() {
		return r.Observe(req)
	}

	namespace := req.Namespace
	name := req.Name

	instance := &unstructured.Unstructured{}
	instance.SetGroupVersionKind(r.config.GroupVersionKind)

	err = r.Get(context.TODO(), req.NamespacedName, instance)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		log.Error(err, "Failed to get a resource", "namespace", namespace, "name", name)
		return reconcile.Result{}, err
	}

	dependents, err := r.getDependents(instance)
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
	ns := s.Copy()

	if isDeleting(instance) && r.finalizer != nil {
		finalized = true
		log.Info("Starting finalizer", "namespace", namespace, "name", name)
		err = r.finalizer.HandleState(ns)
	} else {
		err = r.handler.HandleState(ns)
	}
	if err != nil {
		log.Error(err, "Handler error", "namespace", namespace, "name", name)
		return reconcile.Result{}, err
	}

	err = r.validateState(s, ns)
	if err != nil {
		log.Error(err, "The new state is invalid", "namespace", namespace, "name", name)
		return reconcile.Result{}, err
	}

	r.setOwnerReference(ns)

	if finalized {
		if !ns.Requeue && ns.RequeueAfter == 0 {
			r.unsetFinalizer(ns.Object)
		}
	} else if r.finalizer != nil {
		r.setFinalizer(ns.Object)
	}

	created, updated, deleted := s.Diff(ns)

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

	for _, ev := range ns.Events {
		err := ev.Validate()
		if err != nil {
			log.Info("Ignored event due to the event is invalid", "namespace", namespace, "name", name, "error", err.Error())
			continue
		}
		r.recorder.Event(instance, ev.Type, ev.Reason, ev.Message)
	}

	result := reconcile.Result{}
	if r.requeueAfter != nil {
		result.RequeueAfter = *r.requeueAfter
	}

	result.Requeue = ns.Requeue
	if ns.RequeueAfter > 0 {
		result.RequeueAfter = time.Duration(ns.RequeueAfter) * time.Second
	}

	return result, nil
}

func (r *Reconciler) Observe(req reconcile.Request) (reconcile.Result, error) {
	namespace := req.Namespace
	name := req.Name

	instance := &unstructured.Unstructured{}
	instance.SetGroupVersionKind(r.config.GroupVersionKind)

	err := r.Get(context.TODO(), req.NamespacedName, instance)
	if err != nil && !apierrors.IsNotFound(err) {
		log.Error(err, "Failed to get a resource", "namespace", namespace, "name", name)
		return reconcile.Result{}, nil
	}

	// This allows determination of deleted resources
	instance.SetNamespace(namespace)
	instance.SetName(name)

	s := &state.State{
		Object: instance,
	}

	err = r.handler.HandleState(s)
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
func (r *Reconciler) getDependents(res *unstructured.Unstructured) (map[string][]*unstructured.Unstructured, error) {
	dependents := map[string][]*unstructured.Unstructured{}
	ownerRef := metav1.NewControllerRef(res, res.GroupVersionKind())

	for _, dep := range r.config.Dependents {
		key := state.ResourceKey(dep.GroupVersionKind)
		dependents[key] = []*unstructured.Unstructured{}

		gvk := dep.GroupVersionKind
		gvk.Kind = gvk.Kind + "List"
		dependentList := &unstructured.UnstructuredList{}
		dependentList.SetGroupVersionKind(gvk)

		err := r.List(context.TODO(), dependentList, client.InNamespace(res.GetNamespace()))
		if err != nil {
			return nil, fmt.Errorf("Failed to get a list for dependent resource: %v", err)
		}

		for i := range dependentList.Items {
			depOwnerRefs := dependentList.Items[i].GetOwnerReferences()
			for _, ref := range depOwnerRefs {
				if !reflect.DeepEqual(ref, *ownerRef) {
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

		key := state.ResourceKey(ref.GroupVersionKind)
		refs[key] = []*unstructured.Unstructured{}

		refNames, err := getReferenceNames(res, ref.NameFieldPath)
		if err != nil {
			return nil, fmt.Errorf("failed to get reference name list: %v", err)
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
				return nil, fmt.Errorf("failed to get a resource '%s/%s': %v", res.GetNamespace(), refNames[i], err)
			}

			refs[key] = append(refs[key], refRes)
		}
	}

	return refs, nil
}

// setFinalizer adds it's finalizer name to resource's metadata.
func (r *Reconciler) setFinalizer(res *unstructured.Unstructured) {
	if res == nil {
		return
	}

	finalizers := res.GetFinalizers()
	name := r.getFinalizerName()

	exist := false
	for i := range finalizers {
		if finalizers[i] == name {
			exist = true
		}
	}

	if !exist {
		finalizers = append(finalizers, name)
		res.SetFinalizers(finalizers)
	}
}

// unsetFinalizer removes it's finalizer name from resource's metadata.
func (r *Reconciler) unsetFinalizer(res *unstructured.Unstructured) {
	if res == nil {
		return
	}

	finalizers := res.GetFinalizers()
	name := r.getFinalizerName()

	list := []string{}
	exist := false
	for i := range finalizers {
		if finalizers[i] == name {
			exist = true
			continue
		}
		list = append(list, finalizers[i])
	}

	if exist {
		res.SetFinalizers(list)
	}
}

// getFinalizerName returns controller's finalizer name.
func (r *Reconciler) getFinalizerName() string {
	return fmt.Sprintf("%s-controller.%s", strings.ToLower(r.config.Kind), r.config.Group)
}

// validateState validates specified state.
func (r *Reconciler) validateState(s, ns *state.State) error {
	if ns.Object != nil {
		if !reflect.DeepEqual(r.config.GroupVersionKind, ns.Object.GroupVersionKind()) {
			return errors.New("object: unexpected group/version/kind")
		}
		if ns.Object.GetNamespace() != s.Object.GetNamespace() {
			return errors.New("object: changing namespace is not allowed")
		}
		if ns.Object.GetName() != s.Object.GetName() {
			return errors.New("object: changing name is not allowed")
		}
		if ns.Object.GetUID() != s.Object.GetUID() {
			return errors.New("object: changing UID is not allowed")
		}
	}

	keys := map[string]struct{}{}
	for _, dep := range r.config.Dependents {
		keys[state.ResourceKey(dep.GroupVersionKind)] = struct{}{}
	}

	for key := range ns.Dependents {
		_, ok := keys[key]
		if !ok {
			return fmt.Errorf("dependents[%s]: unexpected group/version/kind", key)
		}

		for i, dep := range ns.Dependents[key] {
			if key != state.ResourceKey(dep.GroupVersionKind()) {
				return fmt.Errorf("dependents[%s][%d]: namespace does not match", key, i)
			}
		}
	}

	return nil
}

// setOwnerReference sets OwnerReference to dependent resources.
func (r *Reconciler) setOwnerReference(s *state.State) {
	if s.Object == nil {
		return
	}

	ownerRef := metav1.NewControllerRef(s.Object, s.Object.GroupVersionKind())

	orphans := map[string]struct{}{}
	for _, dep := range r.config.Dependents {
		if dep.Orphan {
			orphans[state.ResourceKey(dep.GroupVersionKind)] = struct{}{}
		}
	}

	for key, deps := range s.Dependents {
		_, ok := orphans[key]
		if ok {
			continue
		}

		for _, dep := range deps {
			dep.SetOwnerReferences([]metav1.OwnerReference{*ownerRef})
		}
	}
}

// getReferenceNames returns a list of reference resource names based
// on JSON Path and resource.
func getReferenceNames(res *unstructured.Unstructured, namePath string) ([]string, error) {
	jp := jsonpath.New("reference")
	jp.AllowMissingKeys(true)

	err := jp.Parse(fmt.Sprintf("{%s}", namePath))
	if err != nil {
		return []string{}, err
	}

	results, err := jp.FindResults(res.Object)
	if err != nil {
		return []string{}, err
	}

	nameMap := map[string]bool{}
	for x := range results {
		for _, v := range results[x] {
			val, ok := template.PrintableValue(v)
			if !ok {
				return nil, fmt.Errorf("value is not a printable type %s", v.Type())
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
