---
title: DoT
layout: default
parent: Examples
---

# DoT
{: .no_toc }

*dnspyre* supports running benchmarks against [RFC-7858](https://datatracker.ietf.org/doc/html/rfc7858) compatible DNS over TLS servers

```
dnspyre --dot --server 1.1.1.1:853 idnes.cz
```

## DoT with self-signed certificates
In some cases you might want to skip invalid and self-signed certificates, this can be achieved by using `--insecure` argument

```
dnspyre --server 127.0.0.1:5553 --dot --insecure google.com
```
