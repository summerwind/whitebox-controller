# Configuration

Whitebox Controller uses YAML format configuration file. By default Whitebox Controller will read the `config.yaml` in the current directory.

The configuration file consists of two parts: Controller configuration and Webhook configuration. The following sections explain these configurations in detail.

## Controller configuration

The `controllers` key in the configuration file defines the settings for each controller.

For a single controller, specify which resource to watch for changes, and which handler to run at the time of the change. The following is an example of a controller configuration.

```yaml
controllers:
# This defines a controller named 'hello'.
- name: hello

  # Required: A resource to be watched for changes.
  resource:
    group: whitebox.summerwind.github.io
    version: v1alpha1
    kind: Hello

  # Optional: Dependent resources owned by this controoler.
  # These resources are monitored for changes. If it detects a change,
  # the reconciler will be run.
  dependents:
  - group: "apps"
    version: v1
    kind: Deployment
    # Optional: If you set this value to true, reconciler will not set
    # the owner reference to the dependent resource.
    orphan: false

  # Optional: Resources referenced by a specified field of the resource.
  # The contents of the resources specified here are passed when the
  # reconciler is run, but they do not watched for changes.
  #
  # For `nameFieldPath`, specify the JSON path of the field name to refer
  # to another resource.
  references:
  - group: ""
    version: v1
    kind: ConfigMap
    nameFieldPath: ".spec.configMapRef.name"

  # Required: A handler for Reconciler. This handler will be run
  # if there is a change in the resource.
  reconciler:
    exec:
      command: "/bin/controller"
      args: ["reconcile"]
    # Optional: If you set this value to true, ignore the output of reconciler.
    # This setting is useful when you want to detect only changes without 
    # managing the resource status.
    observe: false

  # Optional: A handler for Finalizer. This handler will be run
  # if the resource is deleted.
  finalizer:
    exec:
      command: "/bin/controller"
      args: ["finalize"]

  # Optional: Syncer will run reconciler on all resources at the
  # specified interval.
  #
  # The value of `interval` must be the Go language's duration string.
  # See: https://golang.org/pkg/time/#ParseDuration
  syncer:
    interval: 30s
```

## Webhook configuration

The `webhook` key in the configuration file defines the settings for webhook server. This server provides admission webhooks for Kubernetes.

Webhook configuration contains Validation Webhooks and Mutation Webhooks for resources. The following is an example of a webhook configuration.

```yaml
webhook:
  # Optional: The IP address that the webhook server listen for.
  # If omitted, '0.0.0.0' will be used.
  host: 127.0.0.1

  # Optional: The port number that the webhook server listen for.
  # If omitted, a random port wlll be assigned.
  port: 443

  # Required: Path of certificate file and private key file for TLS.
  # Both of 'certFile' and 'keyFile' must be specified.
  tls:
    certFile: tls//server.pem
    keyFile: tls/server-key.pem

  # Required: Webhook handlers for resources.
  handlers:
  # Required: A resource to be validated of mutated.
  - resource:
      group: whitebox.summerwind.github.io
      version: v1alpha1
      kind: Hello

    # Optional: A handler for resource validation. This handler will be run
    # when the server received a request of validation webhook.
    validator:
      exec:
        command: "/bin/controller"
        args: ["validate"]

    # Optional: A handler for resource mutation. This handler will be run
    # when the server received a request of mutation webhook.
    mutator:
      exec:
        command: "/bin/controller"
        args: ["mutate"]
```

## Resource configuration

Resource configuration are used in the following fields of Controller and Webhook configuration.

- `.controllers[*].resource`
- `.controllers[*].dependents`
- `.controllers[*].references`
- `.webhook.handlers[*].resource`

Resource configuration consists of the following fields. These shows the types of resources in Kubernetes.

```
# Optional: API group for the resource.
# See: https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-groups
group: whitebox.summerwind.github.io

# Required: API version for the resource.
# See: https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning
version: v1alpha1

# Required: Type of the resource.
kind: Test
```

## Handler configuration

Handler configuration are used in the following fields of Controller and Webhook configuration.

- `.controllers[*].reconciler`
- `.controllers[*].finalizer`
- `.webhook.handlers[*].validator`
- `.webhook.handlers[*].mutator`

Handler type can be choosed from 'exec' or 'http'. 'exec' executes the specified command and uses its output. 'http' sends the request to the specified URL and uses the response.

Using both handler type at the same time is not allowed.

```yaml
exec:
  # Required: The path to command.
  command: "/bin/controller"

  # Optional: The arguments for the command.
  args: ["reconcile"]

  # Optional: The directory path where the command to be run.
  workingDir: /workspace

  # Optional: Environment variables.
  env:
    name: value

  # Optional: Execution timeout of the command. default is '60s'.
  #
  # This value of must be the Go language's duration string.
  # See: https://golang.org/pkg/time/#ParseDuration
  timeout: 30s

  # Optional: If you set this to true, stdin, stdout and stderr of the command will be logged.
  debug: false

http:
  # Required: The URL to be sent a request.
  url: http://127.0.0.1:3000/reconcile

  # Optional: TLS configuration for the specified URL.
  tls:
    # Optional: Path of certificate file and private key file for 
    # TLS client authentication. Both of 'certFile' and 'keyFile' 
    # must be specified.
    certFile: tls/server.pem
    keyFile: tls/server-key.pem

    # Optional: CA certificate file to be used for server certificate
    # validation.
    caCertFile: tls/ca.pem

  # Optional: Execution timeout of the command. default is '60s'.
  #
  # This value of must be the Go language's duration string.
  # See: https://golang.org/pkg/time/#ParseDuration
  timeout: 30s

  # Optional: If you set this to true, stdin, stdout and stderr of the command will be logged.
  debug: false
```

