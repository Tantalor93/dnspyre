---
title: Home
layout: home
nav_order: 0
---

# dnspyre

Command-line DNS benchmark tool built to stress test and measure the performance of DNS servers. You can easily run benchmark from MacOS, Linux or Windows systems.

This tool is based and originally forked from [dnstrace](https://github.com/redsift/dnstrace), but was largely rewritten and enhanced with additional functionality.

This tool supports wide variety of options to customize DNS benchmark and benchmark output. For example, you can:
* benchmark DNS servers using DNS queries over UDP or TCP, see [plain DNS example](plaindns.md)
* benchmark DNS servers with all kinds of query types like A, AAAA, CNAME, HTTPS, ... (`--type` option)
* benchmark DNS servers with a lot of parallel queries and connections (`--number`, `--concurrency` options)
* benchmark DNS servers for a specified duration (`--duration` option)
* benchmark DNS servers with DoT, see [DoQ example](doq.md)
* benchmark DNS servers using DoH, see [DoH example](doh.md)
* benchmark DNS servers using DoQ, see [DoQ example](doq.md)
* benchmark DNS servers with uneven random load from provided high volume resources (see `--probability` option)
* plot benchmark results via CLI histogram or plot the benchmark results as boxplot, histogram, line graphs and export them via all kind of image formats like png, svg and pdf. (see `plot` and `plotf` options) 

## Usage

```
usage: dnspyre [<flags>] <queries>...

A high QPS DNS benchmark.

Flags:
      --help                   Show context-sensitive help (also try --help-long and --help-man).
  -s, --server="127.0.0.1"     DNS server IP:port to test. IPv6 is also supported, for example '[fddd:dddd::]:53'. DoH (DNS over HTTPS) servers are supported such as `https://1.1.1.1/dns-query`, when such server is provided, the benchmark automatically
                               switches to the use of DoH. Note that path on which the DoH server handles requests (like `/dns-query`) has to be provided as well. DoQ (DNS over QUIC) servers are also supported, such as `quic://dns.adguard-dns.com`,
                               when such server is provided the benchmark switches to the use of DoQ.
  -t, --type=A ...             Query type. Repeatable flag. If multiple query types are specified then each query will be duplicated for each type.
  -n, --number=NUMBER          How many times the provided queries are repeated. Note that the total number of queries issued = types*number*concurrency*len(queries).
  -c, --concurrency=1          Number of concurrent queries to issue.
  -l, --rate-limit=0           Apply a global questions / second rate limit.
      --query-per-conn=0       Queries on a connection before creating a new one. 0: unlimited. Applicable for plain DNS and DoT, this option is not considered for DoH or DoQ.
  -r, --recurse                Allow DNS recursion. Enabled by default. DNS recursion can be disabled by --no-recurse.
      --probability=1          Each provided hostname will be used with provided probability. Value 1 and above means that each hostname will be used by each concurrent benchmark goroutine. Useful for randomizing queries across benchmark goroutines.
      --edns0=0                Enable EDNS0 with specified size.
      --ednsopt=""             code[:value], Specify EDNS option with code point code and optionally payload of value as a hexadecimal string. code must be an arbitrary numeric value.
      --tcp                    Use TCP for DNS requests.
      --dot                    Use DoT (DNS over TLS) for DNS requests.
      --write=1s               DNS write timeout.
      --read=4s                DNS read timeout.
      --codes                  Enable counting DNS return codes. Enabled by default. Specifying --no-codes disables code counting.
      --min=400µs              Minimum value for timing histogram.
      --max=4s                 Maximum value for timing histogram.
      --precision=[1-5]        Significant figure for histogram precision.
      --distribution           Display distribution histogram of timings to stdout. Enabled by default. Specifying --no-distribution disables histogram display.
      --csv=/path/to/file.csv  Export distribution to CSV.
      --json                   Report benchmark results as JSON.
      --silent                 Disable stdout.
      --color                  ANSI Color output. Enabled by default. By specifying --no-color disables coloring.
      --plot=/path/to/folder   Plot benchmark results and export them to the directory.
      --plotf=png              Format of graphs. Supported formats: png, jpg.
      --doh-method=post        HTTP method to use for DoH requests. Supported values: get, post.
      --doh-protocol=1.1       HTTP protocol to use for DoH requests. Supported values: 1.1, 2 and 3.
      --insecure               Disables server TLS certificate validation. Applicable for DoT, DoH and DoQ.
  -d, --duration=1m            Specifies for how long the benchmark should be executing, the benchmark will run for the specified time while sending DNS requests in an infinite loop based on the data source. After running for the specified duration,
                               the benchmark is canceled. This option is exclusive with --number option. The duration is specified in GO duration format e.g. 10s, 15m, 1h.
      --version                Show application version.

Args:
  <queries>  Queries to issue. It can be a local file referenced using @<file-path>, for example @data/2-domains. It can also be resource accessible using HTTP, like https://raw.githubusercontent.com/Tantalor93/dnspyre/master/data/1000-domains, in that
             case, the file will be downloaded and saved in-memory.
```
