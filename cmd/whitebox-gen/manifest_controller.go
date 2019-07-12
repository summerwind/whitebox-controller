package main

import (
	"bytes"
	"strings"
	"text/template"
)

func genController(o *Option) (string, error) {
	funcMap := template.FuncMap{
		"toLower": strings.ToLower,
	}

	tmpl, err := template.New("").Funcs(funcMap).Parse(controllerTemplate)
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

var controllerTemplate = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: {{ .Name }}
  namespace: {{ .Namespace }}
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: {{ .Name }}
rules:
{{ range .Config.Resources -}}
- apiGroups:
  - {{ .Group }}
  resources:
  - {{ .Kind | toLower }}
  verbs:
  - get
  - list
  - watch
  - create
  - update
  - patch
  - delete
{{ range .Dependents -}}
- apiGroups:
  - {{ .Group }}
  resources:
  - {{ .Kind | toLower }}
  verbs:
  - get
  - list
  - watch
  - create
  - update
  - patch
  - delete
{{ end -}}
{{ range .References -}}
- apiGroups:
  - {{ .Group }}
  resources:
  - {{ .Kind | toLower }}
  verbs:
  - get
  - list
  - watch
{{ end -}}
{{ end -}}
- apiGroups:
  - ""
  resources:
  - events
  verbs:
  - create
  - patch
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: {{ .Name }}
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: {{ .Name }}
subjects:
- kind: ServiceAccount
  name: {{ .Name }}
  namespace: {{ .Namespace }}
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .Name }}
  namespace: {{ .Namespace }}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: {{ .Name }}
  template:
    metadata:
      labels:
        app: {{ .Name }}
    spec:
      containers:
      - name: {{ .Name }}
        image: {{ .Image }}
        imagePullPolicy: IfNotPresent
        volumeMounts:
        - name: certificates
          mountPath: /etc/tls
        ports:
        - containerPort: 443
      volumes:
      - name: certificates
        secret:
          secretName: {{ .Name }}
      serviceAccountName: {{ .Name }}
      terminationGracePeriodSeconds: 60
---
apiVersion: v1
kind: Service
metadata:
  name: {{ .Name }}
  namespace: {{ .Namespace }}
spec:
  selector:
    app: {{ .Name }}
  ports:
  - protocol: TCP
    port: 443
    targetPort: 443
`
