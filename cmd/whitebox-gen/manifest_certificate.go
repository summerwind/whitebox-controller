package main

import (
	"bytes"
	"strings"
	"text/template"
)

func genCertificate(o *Option) (string, error) {
	tmpl, err := template.New("").Parse(certificateTemplate)
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

var certificateTemplate = `
apiVersion: certmanager.k8s.io/v1alpha1
kind: Issuer
metadata:
  name: {{ .Name }}-selfsign
  namespace: {{ .Namespace }}
spec:
  selfSigned: {}
---
apiVersion: certmanager.k8s.io/v1alpha1
kind: Certificate
metadata:
  name: {{ .Name }}-webhook-ca
  namespace: {{ .Namespace }}
spec:
  secretName: {{ .Name }}-webhook-ca
  issuerRef:
    name: {{ .Name }}-selfsign
  commonName: "{{ .Name }} webhook CA"
  duration: 43800h
  isCA: true
---
apiVersion: certmanager.k8s.io/v1alpha1
kind: Issuer
metadata:
  name: {{ .Name }}-webhook-ca
  namespace: {{ .Namespace }}
spec:
  ca:
    secretName: {{ .Name }}-webhook-ca
---
apiVersion: certmanager.k8s.io/v1alpha1
kind: Certificate
metadata:
  name: {{ .Name }}
  namespace: {{ .Namespace }}
spec:
  secretName: {{ .Name }}
  issuerRef:
    name: {{ .Name }}-webhook-ca
  dnsNames:
  - {{ .Name }}
  - {{ .Name }}.{{ .Namespace }}
  - {{ .Name }}.{{ .Namespace }}.svc
  duration: 8760h
`
