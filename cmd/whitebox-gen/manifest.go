package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/summerwind/whitebox-controller/config"
)

type Option struct {
	Name      string
	Namespace string
	Image     string
	Config    *config.Config
}

func (o *Option) Validate() error {
	if o.Name == "" {
		return fmt.Errorf("name must be specified")
	}

	if o.Image == "" {
		return fmt.Errorf("image must be specified")
	}

	err := o.Config.Validate()
	if err != nil {
		return err
	}

	return nil
}

func manifest(args []string) error {
	cmd := flag.NewFlagSet("manifest", flag.ExitOnError)
	configPath := cmd.String("c", "config.yaml", "Path to configuration file")
	name := cmd.String("name", "", "Name of the controller")
	namespace := cmd.String("namespace", "default", "Namespace of the controller")
	image := cmd.String("image", "", "Image name of the controller")

	cmd.Parse(args)

	c, err := config.LoadFile(*configPath)
	if err != nil {
		return fmt.Errorf("could not load configuration file: %v", err)
	}

	o := &Option{
		Name:      *name,
		Namespace: *namespace,
		Image:     *image,
		Config:    c,
	}

	err = o.Validate()
	if err != nil {
		return err
	}

	manifests := []string{}

	crds, err := genCRD(o)
	if err != nil {
		return fmt.Errorf("failed to generate CRD: %v", err)
	}
	manifests = append(manifests, crds...)

	certs, err := genCertificate(o)
	if err != nil {
		return fmt.Errorf("failed to generate certificates: %v", err)
	}
	manifests = append(manifests, certs)

	controller, err := genController(o)
	if err != nil {
		return fmt.Errorf("failed to generate resources for controller: %v", err)
	}
	manifests = append(manifests, controller)

	vwc, err := genValidationWebhookConfig(o)
	if err != nil {
		return fmt.Errorf("failed to generate validation webhook config: %v", err)
	}
	manifests = append(manifests, vwc)

	mwc, err := genMutatingWebhookConfig(o)
	if err != nil {
		return fmt.Errorf("failed to generate mutating webhook config: %v", err)
	}
	manifests = append(manifests, mwc)

	fmt.Println(strings.Join(manifests, "\n---\n"))

	return nil
}
