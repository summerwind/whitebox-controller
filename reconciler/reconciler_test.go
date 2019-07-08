package reconciler

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	. "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/summerwind/whitebox-controller/config"
	"github.com/summerwind/whitebox-controller/reconciler/state"
)

var kconfig *rest.Config

func TestMain(m *testing.M) {
	var err error

	env := &envtest.Environment{
		CRDs: []*apiextensionsv1beta1.CustomResourceDefinition{newCRD("Test")},
	}

	kconfig, err = env.Start()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to start test environment: %v\n", err)
		os.Exit(1)
	}

	code := m.Run()
	env.Stop()
	os.Exit(code)
}

func TestReconcile(t *testing.T) {
	var (
		nn  types.NamespacedName
		err error
	)

	RegisterTestingT(t)

	rc := newResourceConfig()
	recorder := record.NewFakeRecorder(32)
	r, err := New(rc, recorder)
	Expect(err).NotTo(HaveOccurred())

	c := newClient()
	r.InjectClient(c)

	// Create target object
	object := newObject(rc.GroupVersionKind, "test")
	err = r.Create(context.TODO(), object)
	Expect(err).NotTo(HaveOccurred())
	defer r.Delete(context.TODO(), object)

	// Generate owner reference from target object
	ownerRef := metav1.NewControllerRef(object, object.GroupVersionKind())

	// Generate dependent objects
	p1 := newPod("p1")
	p1.SetOwnerReferences([]metav1.OwnerReference{*ownerRef})
	r.Create(context.TODO(), p1)
	defer r.Delete(context.TODO(), p1)

	p2 := newPod("p2")
	p2.SetOwnerReferences([]metav1.OwnerReference{*ownerRef})
	r.Create(context.TODO(), p2)
	defer r.Delete(context.TODO(), p2)

	p3 := newPod("p3")
	p3.SetOwnerReferences([]metav1.OwnerReference{*ownerRef})
	defer r.Delete(context.TODO(), p3)

	// Generate reference objects
	c1 := newConfigMap("c1")
	r.Create(context.TODO(), c1)
	defer r.Delete(context.TODO(), c1)

	// Enable test handler
	h := &testHandler{}
	r.handler = h

	// Set reconcile handler
	h.Func = func(s *state.State) error {
		var err error

		Expect(s.Object).NotTo(BeNil())
		Expect(len(s.Dependents["pod.v1"])).To(Equal(2))
		Expect(len(s.References["configmap.v1"])).To(Equal(1))

		err = SetNestedField(s.Object.Object, "completed", "status", "phase")
		Expect(err).NotTo(HaveOccurred())

		p1 := s.Dependents["pod.v1"][0]
		Expect(p1.GetName()).To(Equal("p1"))
		err = SetNestedField(p1.Object, int64(120), "spec", "activeDeadlineSeconds")
		Expect(err).NotTo(HaveOccurred())

		p3 := s.Dependents["pod.v1"][1]
		Expect(p3.GetName()).To(Equal("p2"))
		p3.SetName("p3")
		RemoveNestedField(p3.Object, "metadata", "resourceVersion")

		c1 := s.References["configmap.v1"][0]
		Expect(c1.GetName()).To(Equal("c1"))

		s.Events = append(s.Events, state.Event{
			Type: "Succeeded",
		})
		s.Events = append(s.Events, state.Event{})

		s.RequeueAfter = 60

		return nil
	}

	// Run reconcile function
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: object.GetNamespace(),
			Name:      object.GetName(),
		},
	}
	result, err := r.Reconcile(req)
	Expect(err).NotTo(HaveOccurred())
	Expect(result.RequeueAfter).To(Equal(time.Duration(60) * time.Second))

	// Test target object state
	o := &Unstructured{}
	o.SetGroupVersionKind(object.GroupVersionKind())
	err = c.Get(context.TODO(), req.NamespacedName, o)
	Expect(err).NotTo(HaveOccurred())

	phase, ok, err := NestedString(o.Object, "status", "phase")
	Expect(err).NotTo(HaveOccurred())
	Expect(ok).To(BeTrue())
	Expect(phase).To(Equal("completed"))

	// Test p1 object state
	nn = types.NamespacedName{
		Namespace: p1.GetNamespace(),
		Name:      p1.GetName(),
	}
	pod := &corev1.Pod{}
	err = c.Get(context.TODO(), nn, pod)
	Expect(err).NotTo(HaveOccurred())
	Expect(*(pod.Spec.ActiveDeadlineSeconds)).To(Equal(int64(120)))

	// Test p2 object state
	nn = types.NamespacedName{
		Namespace: p2.GetNamespace(),
		Name:      p2.GetName(),
	}
	pod = &corev1.Pod{}
	err = c.Get(context.TODO(), nn, pod)
	Expect(apierrors.IsNotFound(err)).To(BeTrue())

	// Test p3 object state
	nn = types.NamespacedName{
		Namespace: p3.GetNamespace(),
		Name:      p3.GetName(),
	}
	pod = &corev1.Pod{}
	err = c.Get(context.TODO(), nn, pod)
	Expect(err).NotTo(HaveOccurred())

	// Test event
	Expect(len(recorder.Events)).To(Equal(1))
}

