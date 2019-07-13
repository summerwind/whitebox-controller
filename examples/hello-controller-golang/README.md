# hello-controller (Golang implementation)

This is an golang implementation of hello-controller intended to demonstrate how to use whitebox-controller as golang library.

This controller reconciles Hello resource. When create a Hello resource, this controller outputs the value message field and update status of the resource.

## Build

Build a binary and a container image.

```
$ CGO_ENABLED=0 GOOS=linux go build -o hello-controller .
$ docker build -t summerwind/hello-controller:golang .
```

## Deploy

Create controller resources that includes CRD, WebhookConfiguration, and Dployments.

```
$ kubectl apply -f manifests/controller.yaml
```

## Test

Create a Hello resource.

```
$ kubectl apply -f manifests/hello.yaml
hello.whitebox.summerwind.dev/hello created
```

Verify that the Hello resource has been created.

```
$ kubectl get hello
NAME    AGE
hello   10s
```

hello-controller outputs the following log:

```
$ kubectl logs -n kube-system hello-controller-b85467859-fk8s5
...
2019/07/13 11:09:08 message: Hello World
...
```
