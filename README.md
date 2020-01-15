# pkg

## Environement Variable

Unless otherwise noted, variables that are not relevant to the application use the prefix SYSTEM_.

Environment Variable | Description | Default
--------- | ----------- | -------
`SYSTEM_POD_NAME` | Pod Name | `""`
`SYSTEM_NAMESPACE` | Kubernetes Namespace used from `knative.dev/pkg` | `""`
`SYSTEM_PROFILE_PORT` | Profiling Server will be listening on this port. | `8018`
`SYSTEM_LOGGING_CONFIG_MAP_NAME` | The name of logging configmap. | `config-logging`
`SYSTEM_OBSERVABILITY_CONFIG_MAP_NAME` | The name of metric configmap. | `config-observability`
`SYSTEM_KUBERNETES_MIN_VERSION` | **REQUIRED**: Expected Kubernetes version. | `""`