func TestReconcileWithFinalizer(t *testing.T) {
	RegisterTestingT(t)

	rc := newResourceConfig()
	recorder := record.NewFakeRecorder(32)
	r, err := New(rc, recorder)
	Expect(err).NotTo(HaveOccurred())

	c := newClient()
	r.InjectClient(c)

	// Create target object
	object := newObject(rc.GroupVersionKind, "test")
	err = r.Create(context.TODO(), object)
	Expect(err).NotTo(HaveOccurred())
	defer r.Delete(context.TODO(), object)

	// Enable test handler
	h := &testHandler{}
	r.finalizer = h

	// Set finalize handler
	h.Func = func(s *state.State) error {
		s.Events = append(s.Events, state.Event{
			Type: "Finalized",
		})
		return nil
	}

	// Run reconcile function
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: object.GetNamespace(),
			Name:      object.GetName(),
		},
	}
	_, err = r.Reconcile(req)
	Expect(err).NotTo(HaveOccurred())

	// Delete target object
	r.Delete(context.TODO(), object)

	// Run reconcile function again
	_, err = r.Reconcile(req)
	Expect(err).NotTo(HaveOccurred())

	// Test target object state
	o := &Unstructured{}
	o.SetGroupVersionKind(object.GroupVersionKind())
	err = c.Get(context.TODO(), req.NamespacedName, o)
	Expect(apierrors.IsNotFound(err)).To(BeTrue())

	// Test event
	Expect(len(recorder.Events)).To(Equal(1))
}

func TestReconcileWithObserve(t *testing.T) {
	RegisterTestingT(t)

	rc := newResourceConfig()
	rc.Reconciler.Observe = true
	recorder := record.NewFakeRecorder(32)
	r, err := New(rc, recorder)
	Expect(err).NotTo(HaveOccurred())

	c := newClient()
	r.InjectClient(c)

	// Create target object
	object := newObject(rc.GroupVersionKind, "test")
	err = r.Create(context.TODO(), object)
	Expect(err).NotTo(HaveOccurred())
	defer r.Delete(context.TODO(), object)

	// Enable test handler
	h := &testHandler{}
	r.handler = h

	// Set reconcile handler
	h.Func = func(s *state.State) error {
		err = SetNestedField(s.Object.Object, "completed", "status", "phase")
		Expect(err).NotTo(HaveOccurred())

		s.Events = append(s.Events, state.Event{
			Type: "Observed",
		})

		return nil
	}

	// Run reconcile function
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: object.GetNamespace(),
			Name:      object.GetName(),
		},
	}
	_, err = r.Reconcile(req)
	Expect(err).NotTo(HaveOccurred())

	// Test target object state
	o := &Unstructured{}
	o.SetGroupVersionKind(object.GroupVersionKind())
	err = c.Get(context.TODO(), req.NamespacedName, o)
	Expect(err).NotTo(HaveOccurred())

	_, ok, err := NestedString(o.Object, "status", "phase")
	Expect(err).NotTo(HaveOccurred())
	Expect(ok).To(BeFalse())

	// Test event
	Expect(len(recorder.Events)).To(Equal(0))
}

