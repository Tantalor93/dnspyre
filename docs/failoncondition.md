---
title: Fail on condition
layout: default
parent: Examples
---



# Fail on condition
v3.1.0
{: .label .label-yellow }
*dnspyre* by default always returns a zero exit code, but this behaviour can be adjusted by using `--fail <condition>` flag, 
which can be used to specify predefined conditions that will cause a *dnspyre* to return a non-zero exit code.

Currently, the dnspyre supports these fail conditions:
* `ioerror` = *dnspyre* exits with a non-zero status code if there is at least 1 IO error (*dnspyre* failed to send DNS request or receive DNS response)
* `negative` = *dnspyre* exits with a non-zero status code if there is at least 1 negative DNS answer (`NXDOMAIN` or `NODATA` response)
* `error` = *dnspyre* exits with a non-zero status code if there is at least 1 error DNS response (`SERVFAIL`, `FORMERR`, `REFUSED`, etc.)
* `idmismatch` = *dnspyre* exits with a non-zero status code if there is at least 1 ID mismatch between DNS request and response

So for example to return a non-zero exit code, when benchmark fails to send request or receive response you would specify `--fail ioerror` flag
```
dnspyre --server 1.2.3.4 google.com --fail ioerror
```

These fail conditions can be combined, this is achieved by repeating the `--flag` flag multiple times with different conditions.
```
dnspyre --server 8.8.8.8 nxdomain.cz  --fail ioerror --fail error --fail negative
```
