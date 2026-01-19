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

### EDNS Client Subnet (ECS)
For easier specification of [EDNS Client Subnet (ECS)](https://datatracker.ietf.org/doc/html/rfc7871) option, you can use the `--ecs` flag with CIDR notation instead of manually constructing the hex string with `--ednsopt`.

#### IPv4 example
```
dnspyre --server '8.8.8.8' aws.amazon.com --ecs '204.15.220.0/22'
```

#### IPv6 example
```
dnspyre --server '8.8.8.8' aws.amazon.com --ecs '2001:db8::/32'
```

The `--ecs` flag can be combined with `--ednsopt` to send additional EDNS options (as long as `--ednsopt` doesn't use code 8, which is reserved for ECS):

```
dnspyre --server '8.8.8.8' aws.amazon.com --ecs '192.0.2.0/24' --ednsopt '10:0001'
```