func TestReconcileWithNoObject(t *testing.T) {
	RegisterTestingT(t)

	rc := newResourceConfig()
	recorder := record.NewFakeRecorder(32)
	r, err := New(rc, recorder)
	Expect(err).NotTo(HaveOccurred())

	c := newClient()
	r.InjectClient(c)

	// Enable test handler
	h := &testHandler{}
	r.handler = h

	// Set reconcile handler
	h.Func = func(s *state.State) error {
		return errors.New("handler error")
	}

	// Run reconcile function
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: "default",
			Name:      "test",
		},
	}
	_, err = r.Reconcile(req)
	Expect(err).NotTo(HaveOccurred())
}

func TestReconcileWithObjectDeletion(t *testing.T) {
	RegisterTestingT(t)

	rc := newResourceConfig()
	recorder := record.NewFakeRecorder(32)
	r, err := New(rc, recorder)
	Expect(err).NotTo(HaveOccurred())

	c := newClient()
	r.InjectClient(c)

	// Create target object
	object := newObject(rc.GroupVersionKind, "test")
	err = r.Create(context.TODO(), object)
	Expect(err).NotTo(HaveOccurred())
	defer r.Delete(context.TODO(), object)

	// Enable test handler
	h := &testHandler{}
	r.handler = h

	// Set reconcile handler
	h.Func = func(s *state.State) error {
		s.Object = nil
		return nil
	}

	// Run reconcile function
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: object.GetNamespace(),
			Name:      object.GetName(),
		},
	}
	_, err = r.Reconcile(req)
	Expect(err).NotTo(HaveOccurred())

	// Test target object state
	o := &Unstructured{}
	o.SetGroupVersionKind(object.GroupVersionKind())
	err = c.Get(context.TODO(), req.NamespacedName, o)
	Expect(apierrors.IsNotFound(err)).To(BeTrue())
}

func TestReconcileWithHandlerError(t *testing.T) {
	RegisterTestingT(t)

	rc := newResourceConfig()
	recorder := record.NewFakeRecorder(32)
	r, err := New(rc, recorder)
	Expect(err).NotTo(HaveOccurred())

	c := newClient()
	r.InjectClient(c)

	// Create target object
	object := newObject(rc.GroupVersionKind, "test")
	err = r.Create(context.TODO(), object)
	Expect(err).NotTo(HaveOccurred())
	defer r.Delete(context.TODO(), object)

	// Enable test handler
	h := &testHandler{}
	r.handler = h

	// Set reconcile handler
	h.Func = func(s *state.State) error {
		return errors.New("handler error")
	}

	// Run reconcile function
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: object.GetNamespace(),
			Name:      object.GetName(),
		},
	}
	_, err = r.Reconcile(req)
	Expect(err).To(HaveOccurred())
}

func TestReconcileWithInvalidState(t *testing.T) {
	RegisterTestingT(t)

	rc := newResourceConfig()
	recorder := record.NewFakeRecorder(32)
	r, err := New(rc, recorder)
	Expect(err).NotTo(HaveOccurred())

	c := newClient()
	r.InjectClient(c)

	// Create target object
	object := newObject(rc.GroupVersionKind, "test")
	err = r.Create(context.TODO(), object)
	Expect(err).NotTo(HaveOccurred())
	defer r.Delete(context.TODO(), object)

	// Enable test handler
	h := &testHandler{}
	r.handler = h

	// Set reconcile handler
	h.Func = func(s *state.State) error {
		s.Object.SetName("invalid")
		return nil
	}

	// Run reconcile function
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: object.GetNamespace(),
			Name:      object.GetName(),
		},
	}
	_, err = r.Reconcile(req)
	Expect(err).To(HaveOccurred())
}

