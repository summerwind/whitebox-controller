# Configuration

Whitebox Controller uses YAML format configuration file. By default Whitebox Controller will read the `config.yaml` in the current directory.

The configuration file consists of two parts: Resource configuration and Webhook configuration. The following sections explain these configurations in detail.

## Resource configuration

The `resources` key in the configuration file defines the settings for each resource.

For each single resource, Whitebox Controller prepares an internal controller and a webhook endpoint. The following is an example of a resource configuration.

```yaml
resources:
# This defines a resource named 'hello'.
- group: whitebox.summerwind.dev
  version: v1alpha1
  kind: Hello

  # Optional: Dependent resources owned by this resource.
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

  # Optional: A handler for Reconciler. This handler will be run
  # if there is a change in the resource.
  reconciler:
    exec:
      command: "/bin/controller"
      args: ["reconcile"]
    # Optional: Reconcile the resource again after the specified time.
    # The value must be the Go language's duration string.
    # See: https://golang.org/pkg/time/#ParseDuration
    requeueAfter: 60s
    # Optional: If you set this value to true, ignore the output of reconciler.
    # This setting is useful when you want to detect only changes without
    # managing the resource status.
    observe: false

  # Optional: A handler for Finalizer. This handler will be run
  # if the resource is going to be deleted.
  finalizer:
    exec:
      command: "/bin/controller"
      args: ["finalize"]

  # Optional: resyncPeriod is the period for reconciling all resources.
  # The value must be the Go language's duration string.
  # See: https://golang.org/pkg/time/#ParseDuration
  resyncPeriod: 30s

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

  # Optional: A handler for resource injection. This handler will be run
  # when the server received a request of injection webhook.
  injector:
    exec:
      command: "/bin/controller"
      args: ["inject"]
    # Required: Path of PEM encoded verification key file.
    verifyKeyFile: /etc/injector/verify.key
```

## Webhook configuration

The `webhook` key in the configuration file defines the settings for admission webhook server. The following is an example of a webhook configuration.

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
    certFile: /etc/tls/tls.crt
    keyFile: /etc/tls/tls.key
```

## Group/Version/Kind

Group/Version/Kind (GVK) are used in the following fields of configuration.

- `.resources[*]`
- `.resources[*].dependents`
- `.resources[*].references`

Group/Version/Kind is used to identify the type of Kubernetes resource. The meaning of each field is as follows.

```yaml
# Optional: API group for the resource.
# See: https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-groups
group: whitebox.summerwind.dev

# Required: API version for the resource.
# See: https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning
version: v1alpha1

# Required: Type of the resource.
kind: Test
```

## Handler configuration

Handler configuration are used in the following fields of configuration.

- `.resources[*].reconciler`
- `.resources[*].finalizer`
- `.resources[*].validator`
- `.resources[*].mutator`
- `.resources[*].injector`

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

