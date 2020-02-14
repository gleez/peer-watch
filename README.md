# Pod-watch sidecar container

Borrowed from https://github.com/robwil/peer-aware-groupcache
https://stackoverflow.com/questions/52951777/listing-pods-from-inside-a-pod-forbidden

### Pod role

Tutorial: https://medium.com/better-programming/k8s-tips-using-a-serviceaccount-801c433d0023

##### Create the pod-reader ROLE

```console
$ kubectl create role pod-reader --verb=get,list,watch --resource=pods -n dev
```

##### Create ROLE BINDING and attach the role to the dev:default service account

```console
$ kubectl create rolebinding sa-read-pods –-role=pod-reader –-user=system:serviceaccount:dev:default -n dev
```