func TestObserve(t *testing.T) {
	RegisterTestingT(t)

	rc := newResourceConfig()
	r, err := New(rc, nil)
	Expect(err).NotTo(HaveOccurred())

	h := &testHandler{}
	rc.Reconciler.HandlerConfig.StateHandler = h

	c := newClient()
	r.InjectClient(c)

	object := newObject(rc.GroupVersionKind, "test")
	err = c.Create(context.TODO(), object)
	Expect(err).NotTo(HaveOccurred())
	defer c.Delete(context.TODO(), object)

	tests := []struct {
		name string
		err  error
	}{
		{object.GetName(), nil},
		{"invalid", nil},
		{object.GetName(), errors.New("error")},
	}

	for _, test := range tests {
		h.Func = func(s *state.State) error {
			return test.err
		}

		req := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Namespace: "default",
				Name:      test.name,
			},
		}

		_, err = r.Observe(req)
		Expect(err).NotTo(HaveOccurred())
	}
}

func TestGetDependents(t *testing.T) {
	RegisterTestingT(t)

	rc := newResourceConfig()
	r, err := New(rc, nil)
	Expect(err).NotTo(HaveOccurred())

	c := newClient()
	r.InjectClient(c)

	object := newObject(rc.GroupVersionKind, "test")
	ownerRef := metav1.NewControllerRef(object, object.GroupVersionKind())

	p1 := newPod("p1")
	p1.SetOwnerReferences([]metav1.OwnerReference{*ownerRef})
	err = c.Create(context.TODO(), p1)
	Expect(err).NotTo(HaveOccurred())
	defer c.Delete(context.TODO(), p1)

	p2 := newPod("p2")
	err = c.Create(context.TODO(), p2)
	Expect(err).NotTo(HaveOccurred())
	defer c.Delete(context.TODO(), p2)

	deps, err := r.getDependents(object)
	Expect(err).NotTo(HaveOccurred())
	Expect(len(deps["pod.v1"])).To(Equal(1))
}

func TestGetReferences(t *testing.T) {
	RegisterTestingT(t)

	rc := newResourceConfig()
	r, err := New(rc, nil)
	Expect(err).NotTo(HaveOccurred())

	c := newClient()
	r.InjectClient(c)

	object := newObject(rc.GroupVersionKind, "test")

	c1 := newConfigMap("c1")
	err = c.Create(context.TODO(), c1)
	Expect(err).NotTo(HaveOccurred())
	defer c.Delete(context.TODO(), c1)

	tests := []struct {
		nameFieldPath string
		length        int
		err           bool
	}{
		{"", 0, false},
		{".spec.configMapRefs[0]", 1, false},
		{".spec.configMapRefs[1]", 0, false},
		{".spec.refs[0]", 0, false},
		{"spec.configMapRefs[0]", 0, true},
	}

	for _, test := range tests {
		rc.References[0].NameFieldPath = test.nameFieldPath

		refs, err := r.getReferences(object)
		if test.err {
			Expect(err).To(HaveOccurred())
		} else {
			Expect(err).NotTo(HaveOccurred())
			Expect(len(refs["configmap.v1"])).To(Equal(test.length))
		}
	}
}

func TestSetFinalizer(t *testing.T) {
	RegisterTestingT(t)

	rc := newResourceConfig()
	r, err := New(rc, nil)
	Expect(err).NotTo(HaveOccurred())

	finalizers := []string{
		"foo-controller.example.com",
	}

	object := newObject(rc.GroupVersionKind, "test")
	object.SetFinalizers(finalizers)

	r.setFinalizer(object)
	list := object.GetFinalizers()
	Expect(len(list)).To(Equal(2))
	Expect(list[0]).To(Equal(finalizers[0]))
}

