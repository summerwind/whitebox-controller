package controller

import (
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/summerwind/whitebox-controller/config"
	"github.com/summerwind/whitebox-controller/reconciler"
	"github.com/summerwind/whitebox-controller/syncer"
)

func New(c *config.ControllerConfig, mgr manager.Manager) (*controller.Controller, error) {
	var (
		r   *reconciler.Reconciler
		err error
	)

	r, err = reconciler.New(c, mgr.GetEventRecorderFor(c.Name))
	if err != nil {
		return nil, fmt.Errorf("could not create reconciler: %v", err)
	}

	ctrl, err := controller.New(c.Name, mgr, controller.Options{Reconciler: r})
	if err != nil {
		return nil, fmt.Errorf("could not create controller: %v", err)
	}

	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(c.Resource)

	err = ctrl.Watch(&source.Kind{Type: obj}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return nil, fmt.Errorf("failed to watch resource: %v", err)
	}

	// No need to setup deps and syncer for observer.
	if r.IsObserver() {
		return &ctrl, nil
	}

	for _, dep := range c.Dependents {
		depObj := &unstructured.Unstructured{}
		depObj.SetGroupVersionKind(dep)

		err = ctrl.Watch(&source.Kind{Type: depObj}, &handler.EnqueueRequestForOwner{
			IsController: true,
			OwnerType:    obj,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to watch dependent resource: %v", err)
		}
	}

	if c.Syncer != nil {
		s, err := syncer.New(c, mgr)
		if err != nil {
			return nil, fmt.Errorf("could not create syncer: %v", err)
		}

		err = ctrl.Watch(&source.Channel{Source: s.C}, &handler.EnqueueRequestForObject{})
		if err != nil {
			return nil, fmt.Errorf("failed to watch sync channel: %v", err)
		}
	}

	return &ctrl, nil
}
