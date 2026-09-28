# vedro

Operator that provides a Kubernetes-native way to manage object storage resources and access across multiple cloud providers.

The name vedro derives from "ведро", a Russian word for bucket.


## Description

Create buckets, cloud service accounts, IAM roles, access keys, and bucket permissions for various cloud providers using Kubernetes Custom Resources.

Typical use cases include:

```
Create and configure object storage buckets for applications.
Create or reference cloud IAM principals.
Grant principals read/write/admin access to buckets.
Create credentials and use them in applications.
```

More indepth info about architecture why/what/when can be found [here](https://github.com/svetoch-dev/rod-docs/tree/master/docs/proposals/architecture/3-manage-cloud-dependencies-in-k8s)


## Contributing

- go version v1.26.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.

**Build and push your image to the location specified by `IMG`:**

```sh
make docker-build docker-push IMG=<some-registry>/vedro:tag
```

**NOTE:** This image ought to be published in the personal registry you specified.
And it is required to have access to pull the image from the working environment.
Make sure you have the proper permission to the registry if the above commands don’t work.

### Deploy via manifests

**Install the CRDs into the cluster:**

```sh
make install
```

**Deploy the Manager to the cluster with the image specified by `IMG`:**

```sh
make deploy IMG=<some-registry>/vedro:tag
```

> **NOTE**: If you encounter RBAC errors, you may need to grant yourself cluster-admin
privileges or be logged in as admin.


You can apply the samples (examples) from the config/sample:


** GCP **

Install

```sh
GCP_PROJECT_ID=<YOUR PROJECT ID> envsubst '${GCP_PROJECT_ID}' < config/samples/manifests/gcp.yaml | kubectl apply -f -
```

Delete

```sh
kubectl delete -f config/samples/manifests/gcp.yaml
```

** YC **

Install

```sh
YC_PROJECT_ID=b1gfu8oas3od212hedtu envsubst '${YC_PROJECT_ID}' < config/samples/manifests/yc.yaml | kubectl apply -f -
```

Delete

```sh
kubectl delete -f config/samples/manifests/yc.yaml

```

**Delete the APIs(CRDs) from the cluster:**

```sh
make uninstall
```

**UnDeploy the controller from the cluster:**

```sh
make undeploy
```

### Deploy via helm

**Install controller helm chart**

```sh
helm upgrade --install vedro-gcp-int helm/controller/
```

**Install test manifests**

**GCP**

Install

```sh
helm upgrade --install sample-gcp helm/vedro/ --values config/samples/helm/values.yaml --set providers.sample.projectId=some-project
```

Delete

```sh
helm delete sample-gcp
```

**YC**

Install

```sh
helm upgrade --install sample-yc helm/vedro/ --values config/samples/helm/values.yaml --set providers.sample.projectId=dawd1212e1e1d1
```

Delete

```sh
helm delete sample-yc
```