func TestUnsetFinalizer(t *testing.T) {
	RegisterTestingT(t)

	rc := newResourceConfig()
	r, err := New(rc, nil)
	Expect(err).NotTo(HaveOccurred())

	finalizers := []string{
		"foo-controller.example.com",
		"test-controller.example.com",
	}

	object := newObject(rc.GroupVersionKind, "test")
	object.SetFinalizers(finalizers)

	r.unsetFinalizer(object)
	list := object.GetFinalizers()
	Expect(len(list)).To(Equal(1))
	Expect(list[0]).To(Equal(finalizers[0]))
}

func TestValidateState(t *testing.T) {
	var err error

	RegisterTestingT(t)

	rc := newResourceConfig()
	r, err := New(rc, nil)
	Expect(err).NotTo(HaveOccurred())

	s := newState(rc)

	// Valid state
	s1 := s.Copy()
	err = r.validateState(s, s1)
	Expect(err).NotTo(HaveOccurred())

	// Invalid state with unexpected object
	s2 := s.Copy()
	s2.Object.SetKind("Invalid")
	err = r.validateState(s, s2)
	Expect(err).To(HaveOccurred())

	// Invalid state with changed namespace
	s3 := s.Copy()
	s3.Object.SetNamespace("Invalid")
	err = r.validateState(s, s3)
	Expect(err).To(HaveOccurred())

	// Invalid state with changed name
	s4 := s.Copy()
	s4.Object.SetName("Invalid")
	err = r.validateState(s, s4)
	Expect(err).To(HaveOccurred())

	// Invalid state with changed UID
	s5 := s.Copy()
	s5.Object.SetUID("Invalid")
	err = r.validateState(s, s5)
	Expect(err).To(HaveOccurred())

	// Invalid state with unexpected dependent key
	s6 := s.Copy()
	s6.Dependents["invalid.v1alpha1.example.com"] = s2.Dependents["pod.v1"]
	err = r.validateState(s, s6)
	Expect(err).To(HaveOccurred())

	// Invalid state with unexpected dependent object
	s7 := s.Copy()
	s7.Dependents["pod.v1"][0].SetKind("Invalid")
	err = r.validateState(s, s7)
	Expect(err).To(HaveOccurred())
}

func TestSetOwnerReference(t *testing.T) {
	RegisterTestingT(t)

	rc := newResourceConfig()
	r, err := New(rc, nil)
	Expect(err).NotTo(HaveOccurred())

	s := newState(rc)
	r.setOwnerReference(s)

	for depKey := range s.Dependents {
		for _, dep := range s.Dependents[depKey] {
			ownerRefs := dep.GetOwnerReferences()
			Expect(len(ownerRefs)).To(Equal(1))
			Expect(ownerRefs[0].Name).To(Equal(s.Object.GetName()))
			Expect(ownerRefs[0].UID).To(Equal(s.Object.GetUID()))
		}
	}
}

func TestGetReferenceNames(t *testing.T) {
	var (
		refs []string
		err  error
	)

	RegisterTestingT(t)

	rc := newResourceConfig()
	object := newObject(rc.GroupVersionKind, "test")

	names := []string{"test1", "test2"}
	SetNestedStringSlice(object.Object, names, "spec", "refs")

	refs, err = getReferenceNames(object, ".spec.refs[0]")
	Expect(err).NotTo(HaveOccurred())
	Expect(len(refs)).To(Equal(1))
	Expect(refs[0]).To(Equal("test1"))

	refs, err = getReferenceNames(object, ".spec.refs[*]")
	Expect(err).NotTo(HaveOccurred())
	Expect(len(refs)).To(Equal(2))
	Expect(refs).To(ConsistOf(names))

	refs, err = getReferenceNames(object, ".spec.deps[0]")
	Expect(err).NotTo(HaveOccurred())
	Expect(len(refs)).To(Equal(0))
}

