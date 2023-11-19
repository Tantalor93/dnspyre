---
title: Kubernetes
layout: default
parent: Examples
---

# Kubernetes

One of the use cases for using *dnspyre* [Docker image](https://github.com/Tantalor93/dnspyre/pkgs/container/dnspyre) is testing the performance of
the internal DNS server from inside of your Kubernetes cluster. This can be achieved by running a *dnspyre* docker image inside a Kubernetes pod,
for example by running a kubectl command like this:

```
kubectl run dnspyre --restart=Never --image=ghcr.io/tantalor93/dnspyre -- https://raw.githubusercontent.com/Tantalor93/dnspyre/master/data/top-1m --server kube-dns.kube-system.svc.cluster.local --duration 1m
```

and then check the output using

```
kubectl logs dnspyre
```
