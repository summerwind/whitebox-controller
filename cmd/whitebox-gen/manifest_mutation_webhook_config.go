package main

import (
	"bytes"
	"html/template"
	"strings"
)

func genMutatingWebhookConfig(o *Option) (string, error) {
	funcMap := template.FuncMap{
		"toLower": strings.ToLower,
	}

	tmpl, err := template.New("").Funcs(funcMap).Parse(mutatingTemplate)
	if err != nil {
		return "", err
	}

	buf := bytes.NewBuffer([]byte{})
	err = tmpl.Execute(buf, o)
	if err != nil {
		return "", err
	}

	return strings.Trim(buf.String(), "\n"), nil
}

var mutatingTemplate = `
{{ $name := .Name -}}
{{ $namespace := .Namespace -}}
apiVersion: admissionregistration.k8s.io/v1beta1
kind: MutatingWebhookConfiguration
metadata:
  name: {{ .Name }}
  annotations:
    certmanager.k8s.io/inject-ca-from: {{ .Namespace }}/{{ .Name }}
webhooks:
{{ range .Config.Resources -}}
{{ if .Mutator -}}
- name: {{ .Kind | toLower }}.{{ .Group }}
  rules:
  - apiGroups:
    - {{ .Group }}
    apiVersions:
    - {{ .Version }}
    resources:
    - {{ .Kind | toLower }}
    operations:
    - CREATE
    - UPDATE
  failurePolicy: Fail
  clientConfig:
    service:
      name: {{ $name }}
      namespace: {{ $namespace }}
      path: /{{ .Group }}/{{ .Version }}/{{ .Kind | toLower }}/mutate
    caBundle: ""
{{ end -}}
{{ end -}}
`
