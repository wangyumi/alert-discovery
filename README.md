# Kubernetes Alert Discovery

Kubernetes container resource usage alerts auto generator

Fetch containers' settings from Kubernetes apiserver, calculate the thresholds and write them into configmaps back to Kubernetes in Prometheus Alertmanager pattern.

# Build and Run

## Build

```
make
```

## Run

running outside Kubernetes (Program will search for kubeconfig in ~/.kube)

```
./alert-discovery --running-in-cluster
```

running inside Kubernetes (Program will use Kubernetes serviceaccount)

```
./alert-discovery
```

## Flags

Name | Type | Description
--- | --- | --- 
configmap-name | string | configmap to put generated rules in
configmap-ns | string | namespace of configmap
critical-threshold | float | threshold of critical alert (default 0.95)
warning-threshold | float | threshold of warning alert (default 0.9)

## General Flags

Name | Description
--- | ---
running-in-cluster | Optional. If this controller is running in a kubernetes cluster, use the pod secrets for creating a Kubernetes client. (default true)
log.level | Only log messages with the given severity or above. Valid levels: [debug, info, warn, error, fatal]. (default info)

## Use Docker

You can deploy this exporter using the cargo.caicloud.io/sysinfra/alert-discovery Docker image.

For example:

```
docker pull cargo.caicloud.io/sysinfra/alert-discovery:v0.0

docker run -d -v ~/.kube/config:/root/.kube/config cargo.caicloud.io/sysinfra/event-exporter --running-in-cluster=false --configmap-name=example --configmap-ns=example
```

then check configmap content:

```
$ kubectl get configmaps example --namespace=example
```

example response:

```
  ...
  kubespace.kubespace-redis-3995357517-uqpsl.generated.rule: |2

    ALERT CPUUsageHigh
        IF sum(rate(container_cpu_usage_seconds_total{id="/docker/82066a6221511d60c3fc2a7cc50be265d48930825f4efe13418e59393980332e"}[5m])) > 0.450
        FOR 5m
        LABELS { severity = "warning", kubernetes_container_name = "redis", kubernetes_pod_name = "kubespace-redis-3995357517-uqpsl", kubernetes_namespace = "kubespace", id = "/docker/82066a6221511d60c3fc2a7cc50be265d48930825f4efe13418e59393980332e"}
        ANNOTATIONS {
            summary = "Container kubespace/kubespace-redis-3995357517-uqpsl/redis cpu usage high",
            description = "Container kubespace/kubespace-redis-3995357517-uqpsl/redis has high cpu usage of {{ $value }}",
        }

    ALERT CPUUsageHigh
        IF sum(rate(container_cpu_usage_seconds_total{id="/docker/82066a6221511d60c3fc2a7cc50be265d48930825f4efe13418e59393980332e"}[5m])) > 0.475
        FOR 5m
        LABELS { severity = "critical", kubernetes_container_name = "redis", kubernetes_pod_name = "kubespace-redis-3995357517-uqpsl", kubernetes_namespace = "kubespace", id = "/docker/82066a6221511d60c3fc2a7cc50be265d48930825f4efe13418e59393980332e"}
        ANNOTATIONS {
            summary = "Container kubespace/kubespace-redis-3995357517-uqpsl/redis cpu usage high",
            description = "Container kubespace/kubespace-redis-3995357517-uqpsl/redis has high cpu usage of {{ $value }}",
        }

    ALERT MemoryUsageHigh
        IF sum(container_memory_usage_bytes{id="/docker/82066a6221511d60c3fc2a7cc50be265d48930825f4efe13418e59393980332e"}) > 120795952
        FOR 5m
        LABELS { severity = "warning", kubernetes_container_name = "redis", kubernetes_pod_name = "kubespace-redis-3995357517-uqpsl", kubernetes_namespace = "kubespace", id = "/docker/82066a6221511d60c3fc2a7cc50be265d48930825f4efe13418e59393980332e"}
        ANNOTATIONS {
            summary = "Container kubespace/kubespace-redis-3995357517-uqpsl/redis memory usage high",
            description = "Container kubespace/kubespace-redis-3995357517-uqpsl/redis has high memory usage of {{ $value }}",
        }

    ALERT MemoryUsageHigh
        IF sum(container_memory_usage_bytes{id="/docker/82066a6221511d60c3fc2a7cc50be265d48930825f4efe13418e59393980332e"}) > 127506840
        FOR 5m
        LABELS { severity = "critical", kubernetes_container_name = "redis", kubernetes_pod_name = "kubespace-redis-3995357517-uqpsl", kubernetes_namespace = "kubespace", id = "/docker/82066a6221511d60c3fc2a7cc50be265d48930825f4efe13418e59393980332e"}
        ANNOTATIONS {
            summary = "Container kubespace/kubespace-redis-3995357517-uqpsl/redis memory usage high",
            description = "Container kubespace/kubespace-redis-3995357517-uqpsl/redis has high memory usage of {{ $value }}",
        }
  ...
```

# Note

Currently we take the `limit` settings of containers to generate alert rules. And use cAdvisor metrics of `container_memory_usage_bytes` and `container_cpu_usage_seconds_total` to write alert conditions. 
