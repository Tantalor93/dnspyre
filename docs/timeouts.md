---
title: Configuring timeouts
layout: default
parent: Examples
---

# Configuring timeouts
*dnspyre* supports configuring various timeouts applied on outgoing DNS requests:
* **connect timeout** - timeout for establishing connection to a DNS server, configurable using `--connect` flag
* **write timeout** - timeout for writing a request to a DNS server, configurable using `--write` flag
* **read timeout** - timeout for reading a response from a DNS server, configurable using `--read` flag
* **request timeout** - overall timeout for establishing connection, sending request and reading response, configurable using `--request` flag

For example to limit request timeout to 100ms, you would pass `--request` flag with value `100ms`
```
dnspyre --request 100ms --duration 10s --server 'quic://dns.adguard-dns.com' https://raw.githubusercontent.com/Tantalor93/dnspyre/master/data/1000-domains
```
