[![Release](https://img.shields.io/github/release/Tantalor93/dnspyre/all.svg)](https://github.com/tantalor93/dnspyre/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/tantalor93/dnspyre/v2)](https://goreportcard.com/report/github.com/tantalor93/dnspyre/v2)
[![Tantalor93](https://circleci.com/gh/Tantalor93/dnspyre/tree/master.svg?style=svg)](https://circleci.com/gh/Tantalor93/dnspyre?branch=master)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://github.com/tantalor93/dnspyre/v2/blob/master/LICENSE)
[![codecov](https://codecov.io/gh/Tantalor93/dnspyre/branch/master/graph/badge.svg?token=MC6PK2OLMK)](https://codecov.io/gh/Tantalor93/dnspyre)

# Table of Contents
- [dnspyre](#dnspyre)
    * [Warning](#warning)
    * [Installation](#installation)
    * [Bash/ZSH Shell completion](#Bash/ZSH-Shell-completion)
    * [Usage](#usage)
    * [Examples](#examples)

# dnspyre

Command-line DNS benchmark tool built to stress test and measure the performance of DNS servers. You can easily run benchmark from MacOS, Linux or Windows systems.

This tool is based and originally forked from [dnstrace](https://github.com/redsift/dnstrace), but was largely rewritten and enhanced with additional functionality.

This tool supports wide variety of options to customize DNS benchmark and benchmark output. For example you can:
* benchmark DNS servers with IPv4 and IPv6 addresses (for example GoogleDNS `8.8.8.8` and `2001:4860:4860::8888`)
* benchmark DNS servers with all kinds of query types (A, AAAA, CNAME, HTTPS, ...)
* benchmark DNS servers with a lot of parallel queries and connections (`--number`, `--concurrency` options)
* benchmark DNS servers for a specified duration (`--duration` option)
* benchmark DNS servers using DNS queries over UDP or TCP
* benchmark DNS servers with DoT
* benchmark DNS servers using DoH  
* benchmark DNS servers with uneven random load from provided high volume resources (see `/data` resources and `--probability` option)  
* plot benchmark results via CLI histogram or plot the benchmark results as boxplot, histogram, line graphs and export
them via all kind of image formats (png, svg, pdf)

## Warning

While `dnspyre` is helpful for testing round trip latency via public networks,
the code was primarily created to provide an [apachebench](https://en.wikipedia.org/wiki/ApacheBench)
style tool for testing your own infrastructure.

It is thus very easy to create significant DNS load with non default settings.
**Do not do this to public DNS services**. You will most likely flag your IP.

## Installation 
using `brew`
```
brew tap tantalor93/dnspyre
brew install dnspyre
```

or `go install`
```
go install github.com/tantalor93/dnspyre/v2@latest
```

## Bash/ZSH Shell completion
For **ZSH**, add to your `~/.zprofile` (or equivalent ZSH configuration file)
```
eval "$(dnspyre --completion-script-zsh)"
```

For **Bash**, add to your `~/.bash_profile` (or equivalent Bash configuration file)
```
eval "$(dnspyre --completion-script-bash)"
```

## Usage

```
$ dnspyre --help
usage: dnspyre [<flags>] <queries>...

A high QPS DNS benchmark.

Flags:
      --help                   Show context-sensitive help (also try --help-long and --help-man).
  -s, --server="127.0.0.1"     DNS server IP:port to test. IPv6 is also supported, for example '[fddd:dddd::]:53'. Also DoH servers are supported such as `https://1.1.1.1/dns-query`, when such server is provided, the benchmark automatically switches to
                               the use of DoH. Note that path on which DoH server handles requests (like `/dns-query`) has to be provided as well.
  -t, --type=A ...             Query type. Repeatable flag. If multiple query types are specified then each query will be duplicated for each type.
  -n, --number=NUMBER          How many times the provided queries are repeated. Note that the total number of queries issued = types*number*concurrency*len(queries).
  -c, --concurrency=1          Number of concurrent queries to issue.
  -l, --rate-limit=0           Apply a global questions / second rate limit.
      --query-per-conn=0       Queries on a connection before creating a new one. 0: unlimited
  -r, --recurse                Allow DNS recursion.
      --probability=1          Each hostname from file will be used with provided probability. Value 1 and above means that each hostname from file will be used by each concurrent benchmark goroutine. Useful for randomizing queries across benchmark
                               goroutines.
      --edns0=0                Enable EDNS0 with specified size.
      --ednsopt=""             code[:value], Specify EDNS option with code point code and optionally payload of value as a hexadecimal string. code must be arbitrary numeric value.
      --tcp                    Use TCP fot DNS requests.
      --dot                    Use DoT for DNS requests.
      --write=1s               DNS write timeout.
      --read=4s                DNS read timeout.
      --codes                  Enable counting DNS return codes.
      --min=400µs              Minimum value for timing histogram.
      --max=4s                 Maximum value for histogram.
      --precision=[1-5]        Significant figure for histogram precision.
      --distribution           Display distribution histogram of timings to stdout.
      --csv=/path/to/file.csv  Export distribution to CSV.
      --silent                 Disable stdout.
      --color                  ANSI Color output.
      --plot=/path/to/folder   Plot benchmark results and export them to directory.
      --plotf=png              Format of graphs. Supported formats: png, jpg.
      --doh-method=post        HTTP method to use for DoH requests. Supported values: get, post.
      --doh-protocol=1.1       HTTP protocol to use for DoH requests. Supported values: 1.1, 2.
      --insecure               disables server TLS certificate validation
  -d, --duration=1m            Specifies for how long the benchmark should be executing, the benchmark will run for the specified time while sending DNS requests in infinite loop based on data source. After running for specified duration, the benchmark
                               is cancelled. This option is exclusive with --number option. The duration is specified in GO duration format e.g. 10s, 15m, 1h.
      --version                Show application version.

Args:
  <queries>  Queries to issue. Can be local file referenced using @<file-path>, for example @data/2-domains.Can also be resource accessible using HTTP, like https://raw.githubusercontent.com/Tantalor93/dnspyre/master/data/1000-domains, in that case the
             file will be downloaded and saved inmemory.
```

## Examples

For examples of usage, see [examples](docs/examples.md)
