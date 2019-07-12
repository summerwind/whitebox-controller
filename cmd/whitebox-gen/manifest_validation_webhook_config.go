package main

import (
	"bytes"
	"html/template"
	"strings"
)

func genValidationWebhookConfig(o *Option) (string, error) {
	funcMap := template.FuncMap{
		"toLower": strings.ToLower,
	}

	tmpl, err := template.New("").Funcs(funcMap).Parse(validationTemplate)
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

var validationTemplate = `
{{ $name := .Name -}}
{{ $namespace := .Namespace -}}
apiVersion: admissionregistration.k8s.io/v1beta1
kind: ValidatingWebhookConfiguration
metadata:
  name: {{ .Name }}
  annotations:
    certmanager.k8s.io/inject-ca-from: {{ .Namespace }}/{{ .Name }}
webhooks:
{{ range .Config.Resources -}}
{{ if .Validator -}}
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
      path: /{{ .Group }}/{{ .Version }}/{{ .Kind | toLower }}/validate
    caBundle: ""
{{ end -}}
{{ end -}}
`
