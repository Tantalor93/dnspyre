---
title: DNS Cookies
layout: default
parent: Examples
---


# DNS cookies
v3.9.0
{: .label .label-yellow }

**dnspyre** supports DNS cookies as defined in [RFC-7873](https://datatracker.ietf.org/doc/html/rfc7873)
and [RFC-9018](https://datatracker.ietf.org/doc/html/rfc9018). DNS cookies are be enabled by providing `--cookie` flag

```
dnspyre --cookie -n 3 -c 2 www.example.com
```

Each concurrent worker generates a unique 8-byte Client Cookie. This cookie is reused for all subsequent requests by that worker:
* The initial request contains only the Client Cookie.
* Subsequent requests will include the concatenated Client + Server Cookie (if the server returned one in the previous response).

{: .note }
Most of the public DNS resolvers like Google (8.8.8.8) and Cloudflare (1.1.1.1) does not work and support DNS cookies and 
they will actively ignore the cookies. It usually works with the ISP DNS servers.
