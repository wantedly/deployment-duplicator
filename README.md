# deployment-duplicator

:warning: This project is currently experimental, expect API breaking changes until the first version.

Deployment Duplicator provides a Custom Resource and Controller to duplicate Deployments.


## Use cases

Deployment duplication is useful for testing, including canary testing in production and debug in development.

### Canary Release

The Canary Release is an approach to release a new application more safely. In short, the Canary Release consists of the following steps:

1. Deploy a new application. (called "Canary server")
1. Send some of the traffics to the Canary server. Typically we'd join a Canary server for the Load Balancer and Load Balancer routes traffic using the DNS round robin. In Kubernetes, Service assumes this role.
1. Evaluate a Canary server by comparing with other server deployed old application (called Baseline server) using some metrics, for example, RPS, resource usage, and error rate.
1. Roll out the new application to all servers OR rollback to the old version.

Deployment Duplicator supports the second phase, managing a Canary server and routing.
It creates a Deployment of a Canary server based on the existing ones.
Additionally, Deployment Duplicator can put some metadata to a generated Deployment for identifying whether an application is a Canary.

* Attach `canary: true` to Deployment's labels
* Give a name the host of Pod to `canary`
* Add `CANARY_ENABLED: 1` to environment variables of all containers

### `kubefork`

At Wantedly, we also use this project as a component of the `kubefork` tool:
it duplicates deployments selectively to be able to develop a service in an environment close to QA without copying the whole cluster.

Read more about it in the introduction article: [マイクロサービスでもポチポチ確認するための Kubefork](https://www.wantedly.com/companies/wantedly/post_articles/313884) (in Japanese)

## Installation

This installation depends on [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) and [kustomize](https://github.com/kubernetes-sigs/kustomize/blob/master/docs/INSTALL.md).
If your environment is OSX then you can install these using the following commands:

```
$ brew install kubernetes-cli
$ brew install kustomize
```

You can deploy to your cluster using `make` command. 

```
$ make deploy
```

Canary Controller is installed in `deployment-duplicator-system` namespace.

## Usage

Apply a very simple manifest if you want to deploy a Canary server: 

```yaml
apiVersion: duplication.k8s.wantedly.com/v1beta1
kind: DeploymentCopy
metadata:
  name: canary
spec:
  targetDeploymentName: foo
  customLabels:
    canary: "true"
  nameSuffix: "canary"
  hostname: "hostname"
  targetContainers:
    - name: nginx
      image: nginx:latest
      env:
      - name: CANARY_ENABLED
        value: "1"
    - name: redis
      image: redis:5.0.5
      env:
      - name: CANARY_ENABLED
        value: "1"
```

Deployment Duplicator looks up existing Deployment named "foo" in the "default" namespace and creates a new Deployment based on "foo" and overrides the image of container named "nginx" to "nginx:latest":

```console
$ kubectl get deploy -n default
NAME      DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
foo       1         1         1            1           1m

$ kubectl get deploy foo -o json | jq '.spec.template.spec.containers[] | if .name == "nginx" then .image else empty end'
"nginx:1.15.4"

$ kubectl apply -f config/samples/duplication_v1beta1_deploymentcopy.yaml
DeploymentCopy.duplication.k8s.wantedly.com "canary" created

$ kubectl get deploy -n default
NAME         DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
foo          1         1         1            1           1m
foo-canary   1         1         1            1           9s

$ kubectl get deploy foo-canary -o json | jq '.spec.template.spec.containers[] | if .name == "nginx" then .image else empty end'
"nginx:latest"

```

After testing the new Deployment, you can clean it up by running the following command:

```console
$ kubectl delete -f config/samples/duplication_v1beta1_deploymentcopy.yaml
DeploymentCopy.duplication.k8s.wantedly.com "deploymentcopy-sample" deleted

$ kubectl get deploy
NAME      DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
foo       1         1         1            1           1m
```