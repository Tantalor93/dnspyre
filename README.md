[![Release](https://img.shields.io/github/release/Tantalor93/dnstrace/all.svg)](https://github.com/Tantalor93/dnstrace/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/Tantalor93/dnstrace)](https://goreportcard.com/report/github.com/Tantalor93/dnstrace)
[![Tantalor93](https://circleci.com/gh/Tantalor93/dnstrace/tree/master.svg?style=svg)](https://circleci.com/gh/Tantalor93/dnstrace?branch=master)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://github.com/Tantalor93/dnstrace/blob/master/LICENSE)

# Table of Contents
- [DNStrace](#dnstrace)
    * [Installation](#installation)
    * [Build](#build)
    * [Usage](#usage)
    * [Warning](#warning)
    * [Example](#example)
        + [IPv6 DNS server benchmarking](#ipv6-dns-server-benchmarking)
        + [hostnames provided directly](#hostnames-provided-directly)
        + [hostnames provided using file](#hostnames-provided-using-file)
        + [using probability to randomize concurrent queries](#using-probability-to-randomize-concurrent-queries)
        + [EDNSOPT usage](#ednsopt-usage)
        + [DoT](#dot)

# DNStrace
forked https://github.com/redsift/dnstrace 

Command-line DNS benchmark tool built to stress test and measure the performance of DNS servers with commodity hardware.
This tool typically consumes ~30kB per concurrent connection and can maintain ~30,000 QPS per modern core if your server, OS and network allows you to reach suitable levels of concurrency.

## Installation 
```
go get github.com/tantalor93/dnstrace
```
will install the binary in your $GOPATH/bin

or download compiled binaries


linux
```
wget https://github.com/Tantalor93/dnstrace/releases/download/1.3.0/dnstrace-linux-amd64
```
darwin(macos)
```
wget https://github.com/Tantalor93/dnstrace/releases/download/1.3.0/dnstrace-darwin
```

## Build
```
make build
```
binaries will be in `bin/` folder

## Usage

```
$ dnstrace --help
usage: dnstrace [<flags>] <queries>...

A high QPS DNS benchmark.

Flags:
      --help                   Show context-sensitive help (also try --help-long and --help-man).
  -s, --server="127.0.0.1"     DNS server IP:port to test. IPv6 is also supported, for example '[fddd:dddd::]:53'.
  -t, --type=A                 Query type.
  -n, --number=1               Number of queries to issue. Note that the total number of queries issued = number*concurrency*len(queries).
  -c, --concurrency=1          Number of concurrent queries to issue.
  -l, --rate-limit=0           Apply a global questions / second rate limit.
      --query-per-conn=0       Queries on a connection before creating a new one. 0: unlimited
  -e, --expect=EXPECT ...      Expect a specific response.
  -r, --recurse                Allow DNS recursion.
      --probability=1          Each hostname from file will be used with provided probability in %. Value 1 and above means that each hostname from file will be used by each concurrent benchmark
                               goroutine. Useful for randomizing queries across benchmark goroutines.
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
      --io-errors              Log I/O errors to stderr.
      --silent                 Disable stdout.
      --color                  ANSI Color output.
      --version                Show application version.

Args:
  <queries>  Queries to issue. Can be file referenced using @<file-path>, for example @data/2-domains
```

## Warning

While `dnstrace` is helpful for testing round trip latency via public networks,
the code was primarily created to provide an [apachebench](https://en.wikipedia.org/wiki/ApacheBench)
style tool for testing your own infrastructure.

It is thus very easy to create significant DNS load with non default settings.
**Do not do this to public DNS services**. You will most likely flag your IP.

## Example
### IPv6 DNS server benchmarking
```
$ dnstrace -n 10 -c 10 --server '[fddd:dddd::]:53' --recurse idnes.cz
Using 1 hostnames
Benchmarking [fddd:dddd::]:53 via udp with 10 concurrent requests

Total requests:	 100
DNS success codes:	100
Truncated responses:	0

DNS response codes:
	NOERROR:	100

Time taken for tests:	 470.980954ms
Questions per second:	 212.3

DNS timings, 100 datapoints
	 min:		 35.651584ms
	 mean:		 45.109739ms
	 [+/-sd]:	 6.021748ms
	 max:		 79.691775ms
	 p99:		 71.303167ms
	 p95:		 58.720255ms
	 p90:		 50.331647ms
	 p75:		 46.137343ms
	 p50:		 46.137343ms

DNS distribution, 100 datapoints
    LATENCY   |                                             | COUNT
+-------------+---------------------------------------------+-------+
  36.700159ms | ▄▄▄▄▄                                       |     4
  38.797311ms | ▄▄▄▄▄▄▄▄▄▄▄▄▄                               |    10
  40.894463ms | ▄▄▄▄▄▄▄▄                                    |     6
  42.991615ms | ▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄          |    25
  45.088767ms | ▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄ |    32
  47.185919ms | ▄▄▄▄▄▄▄▄▄▄▄▄▄                               |    10
  49.283071ms | ▄▄▄▄                                        |     3
  51.380223ms | ▄▄▄                                         |     2
  53.477375ms | ▄                                           |     1
  55.574527ms |                                             |     0
  57.671679ms | ▄▄▄▄▄                                       |     4
  59.768831ms | ▄                                           |     1
  61.865983ms |                                             |     0
  63.963135ms |                                             |     0
  66.060287ms |                                             |     0
  69.206015ms | ▄                                           |     1
  73.400319ms |                                             |     0
  77.594623ms | ▄                                           |     1
```

### hostnames provided directly
```
$ dnstrace -n 10 -c 10 --server 8.8.8.8 --recurse redsift.io
Using 1 hostnames
Benchmarking 8.8.8.8:53 via udp with 10 conncurrent requests

Total requests:	 100 of 100 (100.0%)
DNS success codes:     	100
Truncated responses:	0

DNS response codes:
       	NOERROR:       	100

Time taken for tests:  	 107.091332ms
Questions per second:  	 933.8

DNS timings, 100 datapoints
       	 min:  		 3.145728ms
       	 mean: 		 9.484369ms
       	 [+/-sd]:    5.527339ms
       	 max:  		 27.262975ms
	     p99:		 27.262975ms
	     p95:		 11.068671ms
	     p90:		 10.922943ms
	     p75:		 10.204351ms
	     p50:		 9.485759ms        

DNS distribution, 100 datapoints
    LATENCY   |                                             | COUNT
+-------------+---------------------------------------------+-------+
  3.276799ms  | ▄▄▄▄▄▄▄▄                                    |     2
  3.538943ms  | ▄▄▄▄▄▄▄▄▄▄▄▄                                |     3
  3.801087ms  | ▄▄▄▄▄▄▄▄▄▄▄▄                                |     3
  4.063231ms  | ▄▄▄▄▄▄▄▄▄▄▄▄                                |     3
  4.325375ms  | ▄▄▄▄▄▄▄▄                                    |     2
  4.587519ms  |                                             |     0
  4.849663ms  |                                             |     0
  5.111807ms  | ▄▄▄▄                                        |     1
  5.373951ms  | ▄▄▄▄                                        |     1
  5.636095ms  | ▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄                            |     4
  5.898239ms  | ▄▄▄▄▄▄▄▄▄▄▄▄                                |     3
  6.160383ms  | ▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄                        |     5
  6.422527ms  | ▄▄▄▄▄▄▄▄▄▄▄▄                                |     3
  6.684671ms  | ▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄                        |     5
  6.946815ms  | ▄▄▄▄▄▄▄▄                                    |     2
  7.208959ms  | ▄▄▄▄                                        |     1
  7.471103ms  | ▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄         |     9
  7.733247ms  | ▄▄▄▄▄▄▄▄                                    |     2
  7.995391ms  | ▄▄▄▄▄▄▄▄                                    |     2
  8.257535ms  | ▄▄▄▄▄▄▄▄▄▄▄▄                                |     3
  8.650751ms  | ▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄                        |     5
  9.175039ms  | ▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄ |    11
  9.699327ms  | ▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄                     |     6
  10.223615ms | ▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄                            |     4
  10.747903ms | ▄▄▄▄                                        |     1
  11.272191ms | ▄▄▄▄                                        |     1
  11.796479ms |                                             |     0
  12.320767ms |                                             |     0
  12.845055ms |                                             |     0
  13.369343ms |                                             |     0
  13.893631ms | ▄▄▄▄                                        |     1
  14.417919ms | ▄▄▄▄                                        |     1
  14.942207ms | ▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄                        |     5
  15.466495ms |                                             |     0
  15.990783ms | ▄▄▄▄                                        |     1
  16.515071ms |                                             |     0
  17.301503ms |                                             |     0
  18.350079ms |                                             |     0
  19.398655ms | ▄▄▄▄                                        |     1
  20.447231ms | ▄▄▄▄▄▄▄▄                                    |     2
  21.495807ms | ▄▄▄▄                                        |     1
  22.544383ms |                                             |     0
  23.592959ms |                                             |     0
  24.641535ms | ▄▄▄▄                                        |     1
  25.690111ms | ▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄                            |     4
  26.738687ms | ▄▄▄▄                                        |     1
```

### hostnames provided using file

```
$ dnstrace -n 10 -c 10 --server 8.8.8.8 --recurse @data/2-domains
Using 2 hostnames
Benchmarking 8.8.8.8:53 via udp with 10 concurrent requests

Total requests:	 200
DNS success codes:	200
Truncated responses:	0

DNS response codes:
	NOERROR:	200

Time taken for tests:	 266.985025ms
Questions per second:	 749.1

DNS timings, 200 datapoints
	 min:		 5.767168ms
	 mean:		 11.517952ms
	 [+/-sd]:	 5.128617ms
	 max:		 29.360127ms
	 p99:		 27.262975ms
	 p95:		 23.068671ms
	 p90:		 19.922943ms
	 p75:		 15.204351ms
	 p50:		 10.485759ms

DNS distribution, 200 datapoints
    LATENCY   |                                             | COUNT
+-------------+---------------------------------------------+-------+
  5.898239ms  | ▄▄▄▄▄                                       |     3
  6.160383ms  | ▄▄▄▄▄▄▄                                     |     4
  6.422527ms  | ▄▄▄▄▄▄▄▄▄▄▄                                 |     6
  6.684671ms  | ▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄ |    24
  6.946815ms  | ▄▄▄▄▄▄▄▄▄▄▄▄▄▄                              |     8
  7.208959ms  | ▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄                   |    14
  7.471103ms  | ▄▄▄▄▄                                       |     3
  7.733247ms  | ▄▄▄▄▄▄▄▄▄▄▄▄▄▄                              |     8
  7.995391ms  | ▄▄▄▄▄▄▄▄▄▄▄▄▄                               |     7
  8.257535ms  | ▄▄▄▄                                        |     2
  8.650751ms  | ▄▄▄▄▄▄▄▄▄                                   |     5
  9.175039ms  | ▄▄▄▄▄▄▄                                     |     4
  9.699327ms  | ▄▄▄▄▄▄▄▄▄▄▄                                 |     6
  10.223615ms | ▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄                        |    11
  10.747903ms | ▄▄▄▄▄▄▄                                     |     4
  11.272191ms | ▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄                          |    10
  11.796479ms | ▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄                          |    10
  12.320767ms | ▄▄▄▄▄▄▄▄▄▄▄                                 |     6
  12.845055ms | ▄▄▄▄▄▄▄▄▄                                   |     5
  13.369343ms | ▄▄                                          |     1
  13.893631ms | ▄▄▄▄▄▄▄▄▄                                   |     5
  14.417919ms | ▄▄▄▄▄                                       |     3
  14.942207ms | ▄▄▄▄▄▄▄▄▄▄▄                                 |     6
  15.466495ms | ▄▄▄▄▄▄▄▄▄▄▄                                 |     6
  15.990783ms | ▄▄▄▄▄▄▄▄▄▄▄▄▄▄                              |     8
  16.515071ms | ▄▄▄▄                                        |     2
  17.301503ms | ▄▄▄▄▄▄▄                                     |     4
  18.350079ms | ▄▄▄▄▄▄▄                                     |     4
  19.398655ms | ▄▄▄▄▄▄▄▄▄▄▄▄▄                               |     7
  20.447231ms | ▄▄                                          |     1
  21.495807ms | ▄▄▄▄                                        |     2
  22.544383ms | ▄▄                                          |     1
  23.592959ms | ▄▄▄▄▄▄▄                                     |     4
  24.641535ms | ▄▄▄▄▄                                       |     3
  25.690111ms |                                             |     0
  26.738687ms | ▄▄                                          |     1
  27.787263ms |                                             |     0
  28.835839ms | ▄▄▄▄                                        |     2
```

### using probability to randomize concurrent queries
```
$ dnstrace -c 10 --server 8.8.8.8  --recurse --probability 0.33  @data/alexa
Using 33575 hostnames
Benchmarking 8.8.8.8:53 via udp with 10 concurrent requests

Total requests:	 2713
Connection errors:	0
Read/Write errors:	35
DNS success codes:	2614
Truncated responses:    0

DNS response codes:
	NOERROR:	2614
	SERVFAIL:	15
	NXDOMAIN:	49

Time taken for tests:	 49.149400459s
Questions per second:	 55.2

DNS timings, 2678 datapoints
	 min:		 35.651584ms
	 mean:		 124.141922ms
	 [+/-sd]:	 230.61073ms
	 max:		 3.355443199s
	 p99:		 1.342177279s
	 p95:		 436.207615ms
	 p90:		 260.046847ms
	 p75:		 79.691775ms
	 p50:		 60.817407ms

DNS distribution, 2678 datapoints
    LATENCY    |                                             | COUNT
+--------------+---------------------------------------------+-------+
  36.700159ms  |                                             |     3
  38.797311ms  | ▄▄▄▄▄▄▄▄▄                                   |   121
  40.894463ms  | ▄▄                                          |    28
  42.991615ms  | ▄                                           |    12
  45.088767ms  |                                             |     6
  47.185919ms  |                                             |     5
  49.283071ms  |                                             |     2
  51.380223ms  | ▄▄▄▄▄▄▄                                     |   102
  53.477375ms  | ▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄ |   601
  55.574527ms  | ▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄                            |   229
  57.671679ms  | ▄▄▄▄▄▄▄                                     |    92
  59.768831ms  | ▄▄▄▄▄▄▄▄▄▄▄▄                                |   170
  61.865983ms  | ▄▄▄▄▄▄▄▄▄▄▄▄▄                               |   179
  63.963135ms  | ▄▄▄▄▄▄▄▄▄▄                                  |   141
  66.060287ms  | ▄▄▄▄▄▄▄                                     |    92
  69.206015ms  | ▄▄▄▄▄▄▄▄▄                                   |   127
  73.400319ms  | ▄▄▄▄▄▄                                      |    80
  77.594623ms  | ▄▄▄▄                                        |    60
  81.788927ms  | ▄▄▄▄                                        |    58
  85.983231ms  | ▄▄                                          |    25
  90.177535ms  | ▄                                           |    19
  94.371839ms  | ▄                                           |    17
  98.566143ms  | ▄▄                                          |    28
  102.760447ms | ▄▄                                          |    23
  106.954751ms | ▄                                           |    16
  111.149055ms |                                             |     6
  115.343359ms |                                             |     5
  119.537663ms | ▄                                           |     7
  123.731967ms |                                             |     6
  127.926271ms | ▄                                           |    12
  132.120575ms | ▄                                           |     8
  138.412031ms | ▄                                           |    18
  146.800639ms | ▄                                           |    18
  155.189247ms | ▄                                           |    11
  163.577855ms | ▄▄                                          |    23
  171.966463ms | ▄                                           |     7
  180.355071ms |                                             |     4
  188.743679ms |                                             |     6
  197.132287ms | ▄                                           |    10
  205.520895ms | ▄                                           |     9
  213.909503ms | ▄                                           |     7
  222.298111ms |                                             |     6
  230.686719ms |                                             |     6
  239.075327ms |                                             |     1
  247.463935ms |                                             |     3
  255.852543ms |                                             |     4
  264.241151ms |                                             |     2
  276.824063ms | ▄▄                                          |    23
  293.601279ms | ▄                                           |    16
  310.378495ms | ▄                                           |    20
  327.155711ms | ▄▄                                          |    28
  343.932927ms | ▄                                           |    12
  360.710143ms | ▄                                           |    10
  377.487359ms |                                             |     2
  394.264575ms | ▄                                           |     9
  411.041791ms |                                             |     6
  427.819007ms |                                             |     5
  444.596223ms |                                             |     5
  461.373439ms |                                             |     3
  478.150655ms | ▄                                           |     8
  494.927871ms |                                             |     3
  511.705087ms |                                             |     3
  528.482303ms |                                             |     2
  553.648127ms |                                             |     4
  587.202559ms | ▄                                           |    14
  620.756991ms | ▄                                           |     8
  654.311423ms |                                             |     3
  687.865855ms |                                             |     3
  721.420287ms | ▄                                           |     8
  754.974719ms |                                             |     5
  788.529151ms |                                             |     6
  822.083583ms |                                             |     1
  855.638015ms |                                             |     5
  889.192447ms |                                             |     5
  922.746879ms |                                             |     4
  956.301311ms |                                             |     1
  989.855743ms |                                             |     2
  1.023410175s |                                             |     0
  1.056964607s |                                             |     1
  1.107296255s |                                             |     2
  1.174405119s |                                             |     3
  1.241513983s |                                             |     1
  1.308622847s | ▄                                           |    10
  1.375731711s |                                             |     2
  1.442840575s |                                             |     0
  1.509949439s |                                             |     5
  1.577058303s |                                             |     6
  1.644167167s |                                             |     0
  1.711276031s |                                             |     0
  1.778384895s |                                             |     0
  1.845493759s |                                             |     0
  1.912602623s |                                             |     1
  1.979711487s |                                             |     0
  2.046820351s |                                             |     1
  2.113929215s |                                             |     0
  2.214592511s |                                             |     0
  2.348810239s |                                             |     2
  2.483027967s |                                             |     1
  2.617245695s |                                             |     3
  2.751463423s |                                             |     0
  2.885681151s |                                             |     0
  3.019898879s |                                             |     0
  3.154116607s |                                             |     0
  3.288334335s |                                             |     1
```
### EDNSOPT usage
```
$ dnstrace -n 10 -c 10  --recurse idnes.cz --server 127.0.0.1 --ednsopt=65518:fddddddd100000000000000000000001
Using 1 hostnames
Benchmarking 127.0.0.1:53 via udp with 10 concurrent requests

Total requests:	 100
DNS success codes:	100
Truncated responses:	0

DNS response codes:
	NOERROR:	100

Time taken for tests:	 282.592214ms
Questions per second:	 353.9

DNS timings, 100 datapoints
	 min:		 6.291456ms
	 mean:		 26.523729ms
	 [+/-sd]:	 48.818084ms
	 max:		 192.937983ms
	 p99:		 184.549375ms
	 p95:		 176.160767ms
	 p90:		 32.505855ms
	 p75:		 14.680063ms
	 p50:		 8.388607ms

DNS distribution, 100 datapoints
    LATENCY    |                                             | COUNT
+--------------+---------------------------------------------+-------+
  6.422527ms   | ▄▄▄▄▄▄▄▄▄▄▄▄▄                               |     5
  6.684671ms   | ▄▄▄▄▄▄▄▄▄▄▄▄▄                               |     5
  6.946815ms   | ▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄                     |     9
  7.208959ms   | ▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄ |    17
  7.471103ms   | ▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄                     |     9
  7.733247ms   | ▄▄▄▄▄▄▄▄                                    |     3
  7.995391ms   | ▄▄▄                                         |     1
  8.257535ms   | ▄▄▄                                         |     1
  8.650751ms   |                                             |     0
  9.175039ms   | ▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄                        |     8
  9.699327ms   | ▄▄▄                                         |     1
  10.223615ms  | ▄▄▄▄▄▄▄▄                                    |     3
  10.747903ms  |                                             |     0
  11.272191ms  |                                             |     0
  11.796479ms  | ▄▄▄                                         |     1
  12.320767ms  | ▄▄▄▄▄▄▄▄▄▄                                  |     4
  12.845055ms  | ▄▄▄▄▄▄▄▄                                    |     3
  13.369343ms  | ▄▄▄                                         |     1
  13.893631ms  | ▄▄▄▄▄▄▄▄                                    |     3
  14.417919ms  | ▄▄▄                                         |     1
  14.942207ms  |                                             |     0
  15.466495ms  | ▄▄▄                                         |     1
  15.990783ms  |                                             |     0
  16.515071ms  | ▄▄▄                                         |     1
  17.301503ms  | ▄▄▄▄▄▄▄▄▄▄▄▄▄                               |     5
  18.350079ms  | ▄▄▄▄▄▄▄▄                                    |     3
  19.398655ms  | ▄▄▄▄▄                                       |     2
  20.447231ms  |                                             |     0
  21.495807ms  | ▄▄▄                                         |     1
  22.544383ms  |                                             |     0
  23.592959ms  |                                             |     0
  24.641535ms  |                                             |     0
  25.690111ms  |                                             |     0
  26.738687ms  | ▄▄▄                                         |     1
  27.787263ms  |                                             |     0
  28.835839ms  |                                             |     0
  29.884415ms  |                                             |     0
  30.932991ms  |                                             |     0
  31.981567ms  | ▄▄▄                                         |     1
  33.030143ms  |                                             |     0
  34.603007ms  |                                             |     0
  36.700159ms  |                                             |     0
  38.797311ms  |                                             |     0
  40.894463ms  |                                             |     0
  42.991615ms  |                                             |     0
  45.088767ms  |                                             |     0
  47.185919ms  |                                             |     0
  49.283071ms  |                                             |     0
  51.380223ms  |                                             |     0
  53.477375ms  |                                             |     0
  55.574527ms  |                                             |     0
  57.671679ms  |                                             |     0
  59.768831ms  |                                             |     0
  61.865983ms  |                                             |     0
  63.963135ms  |                                             |     0
  66.060287ms  |                                             |     0
  69.206015ms  |                                             |     0
  73.400319ms  |                                             |     0
  77.594623ms  |                                             |     0
  81.788927ms  |                                             |     0
  85.983231ms  |                                             |     0
  90.177535ms  |                                             |     0
  94.371839ms  |                                             |     0
  98.566143ms  |                                             |     0
  102.760447ms |                                             |     0
  106.954751ms |                                             |     0
  111.149055ms |                                             |     0
  115.343359ms |                                             |     0
  119.537663ms |                                             |     0
  123.731967ms |                                             |     0
  127.926271ms |                                             |     0
  132.120575ms |                                             |     0
  138.412031ms |                                             |     0
  146.800639ms |                                             |     0
  155.189247ms | ▄▄▄▄▄                                       |     2
  163.577855ms | ▄▄▄                                         |     1
  171.966463ms | ▄▄▄▄▄▄▄▄                                    |     3
  180.355071ms | ▄▄▄▄▄▄▄▄                                    |     3
  188.743679ms | ▄▄▄                                         |     1
```
### DoT
```
$ dnstrace -n 10 -c 10  --dot --recurse  --server 1.1.1.1:853 idnes.cz
Using 1 hostnames
Benchmarking 1.1.1.1:853 via tcp with 10 concurrent requests

Total requests:	 100
DNS success codes:	100
Truncated responses:	0

DNS response codes:
	NOERROR:	100

Time taken for tests:	 366.625655ms
Questions per second:	 272.8

DNS timings, 100 datapoints
	 min:		 6.815744ms
	 mean:		 12.059934ms
	 [+/-sd]:	 5.532558ms
	 max:		 44.040191ms
	 p99:		 39.845887ms
	 p95:		 20.971519ms
	 p90:		 16.777215ms
	 p75:		 13.631487ms
	 p50:		 10.485759ms

DNS distribution, 100 datapoints
    LATENCY   |                                             | COUNT
+-------------+---------------------------------------------+-------+
  6.946815ms  | ▄▄▄▄▄▄▄                                     |     2
  7.208959ms  |                                             |     0
  7.471103ms  | ▄▄▄▄▄▄▄                                     |     2
  7.733247ms  | ▄▄▄▄▄▄▄                                     |     2
  7.995391ms  | ▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄                      |     6
  8.257535ms  | ▄▄▄▄                                        |     1
  8.650751ms  | ▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄        |    10
  9.175039ms  | ▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄               |     8
  9.699327ms  | ▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄               |     8
  10.223615ms | ▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄ |    12
  10.747903ms | ▄▄▄▄▄▄▄▄▄▄▄                                 |     3
  11.272191ms | ▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄        |    10
  11.796479ms | ▄▄▄▄▄▄▄▄▄▄▄                                 |     3
  12.320767ms | ▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄                          |     5
  12.845055ms | ▄▄▄▄                                        |     1
  13.369343ms | ▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄                      |     6
  13.893631ms | ▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄                      |     6
  14.417919ms | ▄▄▄▄                                        |     1
  14.942207ms | ▄▄▄▄▄▄▄                                     |     2
  15.466495ms | ▄▄▄▄                                        |     1
  15.990783ms |                                             |     0
  16.515071ms | ▄▄▄▄                                        |     1
  17.301503ms | ▄▄▄▄▄▄▄▄▄▄▄                                 |     3
  18.350079ms |                                             |     0
  19.398655ms | ▄▄▄▄                                        |     1
  20.447231ms | ▄▄▄▄                                        |     1
  21.495807ms |                                             |     0
  22.544383ms |                                             |     0
  23.592959ms | ▄▄▄▄                                        |     1
  24.641535ms | ▄▄▄▄                                        |     1
  25.690111ms |                                             |     0
  26.738687ms |                                             |     0
  27.787263ms |                                             |     0
  28.835839ms | ▄▄▄▄                                        |     1
  29.884415ms |                                             |     0
  30.932991ms |                                             |     0
  31.981567ms |                                             |     0
  33.030143ms |                                             |     0
  34.603007ms |                                             |     0
  36.700159ms |                                             |     0
  38.797311ms | ▄▄▄▄                                        |     1
  40.894463ms |                                             |     0
  42.991615ms | ▄▄▄▄                                        |     1
```
