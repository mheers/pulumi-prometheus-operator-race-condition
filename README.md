# Pulumi-prometheus-operator-race-condition

> Showcases a race condition in [prometheus-operator](https://github.com/prometheus-operator/prometheus-operator) when using [Pulumi](https://www.pulumi.com/) to deploy it to a Kubernetes cluster. The race condition appears when trying to delete the CR.

## Steps to reproduce

1. create a new pulumi stack `pulumi stack init <stack-name>`

2. run `pulumi up` - this deploys the prometheus-operator and the crds to the cluster. It creates a namespace before and also deploys the custom resource for a prometheus instance.

3. comment out the following lines (ll. 51 - 53) from the `main.go`:

```golang
// if err := pm.installPrometheusInstance(); err != nil {
//     return err
// }
```

4. run `pulumi up` again - this should remove the prometheus instance and will get the operator stuck in a loop. The operator will try to delete the prometheus instance but will fail with the error from `logs/operator.log` as it tries to create a ConfigMap for a prometheus instance that does not exist anymore.

## Observations

- The `logs/k8s-api-audit.log` shows the pulumi k8s API requests. They are the same when deleting the CR using `kubectl` - but the problem does not exist when using `kubectl`. The only differences are the user-agent and that pulumi starts a watch on the CR a few milliseconds before the delete request.
