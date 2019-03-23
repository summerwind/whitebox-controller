package main

import (
	"flag"
	"os"

	"github.com/summerwind/whitebox-controller/config"
	"github.com/summerwind/whitebox-controller/reconciler"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	kconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"

	_ "k8s.io/client-go/plugin/pkg/client/auth/azure"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
)

func main() {
	logf.SetLogger(logf.ZapLogger(false))
	log := logf.Log.WithName("whitebox-controller")

	var configPath = flag.String("c", "config.yaml", "Path to configuration file")

	flag.Parse()

	c, err := config.LoadFile(*configPath)
	if err != nil {
		log.Error(err, "could not load configuration file")
		os.Exit(1)
	}

	kc, err := kconfig.GetConfig()
	if err != nil {
		log.Error(err, "could not load kubernetes configuration")
		os.Exit(1)
	}

	mgr, err := manager.New(kc, manager.Options{})
	if err != nil {
		log.Error(err, "could not create manager")
		os.Exit(1)
	}

	reconciler, err := reconciler.NewReconciler(c)
	if err != nil {
		log.Error(err, "could not create reconciler")
		os.Exit(1)
	}

	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(c.Resource)

	ctrl, err := controller.New("controller", mgr, controller.Options{Reconciler: reconciler})
	if err != nil {
		log.Error(err, "could not create controller")
		os.Exit(1)
	}

	err = ctrl.Watch(&source.Kind{Type: obj}, &handler.EnqueueRequestForObject{})
	if err != nil {
		log.Error(err, "failed to watch resource")
		os.Exit(1)
	}

	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Error(err, "could not start manager")
		os.Exit(1)
	}
}
