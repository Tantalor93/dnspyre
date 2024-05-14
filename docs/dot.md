---
title: DoT
layout: default
parent: Examples
---

# DoT
*dnspyre* supports running benchmarks against [RFC-7858](https://datatracker.ietf.org/doc/html/rfc7858) compatible DNS over TLS servers

```
dnspyre --dot --server 8.8.8.8:853 idnes.cz
```

also you can provide a DNS server hostname instead of the server IP address

```
dnspyre --dot --server dns.google google.com
```

## DoT with self-signed certificates
In some cases you might want to skip invalid and self-signed certificates, this can be achieved by using `--insecure` argument

```
dnspyre --server 127.0.0.1:5553 --dot --insecure google.com
```
