---
title: Plain DNS
layout: default
parent: Examples
---

# Plain DNS
*dnspyre* supports running benchmarks against DNS servers using plain DNS over UDP (default option)

```
dnspyre --server 8.8.8.8 google.com
```
or you can benchmark using DNS over TCP

```
dnspyre --tcp --server 8.8.8.8 google.com
```
