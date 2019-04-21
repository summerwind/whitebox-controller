# Implementing controller

This document provides the information needed to implement a Controller.

## Overview

The following two tasks are required to implement Controller.

1. Configure the resource to be watched.
2. Configure the reconciler to handle the resource changed.

## Configuring Resource

*Resource* is a resource on Kubernetes for which you want to detect changes. Whitebox Controller executes reconciler when the resource is changed.

*Resource* is specified in `.controllers[].resource` of the configuration file. The following setting is an example of detecting changes of *ContainerSet* resources.

```
controllers:
- name: containerset-controller
  resource:
    group: whitebox.summerwind.github.io
    version: v1alpha1
    kind: ContainerSet
```

### Dependent Resources

If reconciler creates another type of resource based on the specified *Resource*, you need to specify the type of resource to be created as *Dependent Resources*.

*Dependentr Resources* are specified in `.controllers[].dependents` in the configuration file. Multiple types of dependent resource can be specified. The following setting is an example when Controller creates a *Deployment* resource based on *ContainerSet* resource.

```
controllers:
- name: containerset-controller
  resource:
    group: whitebox.summerwind.github.io
    version: v1alpha1
    kind: ContainerSet
  dependents:
  - group: apps
    version: v1
    kind: Deployment
```

### Reference Resources

If you want to refer to other related resources when processing the specified *Resource*, you need to specify the resource type as *Reference Resources*.

For example, if the *Resource*'s `.spec.config.configMapRef` value indicates a *ConfigMap* name, you may need to get the *ConfigMap* when the reconciler processes the *Resource*. In such a case, the content of ConfigMap is passed to Reconciler by specifying *ConfigMap* in *Reference Resources*.

```
controllers:
- name: containerset-controller
  resource:
    group: whitebox.summerwind.github.io
    version: v1alpha1
    kind: ContainerSet
  references:
  - group: ""
    version: v1
    kind: ConfigMap
    nameFieldPath: ".spec.config.configMapRef"
```

## Configuring Reconciler

*Reconciler* is responsible for processing the changed resources and generating the next state of the resource. *Reconciler* specifies either an *Exec Handler* that executes an command or an *HTTP Handler* that sends a request to an URL.

### Exec Handler

*Exec Handler* executes arbitrary command to process resources. The executed command should read the resource state from **stdin** and write the next state of the resource to **stdout**. If the process is successful, the exit code must be **0**. Otherwise, the exit code must be **nonzero**.

The following example uses *Exec Handler* to specify a shell script called `reconciler.sh`.

```
controllers:
- name: containerset-controller
  reconciler:
    exec:
      command: ./reconciler.sh
```

### HTTP Handler

*HTTP handler* calls an arbitrary URL to process the resource. The server of the called URL must read the state of the resource from the HTTP request body and write the next state of the resource to the response body. If the processing is successful, the status code must be **200**. Otherwise, the status code must be **other than 200**.

The following example uses *HTTP Handler* to specify the URL.

```
controllers:
- name: containerset-controller
  reconciler:
    http:
      url: "http://127.0.0.1/reconciler"
```

### Input and Output

Whitebox Controller inputs the changed resource as the following JSON format data into *Reconciler*, and expects the same format data to be output from *Reconciler*. Note that the value of `.events` is used only output.

| Key | Type | Description |
| --- | --- | --- |
| `.object`            | Object | JSON representation of the changed resource. |
| `.dependents`        | Object | Object containing dependent resource. Object key indicates resource type. |
| `.dependents[*]`     | Array  | Array containing dependent resources by type. |
| `.dependents[*][*]`  | Object | JSON representation of the dependent resource. |
| `.references`        | Object | Object containing reference resource. Object key indicates resource type. |
| `.references[*]`     | Array  | Array containing reference resources by type. |
| `.references[*][*]`  | Object | JSON representation of the reference resource. |
| `.events`            | Array  | Array containing the events for the resource. |
| `.events[*]`         | Object | An event for the resource. |
| `.events[*].type`    | String | Types of the event ("Normal" or "Warning") |
| `.events[*].reason`  | String | The reason this event is generated. It should be in UpperCamelCase format. |
| `.events[*].message` | String | The human readable message. |

The example of the data is as follows.

```
{
  "object": {
    "apiVersion": "whitebox.summerwind.github.io/v1alpha1",
    "kind": "ContainerSet",
    "metadata": {
      "name": "example",
      "namespace": "default",
    },
    ...
  },
  "dependents": {
    "deployment.v1.apps": [
      {
        "apiVersion": "apps/v1",
        "kind": "Deployment",
        "metadata": {
          "name": "example",
          "namespace": "default",
        },
        ...
      }
    ]
  },
  "references": {
    "configmap.v1": [
      {
        "apiVersion": "v1",
        "kind": "ConfigMap",
        "metadata": {
          "name": "example-config",
          "namespace": "default",
        },
        ...
      }
    ]
  },
  "events": [
    {"type": "Normal", "reason":"Created", "message":"Create deployment"}
  ]
}
```

### Observe mode

If you enable the `observe` option as follows, Whitebox Controller does not expect *Reconciler* to output the next state of the resource. This option is useful if you want to detect only changes in resources and execute processing.

```
controllers:
- name: containerset-controller
  reconciler:
    observe: true
    exec:
      command: ./observer.sh
```

### Syncer

If you need *Reconciler* to periodically check the state of all resources, specify an interval for *Syncer* as follows. In this example, *Reconciler* will be run every 10 minutes as if all resources have changed.

```
controllers:
- name: containerset-controller
  syncer:
    interval: 10m
```
