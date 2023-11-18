---
title: EDNS0 and DNSSEC
layout: default
parent: Examples
---

# EDNS0 and DNSSEC
*dnspyre* supports sending DNS requests with [EDNS0](https://datatracker.ietf.org/doc/html/rfc6891) extension, currently these EDNS0 features are supported:

## UDP message size
advertisement for support of larger DNS response size (UDP message size) using `--edns0` flag

```
dnspyre  --server '1.1.1.1' google.com --edns0=1024
```

## DNSSEC
[DNSSEC](https://datatracker.ietf.org/doc/html/rfc9364) security extension using `--dnssec` flag, by using this flag the *dnspyre* will also
count the **number of domains that were successfully validated by DNS resolver**

```
dnspyre  --server '1.1.1.1' cloudflare.com --dnssec
```

## EDNS0 options
sending various EDNS0 options using `--ednsopt` flag, you have to specify the decimal **EDNS0 option code** (see [IANA registry](https://www.iana.org/assignments/dns-parameters/dns-parameters.xhtml#dns-parameters-11)) and hex-string representing **EDNS0 option data**,
data format depends on the EDNS0 option

for example to send [client subnet EDNS0 option](https://datatracker.ietf.org/doc/html/rfc7871) for subnet `81.0.198.170/24` you specify code `8` and data `000118005100c6` (`0001` = IPv4 Family, `18` = source mask `/24`, `00` = no additional scope, `5100C6AA` = `81.0.198.170` )

```
dnspyre  --server '8.8.8.8' aws.amazon.com --ednsopt '8:000118005100c6'
```
