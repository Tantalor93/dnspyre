# DNStrace

[![CircleCI](https://circleci.com/gh/redsift/dnstrace.svg?style=shield)](https://circleci.com/gh/redsift/dnstrace)

[![Go Report Card](https://goreportcard.com/badge/github.com/redsift/dnstrace)](https://goreportcard.com/report/github.com/redsift/dnstrace)

Command-line DNS benchmark tool built to stress test and measure the performance of DNS servers with commodity hardware. This tool typically consumers ~30kB per concurrent connection and can do ~3000 QPS per Xeon E5 core.

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

## Installation

### Option 1 - Docker

This tool is available in a prebuilt image that comes in at ~8MB

`docker run redsift/dnstrace --help`

## Bash/ZSH Shell Completion

`./dnstrace --completion-script-bash` and `./dnstrace --completion-script-zsh` will create shell completion scripts.

## C10K and the like

As you approach thousands of concurrent connections on OS-X, you may run into connection errors due to insufficient file handles or threads. This is likely due to process limits so remember to adjust these limits if you intent to increase concurrency levels beyond 1000.

Note that using `sudo ulimit` will create a root shell, adjusts its limits, and then exit causing no real effect. Instead use `launchctl` first on OS-X.

```
$ sudo launchctl limit maxfiles 1000000 1000000
$ ulimit -n 12288
```

## Example

```
$ dnstrace -n 10 -c 10 --server 8.8.8.8 --recurse redsift.io

Total requests:		100
DNS success codes:	100

DNS Codes
	NOERROR:	100

Time taken for tests:	 87.184678ms

DNS timings, 100 datapoints
	 min:		 3.014656ms
	 mean:		 7.5196ms
	 [+/-sd]:	 3.284911ms
	 max:		 26.214399ms

Distribution
    LATENCY   |                                             | COUNT
+-------------+---------------------------------------------+-------+
  3.080191ms  | ▄▄▄▄▄                                       |     1
  3.211263ms  |                                             |     0
  3.342335ms  | ▄▄▄▄▄                                       |     1
  3.473407ms  | ▄▄▄▄▄                                       |     1
  3.604479ms  | ▄▄▄▄▄▄▄▄▄▄                                  |     2
  3.735551ms  | ▄▄▄▄▄                                       |     1
  3.866623ms  | ▄▄▄▄▄▄▄▄▄▄                                  |     2
  3.997695ms  | ▄▄▄▄▄                                       |     1
  4.128767ms  | ▄▄▄▄▄                                       |     1
  4.325375ms  | ▄▄▄▄▄▄▄▄▄▄▄▄▄▄                              |     3
  4.587519ms  | ▄▄▄▄▄▄▄▄▄▄                                  |     2
  4.849663ms  | ▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄                         |     4
  5.111807ms  | ▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄           |     7
  5.373951ms  | ▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄                         |     4
  5.636095ms  | ▄▄▄▄▄▄▄▄▄▄▄▄▄▄                              |     3
  5.898239ms  | ▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄                         |     4
  6.160383ms  | ▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄                    |     5
  6.422527ms  | ▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄                    |     5
  6.684671ms  | ▄▄▄▄▄▄▄▄▄▄                                  |     2
  6.946815ms  | ▄▄▄▄▄▄▄▄▄▄                                  |     2
  7.208959ms  | ▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄                         |     4
  7.471103ms  | ▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄                         |     4
  7.733247ms  | ▄▄▄▄▄▄▄▄▄▄▄▄▄▄                              |     3
  7.995391ms  | ▄▄▄▄▄▄▄▄▄▄▄▄▄▄                              |     3
  8.257535ms  | ▄▄▄▄▄▄▄▄▄▄                                  |     2
  8.650751ms  | ▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄ |     9
  9.175039ms  | ▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄                    |     5
  9.699327ms  | ▄▄▄▄▄▄▄▄▄▄▄▄▄▄                              |     3
  10.223615ms | ▄▄▄▄▄▄▄▄▄▄                                  |     2
  10.747903ms | ▄▄▄▄▄▄▄▄▄▄                                  |     2
  11.272191ms | ▄▄▄▄▄▄▄▄▄▄▄▄▄▄                              |     3
  11.796479ms | ▄▄▄▄▄                                       |     1
  12.320767ms | ▄▄▄▄▄                                       |     1
  12.845055ms |                                             |     0
  13.369343ms | ▄▄▄▄▄                                       |     1
  13.893631ms | ▄▄▄▄▄                                       |     1
  14.417919ms | ▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄                         |     4
  14.942207ms |                                             |     0
  15.466495ms |                                             |     0
  15.990783ms |                                             |     0
  16.515071ms |                                             |     0
  17.301503ms |                                             |     0
  18.350079ms |                                             |     0
  19.398655ms |                                             |     0
  20.447231ms |                                             |     0
  21.495807ms |                                             |     0
  22.544383ms |                                             |     0
  23.592959ms |                                             |     0
  24.641535ms |                                             |     0
  25.690111ms | ▄▄▄▄▄                                       |     1

```