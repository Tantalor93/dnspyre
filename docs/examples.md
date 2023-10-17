---
title: Examples
layout: default
nav_order: 2
has_children: true
---

# Examples
{: .no_toc }
This page shows examples of *dnspyre* usage and its various options

* TOC
{:toc}


## Sending queries from multiple parallel threads
This example will execute the benchmark in 10 parallel threads (option `-c`) , where each thread will
send 2 (option `-n`) `example.com.` DNS queries of type `A` one after another to the `8.8.8.8` server
```
dnspyre -n 2 -c 10 --server 8.8.8.8 example.com
```

## Run benchmark over specified time
This example will execute the benchmark in 10 parallel threads for a duration of 30 seconds while sending `example.com` DNS queries of type `A`
to the `8.8.8.8` server
```
dnspyre --duration 30s -c 10 --server 8.8.8.8 google.com
```

## Sending AAAA DNS queries
You can choose, which type of query to send to the DNS server using `-t` option, 
```
dnspyre -n 2 -c 10 --server 8.8.8.8 -t AAAA example.com
```

## Pass multiple hostnames
You can pass arbitrary number of domains to be used for the DNS benchmark, by specifying more arguments, in this example domains
`redsift.io`, `example.com`, `google.com` will be used to generate DNS queries
```
dnspyre -n 10 -c 10 --server 8.8.8.8 redsift.io example.com google.com
```

## Hostnames provided using file on local filesystem
Instead of specifying hostnames as arguments to *dnspyre* tool, you can just specify file containing hostnames to be used by the tool,
by referencing the file using `@<path-to-file>`
```
dnspyre -n 10 -c 10 --server 8.8.8.8 @data/2-domains
```

## Hostnames provided using file publicly available using HTTP(s) 
The file containing hostnames does not need to be available locally, it can be also downloaded from the remote location using HTTP(s)
```
dnspyre -n 10 -c 10 --server 8.8.8.8 https://raw.githubusercontent.com/Tantalor93/dnspyre/master/data/2-domains
```

## Combining multiple query types in the benchmark
Multiple DNS query types can be specified for *dnspyre* tool. 
This can be achieved by repeating type `-t`, all queries will be made by each specified query type
```
dnspyre -n 10 -c 10 --server 8.8.8.8 -t A -t AAAA @data/2-domains
```
together with probability option this can be used for generating arbitrary random load
```
dnspyre -n 10 -c 10 --server 8.8.8.8 -t A -t AAAA @data/2-domains --probability 0.33
```

## IPv6 DNS server benchmarking
DNS server address can be also provided as an IPv6 address, note the brackets format when specifying port
```
dnspyre -n 10 -c 10 --server '[2001:4860:4860::8888]:53' idnes.cz
```

or 

```
dnspyre -n 10 -c 10 --server '2001:4860:4860::8888' idnes.cz
```

## Using probability to randomize concurrent queries
You can randomize queries fired by each concurrent thread by using probability lesser than 1, in this example
roughly every third hostname from the datasource will be used by the each concurrent benchmark thread
```
dnspyre --duration 30s -c 10 --server 8.8.8.8  --probability 0.33  https://raw.githubusercontent.com/Tantalor93/dnspyre/master/data/1000-domains
```

## EDNSOPT usage
you can also specify EDNS option with arbitrary payload, here we are specifying EDNSOPT `65518`
coming from the local/experimental range with payload `fddddddd100000000000000000000001`
```
dnspyre -n 10 -c 10 idnes.cz --server 127.0.0.1 --ednsopt=65518:fddddddd100000000000000000000001
```

## Output benchmark results as JSON
By specifying `--json` flag, *dnspyre* can output benchmark results in a JSON format, which is better for further automatic processing
```
dnspyre --duration 5s --server 8.8.8.8 google.com  --json
```

example of chaining of *dnspyre* with [jq](https://stedolan.github.io/jq/) for getting pretty JSON
```
dnspyre --duration 5s --server 8.8.8.8 google.com --no-distribution --json | jq '.'
```
like this
```
{
  "totalRequests": 276,
  "totalSuccessCodes": 276,
  "totalErrors": 0,
  "TotalIDmismatch": 0,
  "totalTruncatedResponses": 0,
  "responseRcodes": {
    "NOERROR": 276
  },
  "questionTypes": {
    "A": 276
  },
  "queriesPerSecond": 55.18,
  "benchmarkDurationSeconds": 5,
  "latencyStats": {
    "minMs": 12,
    "meanMs": 18,
    "stdMs": 13,
    "maxMs": 176,
    "p99Ms": 71,
    "p95Ms": 33,
    "p90Ms": 24,
    "p75Ms": 15,
    "p50Ms": 14
  }
}
```
