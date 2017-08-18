# DNStrace

[![Go Report Card](https://goreportcard.com/badge/github.com/redsift/dnstrace)](https://goreportcard.com/report/github.com/redsift/dnstrace)
[![Release](https://img.shields.io/github/release/redsift/dnstrace/all.svg)](https://github.com/redsift/dnstrace/releases)
[![CircleCI](https://circleci.com/gh/redsift/dnstrace.svg?style=shield)](https://circleci.com/gh/redsift/dnstrace)
[![Docker Image](https://images.microbadger.com/badges/image/redsift/dnstrace.svg)](https://microbadger.com/images/redsift/dnstrace)

Command-line DNS benchmark tool built to stress test and measure the performance of DNS servers with commodity hardware.
This tool typically consumes ~30kB per concurrent connection and can maintain ~30,000 QPS per modern core if your server, OS and network allows you to reach suitable levels of concurrency.

DNStrace bypasses OS resolvers and is provided as a Docker packaged prebuilt static binary.
Basic latency measurement, result checking and histograms are supported.
Currently, only `A`, `AAAA` and `TXT` questions are supported.

## Usage

```
$ dnstrace --help

usage: dnstrace [<flags>] <queries>...

A DNS benchmark.

Flags:
      --help                   Show context-sensitive help (also try --help-long
                               and --help-man).
  -s, --server="127.0.0.1"     DNS server IP:port to test.
  -t, --type=A                 Query type.
  -n, --number=1               Number of queries to issue. Note that the total
                               number of queries issued =
                               number*concurrency*len(queries).
  -c, --concurrency=1          Number of concurrent queries to issue.
  -l, --rate-limit=0           Apply a global questions / second rate limit.
  -e, --expect=EXPECT ...      Expect a specific response.
  -r, --recurse                Allow DNS recursion.
      --edns0=0                Enable EDNS0 with specified size.
      --tcp                    Use TCP fot DNS requests.
      --write=1s               DNS write timeout.
      --read=4s                DNS read timeout.
      --codes                  Enable counting DNS return codes.
      --min=400µs              Minimum value for timing histogram.
      --max=4s                 Maximum value for histogram.
      --precision=[1-5]        Significant figure for histogram precision.
      --distribution           Display distribution histogram of timings to
                               stdout.
      --csv=/path/to/file.csv  Export distribution to CSV.
      --io-errors              Log I/O errors to stderr.
      --silent                 Disable stdout.
      --color                  ANSI Color output.
      --version                Show application version.

Args:
  <queries>  Queries to issue.
```

## Warning

While `dnstrace` is helpful for testing round trip latency via public networks,
the code was primarily created to provide an [apachebench](https://en.wikipedia.org/wiki/ApacheBench)
style tool for testing your own infrastructure.

It is thus very easy to create significant DNS load with non default settings.
**Do not do this to public DNS services**. You will most likely flag your IP.

## Installation

### go get

`go get github.com/redsift/dnstrace` will install the binary in your `$GOPATH/bin`.
On OS-X, the native binary will outperform the Docker container below running under HyperKit significantly e.g. 30% more throughput, 30% lower latency and a 4x decrease in timing spread

### Docker

[![Latest](https://images.microbadger.com/badges/version/redsift/dnstrace.svg)](https://microbadger.com/images/redsift/dnstrace)

This tool is available in a prebuilt image.

`docker run redsift/dnstrace --help`

If your local test setup lets you reach 50k QPS and above, you can expect the docker networking to add ~2% overhead to throughput and ~8% to mean latency (tested on Linux Docker 1.12.3).
If this is significant for your purposes you may wish to run with `--net=host`

## Bash/ZSH Shell Completion

`./dnstrace --completion-script-bash` and `./dnstrace --completion-script-zsh` will create shell completion scripts.

e.g.
```
$ eval "$(./dnstrace --completion-script-zsh)"

$ ./dnstrace --concurrency
  --codes         --distribution  --io-errors     --precision     --server        --version
  --color         --edns0         --max           --rate-limit    --silent        --write
  --concurrency   --expect        --min           --read          --tcp
  --csv           --help          --number        --recurse       --type

```

## C10K and the like

As you approach thousands of concurrent connections on OS-X, you may run into connection errors due to insufficient file handles or threads. This is likely due to process limits so remember to adjust these limits if you intent to increase concurrency levels beyond 1000.

Note that using `sudo ulimit` will create a root shell, adjusts its limits, and then exit causing no real effect. Instead use `launchctl` first on OS-X.

```
$ sudo launchctl limit maxfiles 1000000 1000000
$ ulimit -n 12288
```

## Progress

For long runs, the user can send a SIGHUP via `kill -1 pid` to get the current progress counts.

## Example

```
$ docker run redsift/dnstrace -n 10 -c 10 --server 8.8.8.8 --recurse redsift.io

Benchmarking 8.8.8.8:53 via udp with 10 conncurrent requests


Total requests:	 100 of 100 (100.0%)
DNS success codes:     	100

DNS response codes
       	NOERROR:       	100

Time taken for tests:  	 107.091332ms
Questions per second:  	 933.8

DNS timings, 100 datapoints
       	 min:  		 3.145728ms
       	 mean: 		 9.484369ms
       	 [+/-sd]:    5.527339ms
       	 max:  		 27.262975ms

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