package main

import (
	"bytes"
	"html/template"
	"strings"
)

func genCRD(o *Option) ([]string, error) {
	crds := []string{}

	funcMap := template.FuncMap{
		"toLower": strings.ToLower,
	}

	tmpl, err := template.New("").Funcs(funcMap).Parse(crdTemplate)
	if err != nil {
		return crds, err
	}

	for _, res := range o.Config.Resources {
		buf := bytes.NewBuffer([]byte{})
		err = tmpl.Execute(buf, res)
		if err != nil {
			return crds, err
		}

		crds = append(crds, strings.Trim(buf.String(), "\n"))
	}

	return crds, nil
}

var crdTemplate = `
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: {{ .Kind | toLower }}.{{ .Group }}
spec:
  group: {{ .Group }}
  versions:
  - name: {{ .Version }}
    served: true
    storage: true
  names:
    kind: {{ .Kind }}
    plural: {{ .Kind | toLower }}
    singular: {{ .Kind | toLower }}
  scope: Namespaced
`
