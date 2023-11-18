---
title: Domain sources
layout: default
parent: Examples
---

# Domain sources
*dnspyre* benchmarks DNS servers by querying the domains specified as arguments to the tool, the domains to the tool can be passed in a various ways:


## Domains provided directly as arguments
You can pass an arbitrary number of domains to be used for the DNS benchmark, by specifying more arguments. In this example, domains
`redsift.io`, `example.com`, `google.com` are used to generate DNS queries

```
dnspyre -n 10 -c 10 --server 8.8.8.8 redsift.io example.com google.com
```

## Domains provided using file on local filesystem
Instead of specifying domains as arguments to the *dnspyre* tool, you can just specify a file containing domains to be used by the tool.
By referencing the file using `@<path-to-file>`. In this example, the domains are read from the `data/2-domains` file.

```
dnspyre -n 10 -c 10 --server 8.8.8.8 @data/2-domains
```

## Domains provided using file publicly available using HTTP(s)
The file containing hostnames does not need to be available locally, it can be also downloaded from the remote location using HTTP(s).
In this example, the domains are downloaded from the https://raw.githubusercontent.com/Tantalor93/dnspyre/master/data/2-domains

```
dnspyre -n 10 -c 10 --server 8.8.8.8 https://raw.githubusercontent.com/Tantalor93/dnspyre/master/data/2-domains
```
