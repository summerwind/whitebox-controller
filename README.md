# Whitebox Controller

Whitebox Controller is an extensible general purpose controller for Kubernetes.

This controller performs reconciliation or validation of Kubernetes resources by executing external commands or sending HTTP requests to external URLs. This allows developers to implement the Kubernetes controller simply by providing an external command or HTTP endpoint.

## Motivation

- Allow developers to make controllers without various knowledge of Kubernetes
- Allow developers to implement controllers in their familiar programming languages
- Enable quick validation of new controller ideas

## Documentation

- [Getting Started](docs/getting-started.md)
- [Configuration](docs/configuration.md)
- [Implementing controller](docs/implementing-controller.md)
