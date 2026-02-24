package dnsbench

import (
	"time"
)

const (
	// DefaultEdns0BufferSize default EDNS0 buffer size according to the http://www.dnsflagday.net/2020/
	DefaultEdns0BufferSize = 1232

	// DefaultRequestLogPath is a default path to the file, where the requests will be logged.
	DefaultRequestLogPath = "requests.log"

	// DefaultPlotFormat is a default format for plots.
	DefaultPlotFormat = "svg"

	// DefaultRequestTimeout is a default request timeout.
	DefaultRequestTimeout = 5 * time.Second

	// DefaultConnectTimeout is a default connect timeout.
	DefaultConnectTimeout = time.Second

	// DefaultReadTimeout is a default read timeout.
	DefaultReadTimeout = 3 * time.Second

	// DefaultWriteTimeout is a default write timeout.
	DefaultWriteTimeout = time.Second

	// DefaultProbability is a default probability.
	DefaultProbability = 1.0

	// DefaultConcurrency is a default concurrency.
	DefaultConcurrency = 1

	// DefaultCount is a default count when duration or count is not specified.
	DefaultCount = 1

	// DefaultQueryType is a default type for queries if no other is specified.
	DefaultQueryType = "A"

	// DefaultHistPrecision is a default precision for histogram.
	DefaultHistPrecision = 1
)
