package dnstrace

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alecthomas/kingpin"
	"github.com/fatih/color"
	"github.com/miekg/dns"
	"github.com/tantalor93/dnstrace/internal/sysutil"
)

var (
	// Version is set during release of project during build process
	Version = "development"

	author = "Ondrej Benkovsky <obenky@gmail.com>, Rahul Powar <rahul@redsift.io>"
)

var (
	pApp = kingpin.New("dnstrace", "A high QPS DNS benchmark.").Author(author)

	userInput = BenchmarkInput{
		server: *pApp.Flag("server", "DNS server IP:port to test. IPv6 is also supported, for example '[fddd:dddd::]:53'. "+
			"Also DoH servers are supported such as `https://1.1.1.1/dns-query`, when such server is provided, the benchmark automatically switches to the use of DoH. "+
			"Note that path on which DoH server handles requests (like `/dns-query`) has to be provided as well.").Short('s').Default("127.0.0.1").String(),

		types: 	*pApp.Flag("type", "Query type. Repeatable flag. If multiple query types are specified then each query will be duplicated for each type.").Short('t').Default("A").Enums(getSupportedDNSTypes()...),
		count:  *pApp.Flag("number", "How many times the provided queries are repeated. Note that the total number of queries issued = types*number*concurrency*len(queries).").Short('n').Default("1").Int64(),
		concurrency: *pApp.Flag("concurrency", "Number of concurrent queries to issue.").Short('c').Default("1").Uint32(),
		rate: *pApp.Flag("rate-limit", "Apply a global questions / second rate limit.").Short('l').Default("0").Int(),
		qperConn: *pApp.Flag("query-per-conn", "Queries on a connection before creating a new one. 0: unlimited").Default("0").Int64(),
		expect: *pApp.Flag("expect", "Expect a specific response.").Short('e').Strings(),
		recurse: *pApp.Flag("recurse", "Allow DNS recursion.").Short('r').Default("false").Bool(),
		probability: *pApp.Flag("probability", "Each hostname from file will be used with provided probability. Value 1 and above means that each hostname from file will be used by each concurrent benchmark goroutine. Useful for randomizing queries across benchmark goroutines.").Default("1").Float64(),
		udpSize: *pApp.Flag("edns0", "Enable EDNS0 with specified size.").Default("0").Uint16(),
		ednsOpt: *pApp.Flag("ednsopt", "code[:value], Specify EDNS option with code point code and optionally payload of value as a hexadecimal string. code must be arbitrary numeric value.").Default("").String(),
		tcp: *pApp.Flag("tcp", "Use TCP fot DNS requests.").Default("false").Bool(),
		dot: *pApp.Flag("dot", "Use DoT for DNS requests.").Default("false").Bool(),
		writeTimeout: *pApp.Flag("write", "DNS write timeout.").Default("1s").Duration(),
		readTimeout:  *pApp.Flag("read", "DNS read timeout.").Default(dnsTimeout.String()).Duration(),
		rcodes: *pApp.Flag("codes", "Enable counting DNS return codes.").Default("true").Bool(),
		histMin: *pApp.Flag("min", "Minimum value for timing histogram.").Default((time.Microsecond * 400).String()).Duration(),
		histMax: *pApp.Flag("max", "Maximum value for histogram.").Default(dnsTimeout.String()).Duration(),
		histPre: *pApp.Flag("precision", "Significant figure for histogram precision.").Default("1").PlaceHolder("[1-5]").Int(),
		histDisplay: *pApp.Flag("distribution", "Display distribution histogram of timings to stdout.").Default("true").Bool(),
		csv:  *pApp.Flag("csv", "Export distribution to CSV.").Default("").PlaceHolder("/path/to/file.csv").String(),
		ioerrors: *pApp.Flag("io-errors", "Log I/O errors to stderr.").Default("false").Bool(),
		silent: *pApp.Flag("silent", "Disable stdout.").Default("false").Bool(),
		color: *pApp.Flag("color", "ANSI Color output.").Default("true").Bool(),
		plotDir:  *pApp.Flag("plot", "Plot benchmark results and export them to directory.").Default("").PlaceHolder("/path/to/folder").String(),
		plotFormat: *pApp.Flag("plotf", "Format of graphs. Supported formats png, svg, pdf.").Default("png").Enum("png", "svg", "pdf"),
		dohMethod: *pApp.Flag("doh-method", "HTTP method to use for DoH requests").Default("post").Enum("get", "post"),
		dohProtocol: *pApp.Flag("doh-protocol", "HTTP protocol to use for DoH requests").Default("1.1").Enum("1.1", "2"),
	}
)

const (
	fileNoBuffer = 9 // app itself needs about 9 for libs
)

// Execute starts main logic of command
func Execute() {
	pApp.Version(Version)
	kingpin.MustParse(pApp.Parse(os.Args[1:]))

	// process args
	color.NoColor = !userInput.color

	lim, err := sysutil.RlimitStack()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Cannot check limit of number of files. Skipping check. Please make sure it is sufficient manually.", err)
	} else {
		needed := uint64(userInput.concurrency) + uint64(fileNoBuffer)
		if lim < needed {
			fmt.Fprintf(os.Stderr, "Current process limit for number of files is %d and insufficient for level of requested concurrency.", lim)
			os.Exit(1)
		}
	}

	var csv *os.File
	if userInput.csv != "" {
		f, err := os.Create(userInput.csv)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Failed to create file for CSV export.", err)
			os.Exit(1)
		}

		csv = f
	}

	sigsInt := make(chan os.Signal, 8)
	signal.Notify(sigsInt, syscall.SIGINT)

	defer close(sigsInt)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		_, ok := <-sigsInt
		if !ok {
			// standard exit based on channel close
			return
		}
		fmt.Fprintf(os.Stderr, "\nCancelling benchmark ^C, again to terminate now.\n")
		cancel()
		<-sigsInt
		os.Exit(1)
	}()

	// get going
	rand.Seed(time.Now().UnixNano())

	start := time.Now()
	res := do(ctx, userInput)
	end := time.Now()

	printReport(end.Sub(start), res, csv, userInput)
}

func getSupportedDNSTypes() []string {
	keys := make([]string, 0, len(dns.StringToType))
	for k := range dns.StringToType {
		keys = append(keys, k)
	}
	return keys
}
