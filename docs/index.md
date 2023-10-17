---
title: Home
layout: home
nav_order: 0
---

# dnspyre

*dnspyre* is a command-line DNS benchmark tool built to stress test and measure the performance of DNS servers. You can easily run benchmark from MacOS, Linux or Windows systems.
This tool is based and originally forked from [dnstrace](https://github.com/redsift/dnstrace), but was largely rewritten and enhanced with additional functionality.

This tool supports wide variety of options to customize DNS benchmark and benchmark output. For example, you can:
* benchmark DNS servers using DNS queries over UDP or TCP, see [plain DNS example](plaindns.md)
* benchmark DNS servers with all kinds of query types like A, AAAA, CNAME, HTTPS, ... (`--type` option)
* benchmark DNS servers with a lot of parallel queries and connections (`--number`, `--concurrency` options)
* benchmark DNS servers for a specified duration (`--duration` option)
* benchmark DNS servers with DoT ([DNS over TLS](https://datatracker.ietf.org/doc/html/rfc7858)), see [DoT example](dot.md)
* benchmark DNS servers using DoH ([DNS over HTTPS](https://datatracker.ietf.org/doc/html/rfc8484)), see [DoH example](doh.md)
* benchmark DNS servers using DoQ ([DNS over QUIC](https://datatracker.ietf.org/doc/rfc9250/)), see [DoQ example](doq.md)
* benchmark DNS servers with uneven random load from provided high volume resources (see `--probability` option)
* plot benchmark results via CLI histogram or plot the benchmark results as boxplot, histogram, line graphs and export them via all kind of image formats like png, svg and pdf. (see `--plot` and `--plotf` options) 

## Usage

```
usage: dnspyre [<flags>] <queries>...

A high QPS DNS benchmark.


Flags:
      --[no-]help              Show context-sensitive help (also try --help-long and --help-man).
  -s, --server="127.0.0.1"     DNS server IP:port to test. IPv6 is also supported, for example '[fddd:dddd::]:53'. DoH (DNS over HTTPS) servers are supported such as `https://1.1.1.1`, when such server is provided, the benchmark automatically switches to
                               the use of DoH. Note that by default path `/dns-query` is assumed for DoH queries, if no path is provided in server parameter. DoQ (DNS over QUIC) servers are also supported, such as `quic://dns.adguard-dns.com`, when such
                               server is provided the benchmark switches to the use of DoQ.
  -t, --type=A ...             Query type. Repeatable flag. If multiple query types are specified then each query will be duplicated for each type.
  -n, --number=NUMBER          How many times the provided queries are repeated. Note that the total number of queries issued = types*number*concurrency*len(queries).
  -c, --concurrency=1          Number of concurrent queries to issue.
  -l, --rate-limit=0           Apply a global questions / second rate limit.
      --rate-limit-worker=0    Apply a questions / second rate limit for each concurrent worker specified by --concurrency option.
      --query-per-conn=0       Queries on a connection before creating a new one. 0: unlimited. Applicable for plain DNS and DoT, this option is not considered for DoH or DoQ.
  -r, --[no-]recurse           Allow DNS recursion. Enabled by default.
      --probability=1          Each provided hostname will be used with provided probability. Value 1 and above means that each hostname will be used by each concurrent benchmark goroutine. Useful for randomizing queries across benchmark goroutines.
      --ednsopt=""             code[:value], Specify EDNS option with code point code and optionally payload of value as a hexadecimal string. code must be an arbitrary numeric value.
      --[no-]tcp               Use TCP for DNS requests.
      --[no-]dot               Use DoT (DNS over TLS) for DNS requests.
      --write=1s               write timeout.
      --read=3s                read timeout.
      --connect=1s             connect timeout.
      --request=5s             request timeout.
      --[no-]codes             Enable counting DNS return codes. Enabled by default.
      --min=400µs              Minimum value for timing histogram.
      --max=MAX                Maximum value for timing histogram.
      --precision=[1-5]        Significant figure for histogram precision.
      --[no-]distribution      Display distribution histogram of timings to stdout. Enabled by default.
      --csv=/path/to/file.csv  Export distribution to CSV.
      --[no-]json              Report benchmark results as JSON.
      --[no-]silent            Disable stdout.
      --[no-]color             ANSI Color output. Enabled by default.
      --plot=/path/to/folder   Plot benchmark results and export them to the directory.
      --plotf=png              Format of graphs. Supported formats: png, jpg.
      --doh-method=post        HTTP method to use for DoH requests. Supported values: get, post.
      --doh-protocol=1.1       HTTP protocol to use for DoH requests. Supported values: 1.1, 2 and 3.
      --[no-]insecure          Disables server TLS certificate validation. Applicable for DoT, DoH and DoQ.
  -d, --duration=1m            Specifies for how long the benchmark should be executing, the benchmark will run for the specified time while sending DNS requests in an infinite loop based on the data source. After running for the specified duration,
                               the benchmark is canceled. This option is exclusive with --number option. The duration is specified in GO duration format e.g. 10s, 15m, 1h.
      --[no-]version           Show application version.

Args:
  <queries>  Queries to issue. It can be a local file referenced using @<file-path>, for example @data/2-domains. It can also be resource accessible using HTTP, like https://raw.githubusercontent.com/Tantalor93/dnspyre/master/data/1000-domains, in that
             case, the file will be downloaded and saved in-memory. These data sources can be combined, for example "google.com @data/2-domains https://raw.githubusercontent.com/Tantalor93/dnspyre/master/data/2-domains"
```
