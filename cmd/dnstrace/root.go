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

	pServer = pApp.Flag("server", "DNS server IP:port to test. IPv6 is also supported, for example '[fddd:dddd::]:53'.").Short('s').Default("127.0.0.1").String()
	pType   = pApp.Flag("type", "Query type.").Short('t').Default("A").Enum(getSupportedDNSTypes()...)

	pCount       = pApp.Flag("number", "Number of queries to issue. Note that the total number of queries issued = number*concurrency*len(queries).").Short('n').Default("1").Int64()
	pConcurrency = pApp.Flag("concurrency", "Number of concurrent queries to issue.").Short('c').Default("1").Uint32()
	pRate        = pApp.Flag("rate-limit", "Apply a global questions / second rate limit.").Short('l').Default("0").Int()
	pQperConn    = pApp.Flag("query-per-conn", "Queries on a connection before creating a new one. 0: unlimited").Default("0").Int64()

	pExpect = pApp.Flag("expect", "Expect a specific response.").Short('e').Strings()

	pRecurse     = pApp.Flag("recurse", "Allow DNS recursion.").Short('r').Default("false").Bool()
	pProbability = pApp.Flag("probability", "Each hostname from file will be used with provided probability in %. Value 1 and above means that each hostname from file will be used by each concurrent benchmark goroutine. Useful for randomizing queries across benchmark goroutines.").Default("1").Float64()
	pUDPSize     = pApp.Flag("edns0", "Enable EDNS0 with specified size.").Default("0").Uint16()
	pEdnsOpt     = pApp.Flag("ednsopt", "code[:value], Specify EDNS option with code point code and optionally payload of value as a hexadecimal string. code must be arbitrary numeric value.").Default("").String()
	pTCP         = pApp.Flag("tcp", "Use TCP fot DNS requests.").Default("false").Bool()
	pDOT         = pApp.Flag("dot", "Use DoT for DNS requests.").Default("false").Bool()

	pWriteTimeout = pApp.Flag("write", "DNS write timeout.").Default("1s").Duration()
	pReadTimeout  = pApp.Flag("read", "DNS read timeout.").Default(dnsTimeout.String()).Duration()

	pRCodes = pApp.Flag("codes", "Enable counting DNS return codes.").Default("true").Bool()

	pHistMin     = pApp.Flag("min", "Minimum value for timing histogram.").Default((time.Microsecond * 400).String()).Duration()
	pHistMax     = pApp.Flag("max", "Maximum value for histogram.").Default(dnsTimeout.String()).Duration()
	pHistPre     = pApp.Flag("precision", "Significant figure for histogram precision.").Default("1").PlaceHolder("[1-5]").Int()
	pHistDisplay = pApp.Flag("distribution", "Display distribution histogram of timings to stdout.").Default("true").Bool()
	pCsv         = pApp.Flag("csv", "Export distribution to CSV.").Default("").PlaceHolder("/path/to/file.csv").String()

	pIOErrors = pApp.Flag("io-errors", "Log I/O errors to stderr.").Default("false").Bool()

	pSilent = pApp.Flag("silent", "Disable stdout.").Default("false").Bool()
	pColor  = pApp.Flag("color", "ANSI Color output.").Default("true").Bool()

	pPlotHist = pApp.Flag("hist", "Plot histogram. Based on suffix format is chosen (.png, .svg, .pdf)").Default("").PlaceHolder("/path/to/plot.png").String()
	pPlotBox  = pApp.Flag("box", "Plot box plot. Based on suffix format is chosen (.png, .svg, .pdf)").Default("").PlaceHolder("/path/to/plot.png").String()

	pQueries = pApp.Arg("queries", "Queries to issue. Can be file referenced using @<file-path>, for example @data/2-domains").Required().Strings()
)

var (
	count     int64
	cerror    int64
	ecount    int64
	success   int64
	matched   int64
	mismatch  int64
	truncated int64
)

const (
	fileNoBuffer = 9 // app itself needs about 9 for libs
)

// Execute starts main logic of command
func Execute() {
	pApp.Version(Version)
	kingpin.MustParse(pApp.Parse(os.Args[1:]))

	// process args
	color.NoColor = !*pColor

	lim, err := sysutil.RlimitStack()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Cannot check limit of number of files. Skipping check. Please make sure it is sufficient manually.", err)
	} else {
		needed := uint64(*pConcurrency) + uint64(fileNoBuffer)
		if lim < needed {
			fmt.Fprintf(os.Stderr, "Current process limit for number of files is %d and insufficient for level of requested concurrency.", lim)
			os.Exit(1)
		}
	}

	var csv *os.File
	if *pCsv != "" {
		f, err := os.Create(*pCsv)
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
		<-sigsInt
		fmt.Fprintf(os.Stderr, "\nCancelling benchmark ^C, again to terminate now.\n")
		cancel()
		<-sigsInt
		os.Exit(1)
	}()

	// get going
	rand.Seed(time.Now().UnixNano())

	start := time.Now()
	res := do(ctx)
	end := time.Now()

	printReport(end.Sub(start), res, csv)
}

func getSupportedDNSTypes() []string {
	keys := make([]string, 0, len(dns.StringToType))
	for k := range dns.StringToType {
		keys = append(keys, k)
	}
	return keys
}
