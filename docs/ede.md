---
title: Extended DNS Errors
layout: default
parent: Examples
---

# Extended DNS Errors
v3.12.0
{: .label .label-yellow }
*dnspyre* supports reporting Extended DNS Errors as specified by [RFC-8914](https://www.rfc-editor.org/info/rfc8914/). Though this feature requires benchmark to advertise EDNS0 support (for example by specifying `--edns0`, see [EDNS0](edns0.md))


Example of DNSSEC-broken domain with validating DNS resolver
```
dnspyre --server 1.1.1.1 dnssec-failed.org --edns0
```

Example of domain blocked by DNS resolver:
```
dnspyre --server 9.9.9.9 isitblocked.org  --edns0
```