func TestIsDeleting(t *testing.T) {
	RegisterTestingT(t)

	rc := newResourceConfig()
	object := newObject(rc.GroupVersionKind, "test")
	deleting := newObject(rc.GroupVersionKind, "test")
	SetNestedField(deleting.Object, string(time.Now().Unix()), "metadata", "deletionTimestamp")

	Expect(isDeleting(object)).To(BeFalse())
	Expect(isDeleting(deleting)).To(BeTrue())
}

type testHandler struct {
	Func func(*state.State) error
}

func (h *testHandler) HandleState(s *state.State) error {
	if h.Func != nil {
		return h.Func(s)
	}
	return nil
}

func newClient() client.Client {
	cl, err := client.New(kconfig, client.Options{})
	Expect(err).NotTo(HaveOccurred())

	return cl
}

func newResourceConfig() *config.ResourceConfig {
	return &config.ResourceConfig{
		GroupVersionKind: schema.GroupVersionKind{
			Group:   "example.com",
			Version: "v1alpha1",
			Kind:    "Test",
		},
		Dependents: []config.DependentConfig{
			config.DependentConfig{
				GroupVersionKind: schema.GroupVersionKind{
					Version: "v1",
					Kind:    "Pod",
				},
				Orphan: false,
			},
		},
		References: []config.ReferenceConfig{
			config.ReferenceConfig{
				GroupVersionKind: schema.GroupVersionKind{
					Version: "v1",
					Kind:    "ConfigMap",
				},
				NameFieldPath: ".spec.configMapRefs[*]",
			},
		},
		Reconciler: &config.ReconcilerConfig{
			HandlerConfig: config.HandlerConfig{
				StateHandler: &testHandler{},
			},
			RequeueAfter: "30s",
			Observe:      false,
		},
	}
}

func newState(rc *config.ResourceConfig) *state.State {
	s := &state.State{
		Object:     newObject(rc.GroupVersionKind, "test"),
		Dependents: map[string][]*Unstructured{},
		References: map[string][]*Unstructured{},
	}

	for i := range rc.Dependents {
		gvk := rc.Dependents[i].GroupVersionKind
		dep := newObject(gvk, "")
		depKey := state.ResourceKey(gvk)
		s.Dependents[depKey] = []*Unstructured{}

		depNames := []string{"test1", "test2"}
		for _, name := range depNames {
			d := dep.DeepCopy()
			d.SetName(name)
			d.SetUID(uuid.NewUUID())
			s.Dependents[depKey] = append(s.Dependents[depKey], d)
		}
	}

	for i := range rc.References {
		gvk := rc.References[i].GroupVersionKind
		ref := newObject(gvk, "")
		refKey := state.ResourceKey(gvk)
		s.References[refKey] = []*Unstructured{}

		refNames := []string{"test1", "test2"}
		for _, name := range refNames {
			r := ref.DeepCopy()
			r.SetName(name)
			r.SetUID(uuid.NewUUID())
			s.References[refKey] = append(s.References[refKey], r)
		}
	}

	return s
}

func newObject(gvk schema.GroupVersionKind, name string) *Unstructured {
	object := &Unstructured{}
	object.SetGroupVersionKind(gvk)
	object.SetNamespace("default")
	object.SetName(name)
	object.SetUID(uuid.NewUUID())

	SetNestedStringSlice(object.Object, []string{"c1", "c2"}, "spec", "configMapRefs")

	return object
}

func newPod(name string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      name,
			UID:       uuid.NewUUID(),
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				corev1.Container{
					Name:  "test",
					Image: "nginx:latest",
				},
			},
		},
	}
}

func newConfigMap(name string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      name,
			UID:       uuid.NewUUID(),
		},
	}
}

func newCRD(kind string) *apiextensionsv1beta1.CustomResourceDefinition {
	group := "example.com"
	version := "v1alpha1"
	plural := strings.ToLower(kind)
	name := fmt.Sprintf("%s.%s", plural, group)

	return &apiextensionsv1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
			Group:   group,
			Version: version,
			Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
				Kind:     kind,
				Plural:   plural,
				Singular: plural,
			},
			Scope: apiextensionsv1beta1.NamespaceScoped,
		},
	}
}
