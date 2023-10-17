---
title: DoH
layout: default
parent: Examples
---

# DoH
{: .no_toc }

*dnspyre* supports running benchmarks against [RFC-8484](https://www.rfc-editor.org/rfc/rfc8484) compatible DNS over HTTPS servers
```
dnspyre --server 'https://1.1.1.1' google.com
```

See other examples of customization of DoH benchmarks
* TOC
{:toc}


## DoH via GET/POST
you can also specify whether the DoH is done via GET or POST using `--doh-method`
```
dnspyre --server 'https://1.1.1.1' --doh-method get google.com
```

benchmarking DoH server via DoH over POST method 
```
dnspyre --server 'https://1.1.1.1' --doh-method post google.com
```

## DoH/1.1, DoH/2, DoH/3
you can also specify whether the DoH is done over HTTP/1.1, HTTP/2, HTTP/3 using `--doh-protocol`, for example:
```
dnspyre --server 'https://1.1.1.1' --doh-protocol 2 google.com
```

## DoH via plain HTTP
even plain HTTP without TLS can be used as transport for DoH requests, this is configured based on server URL containing either `https://` or `http://`

```
dnspyre --server http://127.0.0.1 google.com
```

## DoH with self-signed certificates
In some cases you might want to skip invalid and self-signed certificates, this can be achieved by using `--insecure` argument

```
dnspyre --server https://127.0.0.1  --insecure google.com
```
