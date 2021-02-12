
# DNStrace
forked https://github.com/redsift/dnstrace 

Command-line DNS benchmark tool built to stress test and measure the performance of DNS servers with commodity hardware.
This tool typically consumes ~30kB per concurrent connection and can maintain ~30,000 QPS per modern core if your server, OS and network allows you to reach suitable levels of concurrency.

## Installation 
```
go get github.com/tantalor93/dnstrace
```
will install the binary in your $GOPATH/bin

## Build
### for linux
```
env GOOS=linux GARCH=amd64 go build
```
### for macos
```
env GOOS=darwin go build
```

## Usage

```
$ dnstrace --help
usage: dnstrace [<flags>] <queries>...

A high QPS DNS benchmark.

Flags:
      --help                   Show context-sensitive help (also try --help-long and --help-man).
  -s, --server="127.0.0.1"     DNS server IP:port to test.
  -t, --type=A                 Query type.
  -n, --number=1               Number of queries to issue. Note that the total number of queries issued = number*concurrency*len(queries).
  -c, --concurrency=1          Number of concurrent queries to issue.
  -l, --rate-limit=0           Apply a global questions / second rate limit.
      --query-per-conn=0       Queries on a connection before creating a new one. 0: unlimited
  -e, --expect=EXPECT ...      Expect a specific response.
  -r, --recurse                Allow DNS recursion.
      --probability=100        Each hostname from file will be used with provided probability in %. Value 100 and above means that each hostname from file will be used by each concurrent benchmark
                               goroutine. Useful for randomizing queries accross benchmark goroutines.
      --edns0=0                Enable EDNS0 with specified size.
      --tcp                    Use TCP fot DNS requests.
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
hostnames provided directly
```
$ dnstrace -n 10 -c 10 --server 8.8.8.8 --recurse redsift.io
Using 1 hostnames

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

```
$ dnstrace -n 10 -c 10 --server 8.8.8.8 --recurse @data/2-domains
Using 2 hostnames

Benchmarking 8.8.8.8:53 via udp with 10 concurrent requests


Total requests:	 200
DNS success codes:	200

DNS response codes
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