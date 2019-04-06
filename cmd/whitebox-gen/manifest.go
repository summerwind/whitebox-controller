package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/summerwind/whitebox-controller/config"
)

func manifest(args []string) error {
	cmd := flag.NewFlagSet("manifest", flag.ExitOnError)
	configPath := cmd.String("c", "config.yaml", "Path to configuration file")
	name := cmd.String("service-name", "", "Service name of the webhook configuration")
	namespace := cmd.String("service-namespace", "", "Service namespace of the webhook configuration")
	caBundlePath := cmd.String("ca-bundle", "", "Path to CA bundle file")

	cmd.Parse(args)

	c, err := config.LoadFile(*configPath)
	if err != nil {
		return fmt.Errorf("could not load configuration file: %v", err)
	}

	manifests := []string{}

	crds, err := crd(c)
	if err != nil {
		return fmt.Errorf("failed to generate CRD: %v", err)
	}
	manifests = append(manifests, crds...)

	if c.Webhook != nil {
		if *name == "" {
			return errors.New("-service-name must be specified")
		}
		if *namespace == "" {
			return errors.New("-service-namespace must be specified")
		}
		if *caBundlePath == "" {
			return errors.New("-ca-bundle must be specified")
		}

		caBundle, err := ioutil.ReadFile(*caBundlePath)
		if err != nil {
			return fmt.Errorf("could not read CA bundle file: %v", err)
		}

		webhooks, err := webhook(c.Webhook, *name, *namespace, caBundle)
		if err != nil {
			return fmt.Errorf("failed to generate webhook config: %v", err)
		}
		manifests = append(manifests, webhooks...)
	}

	fmt.Println(strings.Join(manifests, "\n---\n"))

	return nil
}
