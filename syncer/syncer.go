package syncer

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"

	"github.com/summerwind/whitebox-controller/config"
)

type Syncer struct {
	client.Client
	C        chan event.GenericEvent
	config   *config.Config
	interval time.Duration
	log      logr.Logger
}

func New(c *config.Config, mgr manager.Manager) (*Syncer, error) {
	interval, err := time.ParseDuration(c.Syncer.Interval)
	if err != nil {
		return nil, err
	}

	s := &Syncer{
		Client:   mgr.GetClient(),
		C:        make(chan event.GenericEvent),
		config:   c,
		interval: interval,
		log:      logf.Log.WithName("syncer"),
	}

	return s, mgr.Add(s)
}

func (s *Syncer) Start(stop <-chan struct{}) error {
	t := time.NewTicker(s.interval)

	for {
		select {
		case <-t.C:
			err := s.Sync()
			if err != nil {
				s.log.Error(err, "Sync error")
			}
			s.log.Info("Synced")
		case <-stop:
			s.log.Info("Stopping syncer")
			return nil
		}
	}
}

func (s *Syncer) Sync() error {
	instanceList := &unstructured.UnstructuredList{}
	instanceList.SetGroupVersionKind(s.config.Resource)

	err := s.List(context.TODO(), &client.ListOptions{}, instanceList)
	if err != nil {
		return err
	}

	for _, instance := range instanceList.Items {
		s.C <- event.GenericEvent{
			Meta: &metav1.ObjectMeta{
				Name:      instance.GetName(),
				Namespace: instance.GetNamespace(),
			},
		}
	}

	return nil
}
