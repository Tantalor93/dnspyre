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
kubectl run dnspyre --restart=Never --image=ghcr.io/tantalor93/dnspyre -- https://raw.githubusercontent.com/Tantalor93/dnspyre/master/data/alexa --server kube-dns.kube-system.svc.cluster.local --duration 1m
```

and then check the output using

```
kubectl logs dnspyre
```

You might want to test the performance from multiple instances/pods, this can be easily achieved by deploying *dnspyre* in multiple pods,
for example using Kubernetes Deployment :

```
apiVersion: apps/v1
kind: Deployment
metadata:
  name: dnspyre-deployment
spec:
  replicas: 2
  selector:
    matchLabels:
      app: dnspyre
  template:
    metadata:
      labels:
        app: dnspyre
    spec:
      containers:
      - name: dnspyre
        image: ghcr.io/tantalor93/dnspyre
        command:
        - "/dnspyre"
        args:
        - "https://raw.githubusercontent.com/Tantalor93/dnspyre/master/data/alexa"
        - "--server"
        - "kube-dns.kube-system.svc.cluster.local"
        - "--duration"
        - "1m"
        - "-c"
        - "100"
        resources:
          limits:
            cpu: "1"      
            memory: "900Mi" 
          requests:
            cpu: "0.1"      
            memory: "128Mi" 
```
and then applying this Deployment to your cluster:

```
kubectl apply -f dnspyre-deployment.yml
```
