# hello-controller (Exec handler)

This is an example of a controller that uses an exec handler.

This controller reconciles Hello resource. When create a Hello resource, this controller outputs the value of message field and update status of the resource.

## Build

Build a container image.

```
$ docker build -t summerwind/hello-controller:exec .
```

## Deploy

Create controller resources that includes CRD and Dployment.

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
[exec] stderr: 2019/07/13 11:20:01 message: Hello World
...
```
