package main

import (
	"flag"
	"io/ioutil"
	"os"

	"github.com/summerwind/whitebox-controller/reconciler"
	yaml "gopkg.in/yaml.v2"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
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

	buf, err := ioutil.ReadFile(*configPath)
	if err != nil {
		log.Error(err, "could not load configuration file")
		os.Exit(1)
	}

	kc, err := config.GetConfig()
	if err != nil {
		log.Error(err, "cloud not load kubernetes configuration")
		os.Exit(1)
	}

	mgr, err := manager.New(kc, manager.Options{})
	if err != nil {
		log.Error(err, "could not create manager")
		os.Exit(1)
	}

	c := &reconciler.Config{}
	err = yaml.Unmarshal(buf, c)
	if err != nil {
		log.Error(err, "failed to parse configuration file")
		os.Exit(1)
	}

	reconciler, err := reconciler.NewReconciler(c, kc)
	if err != nil {
		log.Error(err, "could not create reconciler")
		os.Exit(1)
	}

	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(c.Resource)

	err = builder.ControllerManagedBy(mgr).For(obj).Complete(reconciler)
	if err != nil {
		log.Error(err, "could not create controller")
		os.Exit(1)
	}

	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Error(err, "could not start manager")
		os.Exit(1)
	}
}
