# k8s-collector
A Kubernetes Job to collect resources, logs and events from a Kubernetes cluster.
k8s-collector is a Job that can collect both resources (YAMLs) and logs when created.

`k8s/collector.yaml` contains the YAML to run it. Please be aware you will have to modify Job volume mounts.
It expects two argurments:

1. config-map => this is the name of the ConfigMap that contain the configuration on which logs/resources to collect. This README contains an example for such ConfigMap. ConfigMap must be in the same namespae of the Job.
2. dir => this is the directory when all collected resources and logs will be stored 

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: k8s-collector
  namespace: default
spec:
  template:
    spec:
      restartPolicy: Never
      serviceAccountName: k8s-collector
      containers:
      - name: k8s-collector
        image: projectsveltos/k8s-collector-amd64:main
        imagePullPolicy: IfNotPresent
        env:
          - name: COLLECTOR_NAMESPACE
            valueFrom:
              fieldRef:
                fieldPath: metadata.namespace
        command:
          - /k8s-collector
        args:
          - --config-map=k8s-collector
          - --dir=/collection
```

### ConfigMap example
Following is an example of ConfigMap containing the Collector configuration.
Configuration is asking for:

- v1 Secrets and appsv1 Deployments to be collected in all namespaces
- logs to be collected for all pods in the kube-system namespace. Only the last 600 seconds.

```yaml
apiVersion: v1
data:
  config.yaml: '{"resources":[{"group":"","version":"v1","kind":"Secret"},{"group":"apps","version":"v1","kind":"Deployment"}],"logs":[{"namespace":"kube-system","sinceSeconds":600}]}'
kind: ConfigMap
metadata:
  name: k8s-collector
  namespace: default
```

### Collection folders
k8s-collector will create two folders:

1. ```logs``` => this will contain collected logs
2. ```resources``` => this will contain collected resources

Each directory contains one subdirectory per namespace. Sticking with above example in the ```logs``` directory we have a ```kube-system``` subdirectory (since we asked k8s-collector to collect logs in that directory only).
Then within the ```kube-system``` sudirectory there is a log per pod/container pair.
For instance

```
coredns-5dd5756b68-bb5hr-coredns	    kube-apiserver-sveltos-management-control-plane-kube-apiserver
coredns-5dd5756b68-jjknb-coredns	    kube-controller-manager-sveltos-management-control-plane-kube-controller-manager
etcd-sveltos-management-control-plane-etcd  kube-proxy-qcvbf-kube-proxy
kindnet-bf559-kindnet-cni		    kube-proxy-trk6j-kube-proxy
kindnet-qzs4f-kindnet-cni		    kube-scheduler-sveltos-management-control-plane-kube-scheduler
```

The ```resource``` subdirectory contains one directory per namespace. And within each namespace directory, there is one directory per ```Kind```
For instance, we asked k8s-collector to collect __Secret__ and __Deployment__ from any namespace, so 

```resources/cert-manager/``` contains two subdirectories:
- Deployment => all collected Deployment instances in the cert-manager namespace will be here
- Secret => all collected Secret instance in the cert-manager namespace will be here

I developed to be used along with [Sveltos](https://github.com/projectsveltos) but it can be used on its own.