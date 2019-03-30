# noop-controller

noop-controller is an controller implementation using whitebox-controller.

This controller watches the Noop resource but basically does nothing. So "noop" means "No Operation".

This controller consists of multiple python scripts:

- `reconciler.py`: This script is executed inside the Reconcilation Loop to reconcile resource status.
- `validator.py`: This validates created or updated resource via Admission Webhook.
- `mutator.py`: This modifies a resource via Admission Webhook.

These scripts are assigned to Controller functions in the configuration file. See `config.yaml` for details.

## Build

The container image of noop-controller is built as follows.

```
$ docker build -t noop-controller:latest
```

## Deploy

Deploy to the Kubernetes cluster in the following order:

```
# Install CRD for Noop resource.
$ kubectl apply -f manifests/crd.yaml

# Install webhook configurations for Admission Webhook.
$ kubectl apply -f manifests/webhook.yaml

# Install secrets for noop-controller.
$ kubectl apply -f manifests/secret.yaml

# Install noop-controller.
$ kubectl apply -f manifests/controller.yaml
```

## Test

Create a valid 'Noop' resource.

```
$ kubectl apply -f manifests/noop-valid.yaml
```

Then, resources have `noop` annotation by `mutator.py` and `phase` field by `reconciler.py`.

```
$ kubectl get noop valid -o yaml
apiVersion: whitebox.summerwind.github.io/v1alpha1
kind: Noop
metadata:
  annotations:
    noop: "true"
  creationTimestamp: 2019-03-30T02:17:45Z
  generation: 1
  name: valid
  namespace: default
  resourceVersion: "14084247"
  selfLink: /apis/whitebox.summerwind.github.io/v1alpha1/namespaces/default/noop/valid
  uid: 0141c7de-5292-11e9-afad-42010a8c01f3
spec:
  data: hello
status:
  phase: completed
```

Creation of a invalid 'Noop' resource will fail by `validator.py`.

```
$ kubectl apply -f manifests/noop-invalid.yaml
Error from server ('spec.data' is empty): error when creating "manifests/noop-invalid.yaml": admission webhook "noop.whitebox.summerwind.github.io" denied the request: 'spec.data' is empty
```

