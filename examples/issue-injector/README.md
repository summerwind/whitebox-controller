# issue-injector

This is an example of a controller with injector.

This controller receives webhook request from GitHub and creates an Issue resource.

## Build

Build a container image.

```
$ docker build -t summerwind/issue-injector:latest .
```

## Deploy

Create controller resources that includes CRD and Dployment.

```
$ kubectl apply -f manifests/controller.yaml
```

## Test

Generate an injection token.

```
$ whitebox-gen token -name test -namespace default -signing-key injector/signing-key.pem
```

Register the following webhook URL with the generated injection token to your GitHub repository. Note that you need to set the issue of event to be received by webhook.

- `https://${SERVICE_URL}/whitebox.summerwind.dev/v1alpha1/issue/inject?token=${TOKEN}`

If you create an issue on GitHub repository, an Issue resource will be created in your Kubernetes.

```
$ kubectl get issue
NAME                     AGE
whitebox-controller-10   27m
```
