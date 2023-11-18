---
title: DNS query types
layout: default
parent: Examples
---

# DNS query types
You can choose, which type of query to send to the DNS server using `-t` option, by default *dnspyre* generates *A* (IPv4 DNS query).
In this example, the *dnspyre* is configured to send *AAAA* queries (IPv6 queries)

```
dnspyre -n 2 -c 10 --server 8.8.8.8 -t AAAA example.com
```

## Combining multiple query types in the benchmark
Multiple DNS query types can be specified for *dnspyre* tool. This can be achieved by repeating type `-t`, the queries for domains will be 
repeated for each specified type

```
dnspyre -n 10 -c 10 --server 8.8.8.8 -t A -t AAAA example.com
```
