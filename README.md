# Pod-watch sidecar container

Borrowed from https://github.com/robwil/peer-aware-groupcache


### Pod role

Tutorial: https://medium.com/better-programming/k8s-tips-using-a-serviceaccount-801c433d0023

##### Cluster Role example

```
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: pod-reader
rules:
- apiGroups: [""]
  resources: ["pods"]
  verbs: ["get", "watch", "list"]
---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: pod-reader
subjects:
- kind: ServiceAccount
  name: default
  namespace: default
roleRef:
  kind: ClusterRole
  name: pod-reader
  apiGroup: rbac.authorization.k8s.io
```

##### Role example - Create the pod-reader role in dev namespace

```console
$ kubectl create role pod-reader --verb=get,list,watch --resource=pods -n dev
```

##### Create ROLE BINDING and attach the role to the dev:default service account in dev namespace

```console
$ kubectl create rolebinding sa-read-pods –-role=pod-reader –-user=system:serviceaccount:dev:default -n dev
```