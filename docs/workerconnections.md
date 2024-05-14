---
title: Configuring connection sharing
layout: default
parent: Examples
---

# Configuring connection sharing
v3.3.0
{: .label .label-yellow }
*dnspyre* by default tries to share connections between spawned concurrent workers as much as possible, so for example 
if DoH benchmark over HTTPS/2 with multiple concurrent workers is executed, then all the workers will share same single HTTPS/2 connection 
to the DNS server

```
dnspyre --server https://1.1.1.1 google.com -c 5 --doh-protocol 2
```

If you want to avoid sharing connection between concurrent workers, you can use `--separate-worker-connections` flag

```
dnspyre --server https://1.1.1.1 google.com -c 5 --doh-protocol 2 --separate-worker-connections
```
