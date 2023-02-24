module github.com/summerwind/whitebox-controller

go 1.13

require (
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/ghodss/yaml v1.0.0
	github.com/go-logr/logr v0.2.0
	github.com/go-logr/zapr v0.1.1 // indirect
	github.com/imdario/mergo v0.3.7 // indirect
	github.com/onsi/gomega v1.7.0
	github.com/prometheus/client_golang v0.9.3-0.20190127221311-3c4408c8b829 // indirect
	github.com/prometheus/procfs v0.0.0-20190315082738-e56f2e22fc76 // indirect
	k8s.io/api v0.20.0-alpha.2
	k8s.io/apiextensions-apiserver v0.0.0-20190918161926-8f644eb6e783
	k8s.io/apimachinery v0.20.0-alpha.2
	k8s.io/client-go v0.20.0-alpha.2
	sigs.k8s.io/controller-runtime v0.4.0
)
