package main

import (
	"bytes"
	"errors"
	"html/template"
	"strings"

	"github.com/summerwind/whitebox-controller/config"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var crdTemplate = `
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: {{.Plural}}.{{.Group}}
spec:
  group: {{.Group}}
  versions:
  - name: {{.Version}}
    served: true
    storage: true
  names:
    kind: {{.Kind}}
    plural: {{.Plural}}
    singular: {{.Singular}}
  scope: Namespaced
`

type CRDData struct {
	Group    string
	Version  string
	Kind     string
	Plural   string
	Singular string
}

func genCRD(res schema.GroupVersionKind) (string, error) {
	data := CRDData{
		Group:    res.Group,
		Version:  res.Version,
		Kind:     res.Kind,
		Plural:   strings.ToLower(res.Kind),
		Singular: strings.ToLower(res.Kind),
	}

	tmpl, err := template.New("").Parse(crdTemplate)
	if err != nil {
		return "", err
	}

	buf := bytes.NewBuffer([]byte{})
	err = tmpl.Execute(buf, data)
	if err != nil {
		return "", err
	}

	return strings.Trim(buf.String(), "\n"), nil
}

func crd(c *config.Config) ([]string, error) {
	crds := []string{}
	checklist := map[string]bool{}

	for _, ctrl := range c.Controllers {
		if ctrl.Resource.Empty() {
			return nil, errors.New("invalid configuration: resource not found")
		}

		_, ok := checklist[ctrl.Resource.String()]
		if ok {
			continue
		}

		m, err := genCRD(ctrl.Resource)
		if err != nil {
			return nil, err
		}

		crds = append(crds, m)
		checklist[ctrl.Resource.String()] = true
	}

	if c.Webhook != nil {
		for _, h := range c.Webhook.Handlers {
			if h.Resource.Empty() {
				return nil, errors.New("invalid configuration: resource not found")
			}

			_, ok := checklist[h.Resource.String()]
			if ok {
				continue
			}

			m, err := genCRD(h.Resource)
			if err != nil {
				return nil, err
			}

			crds = append(crds, m)
			checklist[h.Resource.String()] = true
		}
	}

	return crds, nil
}
