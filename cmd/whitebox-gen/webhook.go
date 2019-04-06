package main

import (
	"bytes"
	"encoding/base64"
	"errors"
	"html/template"
	"strings"

	"github.com/summerwind/whitebox-controller/config"
)

const (
	KindValidatingWebhook = "ValidatingWebhookConfiguration"
	KindMutatingWebhook   = "MutatingWebhookConfiguration"

	PathValidatingWebhook = "validate"
	PathMutatingWebhook   = "mutate"
)

var webhookTemplate = `
apiVersion: admissionregistration.k8s.io/v1beta1
kind: {{.Kind}}
metadata:
  name: {{.ServiceName}}-{{.Plural}}
webhooks:
- name: {{.Plural}}.{{.Group}}
  rules:
  - apiGroups:
    - {{.Group}}
    apiVersions:
    - "*"
    resources:
    - {{.Plural}}
    operations:
    - CREATE
    - UPDATE
  failurePolicy: Fail
  clientConfig:
    service:
      name: {{.ServiceName}}
      namespace: {{.ServiceNamespace}}
      path: /{{.Plural}}.{{.Group}}/{{.Version}}/{{.ServicePath}}
    caBundle: {{.CABundle}}
`

type WebhookData struct {
	Kind             string
	Group            string
	Version          string
	Plural           string
	ServiceName      string
	ServiceNamespace string
	ServicePath      string
	CABundle         string
}

func genWebhookConfig(data WebhookData) (string, error) {
	tmpl, err := template.New("").Parse(webhookTemplate)
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

func webhook(c *config.WebhookConfig, name, namespace string, caBundle []byte) ([]string, error) {
	caBundleStr := base64.StdEncoding.EncodeToString(caBundle)

	manifests := []string{}
	vchecklist := map[string]bool{}
	mchecklist := map[string]bool{}

	for _, h := range c.Handlers {
		if h.Resource.Empty() {
			return nil, errors.New("invalid configuration: resource not found")
		}

		data := WebhookData{
			Group:            h.Resource.Group,
			Version:          h.Resource.Version,
			Plural:           strings.ToLower(h.Resource.Kind),
			ServiceName:      name,
			ServiceNamespace: namespace,
			CABundle:         caBundleStr,
		}

		_, vok := vchecklist[h.Resource.String()]
		_, mok := mchecklist[h.Resource.String()]

		if h.Validator != nil && !vok {
			data.Kind = KindValidatingWebhook
			data.ServicePath = PathValidatingWebhook

			m, err := genWebhookConfig(data)
			if err != nil {
				return nil, err
			}

			manifests = append(manifests, m)
			vchecklist[h.Resource.String()] = true
		}

		if h.Mutator != nil && !mok {
			data.Kind = KindMutatingWebhook
			data.ServicePath = PathMutatingWebhook

			m, err := genWebhookConfig(data)
			if err != nil {
				return nil, err
			}

			manifests = append(manifests, m)
			mchecklist[h.Resource.String()] = true
		}
	}

	return manifests, nil
}
