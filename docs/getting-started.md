# Getting Started

This guide shows you how to implement a controller using Whitebox Controller.

## Before you begin

- Make sure you have a Kubernetes cluster
- Install [jq](https://stedolan.github.io/jq/)

## What is Whitebox Controller?

Whitebox Controller is an extensible general purpose controller for Kuberentes. With Whitebox Controller, you can implement a Kubernetes controller simply by creating the command that you need to proccessing resource state, and the configuration file.

## Install

First install the `whitebox-controller` command. You can download the binary from [GitHub Release](https://github.com/summerwind/whitebox-controller/releases).

**For Linux**

```
$ curl -L -O https://github.com/summerwind/whitebox-controller/releases/latest/download/whitebox-controller-linux-amd64.tar.gz
$ tar zxvf whitebox-controller-linux-amd64.tar.gz
$ mv whitebox-controller /usr/local/bin/
```

**For macOS**

```
$ curl -L -O https://github.com/summerwind/whitebox-controller/releases/latest/download/whitebox-controller-darwin-amd64.tar.gz
$ tar zxvf whitebox-controller-darwin-amd64.tar.gz
$ mv whitebox-controller /usr/local/bin/
```

## Creating configuration file

To begin implementing a controller, create a configuration file first. The configuration file defines which resources you want to watch the changes and which commands you want to execute when changes are made.

Create a configuration file as follows. This file defines the configuration to watches the status of 'Hello' custom resource and to execute the command `reconciler.sh` when changes are made.

```
$ vim config.yaml
```
```
controllers:
- name: hello-controller
  resource:
    group: whitebox.summerwind.github.io
    version: v1alpha1
    kind: Hello
  reconciler:
    exec:
      command: ./reconciler.sh
      debug: true
```

## Creating your custom resource

Add 'Hello' custom resource on Kubernetes. To add a custom resource, you need to a manifest file that containes CustomResourceDefinition resource as follows.

```
$ vim crd.yaml
```
```
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: hello.whitebox.summerwind.github.io
spec:
  group: whitebox.summerwind.github.io
  versions:
  - name: v1alpha1
    served: true
    storage: true
  names:
    kind: Hello
    plural: hello
    singular: hello
  scope: Namespaced
```

Once you have a manifest file, apply it to Kubernetes.

```
$ kubectl apply -f crd.yaml
customresourcedefinition.apiextensions.k8s.io "hello.whitebox.summerwind.github.io" created
```

Now that 'Hello' custom resource is available. Let's create a 'Hello' resource on Kubernetes.

```
$ vim hello.yaml
```
```
apiVersion: whitebox.summerwind.github.io/v1alpha1
kind: Hello
metadata:
  name: hello
spec:
  message: "Hello World"
```
```
$ kubectl apply -f hello.yaml
hello.whitebox.summerwind.github.io "hello" created
```

You can see that the resource has been created on Kubernetes.

```
$ kubectl get hello
NAME      AGE
hello     10m
```

## Building your reconciler

Next, implement `reconciler.sh`, which is a command to be executed when the status of 'Hello' resource changes. **Reconciler** is a component in the Kubernetes controller that is responsible for coordinating the state between resources. Whitebox Controller can replace reconciler with any command.

Before implementing your reconciler, let's understand the inputs and outputs. Whitebox Controller executes a command when changes are made in the watched resource, and writes the changed resource information with JSON format as follows to the stdin of the command.

```
{
  "resource": {
    "apiVersion": "whitebox.summerwind.github.io/v1alpha1",
    "kind": "Hello",
    "metadata": {
      "annotations": {
        "kubectl.kubernetes.io/last-applied-configuration": "{\"apiVersion\":\"whitebox.summerwind.github.io/v1alpha1\",\"kind\":\"Hello\",\"metadata\":{\"annotations\":{},\"name\":\"hello\",\"namespace\":\"default\"},\"spec\":{\"message\":\"Hello World\"}}\n"
      },
      "creationTimestamp": "2019-04-01T04:02:01Z",
      "generation": 1,
      "name": "hello",
      "namespace": "default",
      "resourceVersion": "14412715",
      "selfLink": "/apis/whitebox.summerwind.github.io/v1alpha1/namespaces/default/hello/hello",
      "uid": "e6f446eb-5432-11e9-afad-42010a8c01f3"
    },
    "spec": {
      "message": "Hello World"
    }
  }
}
```

The command should read the resource state from stdin, modify its state if necessary, and write it out with JSON format to stdout at the end. The output here is applied to the resource on Kuberenetes by Whitebox Controller. For example, if you need to add the value `completed` to the `.resource.status.phase` field of state, command will write out the following state to stdout.

```
{
  "resource": {
    "apiVersion": "whitebox.summerwind.github.io/v1alpha1",
    "kind": "Hello",
    "metadata": {
      "annotations": {
        "kubectl.kubernetes.io/last-applied-configuration": "{\"apiVersion\":\"whitebox.summerwind.github.io/v1alpha1\",\"kind\":\"Hello\",\"metadata\":{\"annotations\":{},\"name\":\"hello\",\"namespace\":\"default\"},\"spec\":{\"message\":\"Hello World\"}}\n"
      },
      "creationTimestamp": "2019-04-01T04:02:01Z",
      "generation": 1,
      "name": "hello",
      "namespace": "default",
      "resourceVersion": "14412715",
      "selfLink": "/apis/whitebox.summerwind.github.io/v1alpha1/namespaces/default/hello/hello",
      "uid": "e6f446eb-5432-11e9-afad-42010a8c01f3"
    },
    "spec": {
      "message": "Hello World"
    },
    "status": {
      "phase": "completed"
    }
  }
}
```

Now that you understand the input and output, let's implement `reconciler.sh`. Implement a reconcier that outputs the value of the `.spec.message` field to stderr and sets the valud `completed` to the `.status.phase` field.

```
$ vim reconciler.sh
```
```
#!/bin/bash

# Read current state from stdio.
STATE=`cat -`

# Write message to stder
echo "${STATE}" | jq -r '.resource.spec.message' >&2

# Set `.status.phase` field to the resource
NEW_STATE=`echo "${STATE}" | jq -r '.resource.status.phase = "completed"'`

# Write new state to stdio.
echo "${NEW_STATE}"
```

Do not forget to give execute permission to `reconciler.sh`.

```
$ chmod +x reconciler.sh
```

## Testing your controller

Now that the reconciler has been implemented, run the `whitebox-controller` command to verify that your controller works properly.

```
$ whitebox-controller
{"level":"info","ts":1554099388.813269,"logger":"controller-runtime.controller","msg":"Starting EventSource","controller":"hello-controller","source":"kind source: whitebox.summerwind.github.io/v1alpha1, Kind=Hello"}
{"level":"info","ts":1554099388.915076,"logger":"controller-runtime.controller","msg":"Starting Controller","controller":"hello-controller"}
{"level":"info","ts":1554099389.02052,"logger":"controller-runtime.controller","msg":"Starting workers","controller":"hello-controller","worker count":1}
[exec] stdin: {"resource":{"apiVersion":"whitebox.summerwind.github.io/v1alpha1","kind":"Hello","metadata":{"annotations":{"kubectl.kubernetes.io/last-applied-configuration":"{\"apiVersion\":\"whitebox.summerwind.github.io/v1alpha1\",\"kind\":\"Hello\",\"metadata\":{\"annotations\":{},\"name\":\"hello\",\"namespace\":\"default\"},\"spec\":{\"message\":\"Hello World\"}}\n"},"creationTimestamp":"2019-04-01T05:58:29Z","generation":1,"name":"hello","namespace":"default","resourceVersion":"14427301","selfLink":"/apis/whitebox.summerwind.github.io/v1alpha1/namespaces/default/hello/hello","uid":"2c2673ab-5443-11e9-afad-42010a8c01f3"},"spec":{"message":"Hello World"},"status":{"phase":"completed"}},"dependents":[],"references":[],"events":[]}
[exec] stderr: Hello World
[exec] stdout: {
  "resource": {
    "apiVersion": "whitebox.summerwind.github.io/v1alpha1",
    "kind": "Hello",
    "metadata": {
      "annotations": {
        "kubectl.kubernetes.io/last-applied-configuration": "{\"apiVersion\":\"whitebox.summerwind.github.io/v1alpha1\",\"kind\":\"Hello\",\"metadata\":{\"annotations\":{},\"name\":\"hello\",\"namespace\":\"default\"},\"spec\":{\"message\":\"Hello World\"}}\n"
      },
      "creationTimestamp": "2019-04-01T05:58:29Z",
      "generation": 1,
      "name": "hello",
      "namespace": "default",
      "resourceVersion": "14427301",
      "selfLink": "/apis/whitebox.summerwind.github.io/v1alpha1/namespaces/default/hello/hello",
      "uid": "2c2673ab-5443-11e9-afad-42010a8c01f3"
    },
    "spec": {
      "message": "Hello World"
    },
    "status": {
      "phase": "completed"
    }
  },
  "dependents": [],
  "references": [],
  "events": []
}
```

Logs with the `[exec]` prefix are for debugging. These logs are come from stdin, stdout, and stderr of the reconciler command. If you look at the log starting with `[exec] stdout:`, you can see that the `.resource.stat us.phase` field has the value `completed` as intended.

You can also see that the value of the `.resource.stat us.phase` field in Kubernetes has been changed.

```
$ kubectl describe hello hello
Name:         hello
Namespace:    default
Labels:       <none>
Annotations:  kubectl.kubernetes.io/last-applied-configuration={"apiVersion":"whitebox.summerwind.github.io/v1alpha1","kind":"Hello","metadata":{"annotations":{},"name":"hello","namespace":"default"},"spec":{"messa...
API Version:  whitebox.summerwind.github.io/v1alpha1
Kind:         Hello
Metadata:
  Creation Timestamp:  2019-04-01T05:58:29Z
  Generation:          1
  Resource Version:    14427301
  Self Link:           /apis/whitebox.summerwind.github.io/v1alpha1/namespaces/default/hello/hello
  UID:                 2c2673ab-5443-11e9-afad-42010a8c01f33
Spec:
  Message:  Hello World
Status:
  Phase:  completed
Events:   <none>
```

Now we have confirmed that the controller works!

## Building container image

Build a container image to deploy your controller on Kubernetes. First, create a `Dockerfile` that defines the contents of the container image.

```
$ vim Dockerfile
```
```
FROM summerwind/whitebox-controller:latest AS base

#######################################

FROM ubuntu:18.04

RUN apt update \
  && apt install -y jq \
  && rm -rf /var/lib/apt/lists/\*

COPY --from=base /bin/whitebox-controller /bin/whitebox-controller

COPY reconciler.sh /reconciler.sh
COPY config.yaml   /config.yaml

ENTRYPOINT ["/bin/whitebox-controller"]
```

Build the container image using the `docker` command. Specify an arbitrary name for your container registry to `$ {NAME}`.

```
$ docker build -t ${NAME}/hello-controller:latest .
```

The built container image is pushed to the container registry.

```
$ docker push ${NAME}/hello-controller:latest
```

## Deploying your controller to Kubernetes

Finally, let's deploy the controller to Kubernetes. This example deploys a controller to `kube-system` namespace. Create a manifest file as follows.

```
$ vim controller.yaml
```
```
apiVersion: v1
kind: ServiceAccount
metadata:
  name: hello-controller
  namespace: kube-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: hello-controller
rules:
- apiGroups:
  - whitebox.summerwind.github.io
  resources:
  - hello
  verbs:
  - get
  - list
  - watch
  - create
  - update
  - patch
  - delete
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: hello-controller
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: hello-controller
subjects:
- kind: ServiceAccount
  name: hello-controller
  namespace: kube-system
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: hello-controller
  namespace: kube-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app: hello-controller
  template:
    metadata:
      labels:
        app: hello-controller
    spec:
      containers:
      - name: hello-controller
        image: ${NAME}/hello-controller:latest  # You need to set ${NAME} !
        imagePullPolicy: Always
        resources:
          requests:
            cpu: 100m
            memory: 20Mi
      serviceAccountName: hello-controller
```

Apply the manifest file to Kubernetes and make sure that the Controller is now running.

```
$ kubectl apply -f controller.yaml
$ kubectl get -n kube-system pod
NAME                               READY     STATUS    RESTARTS   AGE
...
hello-controller-54d9456cb4-v5swt  1/1       Running   0          10s
...
```
