# DNStrace

Commandline DNS benchmark tool.

## Usage

```
$ dnstrace --help

usage: dnstrace [<flags>] <queries>...

A DTrace enabled DNS benchmark.

Flags:
      --help                Show context-sensitive help (also try --help-long
                            and --help-man).
      --dtrace              Enable DTrace probes
      --silent              Disable stdout
      --recurse             Allow DNS recursion
      --server="127.0.0.1"  Server IP and port to query
      --type=TXT            Query type
      --expect=EXPECT       Expect a specific response
  -n, --number=1            Number of queries to issue. Note that the total
                            number of queries issued =
                            number*concurrency*len(queries)
  -c, --concurrency=1       Number of concurrent queries to issue
      --edns0=0             Enable EDNS0 with specified size
      --write=1s            DNS write timeout
      --read=4s             DNS read timeout
      --min=100000          Minimum value for timing histogram in nanoseconds
      --max=4000000000      Maximum value for histogram in nanoseconds
      --precision=1         Significant figure for histogram precision
      --distribution        Display distribution histogram of timings
      --codes               Enable counting DNS return codes
      --io-errors           Log I/O errors to stderr
      --color               Color output
      --version             Show application version.

Args:
  <queries>  Queries to issue.
```

### Bash/ZSH Shell Completion

`./dnstrace --completion-script-bash` and `./dnstrace --completion-script-zsh` will completion scripts.

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