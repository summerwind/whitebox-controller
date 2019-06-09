# hello-controller (Golang implementation)

This is an golang implementation of hello-controller. This controller demonstrates how to use whitebox-controller as golang library.

## Build

```
$ go build -o hello-controller .
```

## Deploy

Install CRD for Hello resource.

```
$ kubectl apply -f manifests/crd.yaml
```

Start hello-controller locally.

```
$ ./hello-controller
```

## Test

Create a Hello resource.

```
$ kubectl apply -f hello-controller/manifests/hello.yaml
hello.whitebox.summerwind.github.io/hello created
```

Verify that the Hello resource has been created.

```
$ kubectl get hello
NAME    AGE
hello   10s
```

hello-controller outputs the following log:

```
2019/06/09 15:09:08 message: Hello World
```
