# noop-controller

noop-controller is an example implementation using whitebox-controller. "noop" means "No Operation".

## Build

The container image of noop-controller is built as follows.

```
$ docker build -t noop-controller:latest
```

## Deploy

Deploy to the Kubernetes cluster in the following order:

```
$ kubectl apply -f manifests/crd.yaml
$ kubectl apply -f manifests/secret.yaml
$ kubectl apply -f manifests/controller.yaml
$ kubectl apply -f manifests/webhook.yaml
```

## Verify

Create a resource and verify controller.

```
$ kubectl apply -f manifests/example.yaml
```

Creation of 'invalid' resource will fail by Controller's validator. This is intentional to verify the validation behavior.

```
noop.whitebox.summerwind.github.io "valid" created
Error from server ('spec.data' is empty): error when creating "manifests/example.yaml": admission webhook "noop.whitebox.summerwind.github.io" denied the request: 'spec.data' is empty
```
