# containerset-controller

containerset-controller is an example controller that watches ContainerSet resource and creates Deployment resource.

This controller consists of three python scripts:

- `reconciler.py`: The script that reconcile resource state.
- `validator.py`: The script that validate created or updated resource via Admission Webhook.
- `mutator.py`: The script that defaulting a resource via Admission Webhook.

These scripts are assigned to the controller in the configuration file. See `config.yaml` for more details.

## Build

```
$ docker build -t containerset-controller:latest .
```

## Deploy

```
$ kubectl apply -f manifests/controller.yaml
```

## Test

Create a ContainerSet resource.

```
$ kubectl apply -f manifests/containerset.yaml
containerset.whitebox.summerwind.dev "containerset-example" created
```

Verify that the ContainerSet resource has been created.

```
$ kubectl get containerset
NAME      AGE
containerset-example   10s
```

The deployment resource also have been created by the containerset-controller.

```
$ kubectl get deployment containerset-example
NAME                   DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
containerset-example   2         2         2            2           1m
```

Invalid ContainerSet resource will be rejected by the admission webhook.

```
$ kubectl apply -f manifests/containerset-invalid.yaml
Error from server ('spec.image' must be specified): error when creating "manifests/containerset-invalid.yaml": admission webhook "containerset.whitebox.summerwind.github.io" denied the request: 'spec.image' must be specified
```

