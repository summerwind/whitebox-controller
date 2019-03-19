package reconciler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
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
	config *Config
	api    kubernetes.Interface
	log    logr.Logger
}

func NewReconciler(config *Config, kconfig *rest.Config) (*Reconciler, error) {
	api, err := kubernetes.NewForConfig(kconfig)
	if err != nil {
		return nil, err
	}

	r := &Reconciler{
		config: config,
		api:    api,
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

	instance := &unstructured.Unstructured{}
	instance.SetGroupVersionKind(r.config.Resource)

	err := r.Get(context.TODO(), req.NamespacedName, instance)
	if err != nil {
		r.log.Error(err, "Failed to get a resource", "namespace", req.NamespacedName.Namespace, "name", req.NamespacedName.Name)
		return reconcile.Result{}, err
	}

	namespace := instance.GetNamespace()
	name := instance.GetName()

	status, ok := instance.Object["status"].(map[string]interface{})
	if ok {
		phase, _ = status["phase"].(string)
	}

	if phase == "" {
		phase = "new"
	}

	handler, ok := r.config.Handlers[phase]
	if !ok {
		r.log.Error(err, "No handler", "namespace", namespace, "name", name, "phase", phase)
		return reconcile.Result{}, nil
	}

	pod, err := newPod(instance, phase, handler)
	if err != nil {
		r.log.Error(err, "Invalid handler container", "namespace", namespace, "name", name, "phase", phase)
		return reconcile.Result{}, err
	}

	err = r.Create(context.TODO(), pod)
	if err != nil {
		r.log.Error(err, "Failed to create handler pod", "namespace", namespace, "name", name, "phase", phase)
		return reconcile.Result{}, err
	}

	r.log.Info("Handler pod started", "namespace", namespace, "name", name, "phase", phase, "pod", pod.Name)

	exitCode, err := r.waitPod(pod)
	if err != nil {
		r.log.Error(err, "Failed to wait for handler pod", "namespace", namespace, "name", name, "phase", phase)
		return reconcile.Result{}, err
	}

	r.log.Info("Handler pod completed", "namespace", namespace, "name", name, "phase", phase, "pod", pod.Name, "code", exitCode)

	return reconcile.Result{}, nil
}

func (r *Reconciler) waitPod(pod *corev1.Pod) (int32, error) {
	var exitCode int32

	watcher, err := r.api.CoreV1().Pods(pod.Namespace).Watch(metav1.ListOptions{
		LabelSelector: getLabelSelector(pod),
	})
	if err != nil {
		return exitCode, err
	}
	defer watcher.Stop()

	for ev := range watcher.ResultChan() {
		p, ok := ev.Object.(*corev1.Pod)
		if !ok {
			return exitCode, errors.New("unexpected resource")
		}

		phase := p.Status.Phase
		if phase == corev1.PodSucceeded || phase == corev1.PodFailed {
			for _, cs := range pod.Status.ContainerStatuses {
				if cs.Name != HandlerContainerName {
					continue
				}

				state := cs.LastTerminationState.Terminated
				if state == nil {
					return exitCode, errors.New("unexpected status")
				}

				exitCode = state.ExitCode
			}
			break
		}
	}

	return exitCode, nil
}

func newPod(instance *unstructured.Unstructured, phase string, handler Handler) (*corev1.Pod, error) {
	handlerID, err := getHandlerID(instance.GetName(), phase)
	if err != nil {
		return nil, err
	}

	container := handler.Container.DeepCopy()
	container.Name = HandlerContainerName

	gvk := instance.GroupVersionKind()

	envVars := []corev1.EnvVar{
		corev1.EnvVar{
			Name:  "HANDLER_RESOURCE_GROUP",
			Value: gvk.Group,
		},
		corev1.EnvVar{
			Name:  "HANDLER_RESOURCE_VERSION",
			Value: gvk.Version,
		},
		corev1.EnvVar{
			Name:  "HANDLER_RESOURCE_KIND",
			Value: gvk.Kind,
		},
		corev1.EnvVar{
			Name:  "HANDLER_RESOURCE_NAMESPACE",
			Value: instance.GetNamespace(),
		},
		corev1.EnvVar{
			Name:  "HANDLER_RESOURCE_NAME",
			Value: instance.GetName(),
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: instance.GetNamespace(),
			Name:      handlerID,
			Labels: map[string]string{
				LabelHandlerID: handlerID,
			},
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			Containers:    []corev1.Container{*container},
			Volumes: []corev1.Volume{
				corev1.Volume{
					Name: "workspace",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			},
		},
	}

	for i, _ := range pod.Spec.InitContainers {
		pod.Spec.InitContainers[i].Env = append(pod.Spec.InitContainers[i].Env, envVars...)
	}
	for i, _ := range pod.Spec.Containers {
		pod.Spec.Containers[i].Env = append(pod.Spec.Containers[i].Env, envVars...)
	}

	return pod, nil
}

func getHandlerID(name, phase string) (string, error) {
	b, err := ioutil.ReadAll(io.LimitReader(rand.Reader, 3))
	if err != nil {
		return "", err
	}

	gibberish := hex.EncodeToString(b)
	id := fmt.Sprintf("%s-%s-%s", name, phase, gibberish)

	return id, nil
}

func getLabelSelector(pod *corev1.Pod) string {
	return fmt.Sprintf("%s=%s", LabelHandlerID, pod.Name)
}
