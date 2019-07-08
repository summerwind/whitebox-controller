package state

import (
	"reflect"
	"testing"

	. "github.com/onsi/gomega"
	. "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestCopy(t *testing.T) {
	RegisterTestingT(t)

	s := &State{
		Object: newObject("Resource", "test"),
		Dependents: map[string][]*Unstructured{
			"a.v1alpha1.example.com": []*Unstructured{
				newObject("A", "a1"),
				newObject("A", "a2"),
			},
			"b.v1alpha1.example.com": []*Unstructured{
				newObject("B", "b1"),
				newObject("B", "b2"),
			},
		},
		References: map[string][]*Unstructured{
			"c.v1alpha1.example.com": []*Unstructured{
				newObject("C", "c1"),
				newObject("C", "c2"),
			},
			"d.v1alpha1.example.com": []*Unstructured{
				newObject("D", "d1"),
				newObject("D", "d2"),
			},
		},
	}

	ns := s.Copy()
	Expect(reflect.DeepEqual(*s, *ns)).To(BeTrue())
}

func TestDiff(t *testing.T) {
	RegisterTestingT(t)

	s := &State{
		Object: newObject("Resource", "test"),
		Dependents: map[string][]*Unstructured{
			"a.v1alpha1.example.com": []*Unstructured{
				newObject("A", "a1"),
				newObject("A", "a2"),
			},
			"b.v1alpha1.example.com": []*Unstructured{
				newObject("B", "b1"),
				newObject("B", "b2"),
			},
		},
	}

	ns := &State{
		Object: newObject("Resource", "test"),
		Dependents: map[string][]*Unstructured{
			"a.v1alpha1.example.com": []*Unstructured{
				newObject("A", "a1"),
				newObject("A", "a3"),
			},
			"b.v1alpha1.example.com": []*Unstructured{
				newObject("B", "b1"),
				newObject("B", "b3"),
			},
		},
	}

	SetNestedField(ns.Object.Object, "bye", "spec", "message")
	SetNestedField(ns.Dependents["a.v1alpha1.example.com"][0].Object, "bye", "spec", "message")
	SetNestedField(ns.Dependents["b.v1alpha1.example.com"][0].Object, "bye", "spec", "message")

	created, updated, deleted := s.Diff(ns)

	Expect(len(created)).To(Equal(2))
	Expect(created[0].GetName()).To(Equal("a3"))
	Expect(created[1].GetName()).To(Equal("b3"))

	Expect(len(updated)).To(Equal(3))
	Expect(updated[0].GetName()).To(Equal("test"))
	Expect(updated[1].GetName()).To(Equal("a1"))
	Expect(updated[2].GetName()).To(Equal("b1"))

	Expect(len(deleted)).To(Equal(2))
	Expect(deleted[0].GetName()).To(Equal("a2"))
	Expect(deleted[1].GetName()).To(Equal("b2"))
}

func TestPack(t *testing.T) {
	RegisterTestingT(t)

	ts := testState{
		Object: newResource("Resource", "test"),
		Dependents: map[string][]*resource{
			"a.v1alpha1.example.com": []*resource{
				newResource("A", "test1"),
				newResource("A", "test2"),
			},
		},
		References: map[string][]*resource{
			"b.v1alpha1.example.com": []*resource{
				newResource("B", "test1"),
				newResource("B", "test2"),
			},
		},
	}

	s := &State{}
	err := Pack(ts, s)
	Expect(err).NotTo(HaveOccurred())
	Expect(s.Object.GetName()).To(Equal("test"))

	deps := s.Dependents["a.v1alpha1.example.com"]
	Expect(deps[0].GetName()).To(Equal("test1"))
	Expect(deps[1].GetName()).To(Equal("test2"))

	refs := s.References["b.v1alpha1.example.com"]
	Expect(refs[0].GetName()).To(Equal("test1"))
	Expect(refs[1].GetName()).To(Equal("test2"))
}

func TestUnpack(t *testing.T) {
	RegisterTestingT(t)

	s := &State{
		Object: newObject("Resource", "test"),
		Dependents: map[string][]*Unstructured{
			"a.v1alpha1.example.com": []*Unstructured{
				newObject("A", "test1"),
				newObject("A", "test2"),
			},
		},
		References: map[string][]*Unstructured{
			"b.v1alpha1.example.com": []*Unstructured{
				newObject("B", "test1"),
				newObject("B", "test2"),
			},
		},
	}

	ts := &testState{}
	err := Unpack(s, ts)
	Expect(err).NotTo(HaveOccurred())
	Expect(ts.Object.Name).To(Equal("test"))

	deps := ts.Dependents["a.v1alpha1.example.com"]
	Expect(deps[0].Name).To(Equal("test1"))
	Expect(deps[1].Name).To(Equal("test2"))

	refs := ts.References["b.v1alpha1.example.com"]
	Expect(refs[0].Name).To(Equal("test1"))
	Expect(refs[1].Name).To(Equal("test2"))
}

type resource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec resourceSpec `json:"spec,omitempty"`
}

type resourceSpec struct {
	Message string `json:"message"`
}

type testState struct {
	Object     *resource              `json:"object"`
	Dependents map[string][]*resource `json:"dependents"`
	References map[string][]*resource `json:"references"`
}

func newResource(kind, name string) *resource {
	r := &resource{
		Spec: resourceSpec{
			Message: "hello",
		},
	}
	r.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "example.com",
		Version: "v1alpha1",
		Kind:    kind,
	})
	r.SetNamespace("default")
	r.SetName(name)

	return r
}

func newObject(kind, name string) *unstructured.Unstructured {
	object := &Unstructured{}
	object.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "example.com",
		Version: "v1alpha1",
		Kind:    kind,
	})
	object.SetNamespace("default")
	object.SetName(name)

	SetNestedField(object.Object, "hello", "spec", "message")

	return object
}
