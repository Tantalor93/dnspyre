package reporter

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tantalor93/dnspyre/v3/pkg/dnsbench"
	"gonum.org/v1/plot/plotter"
)

var testStart = time.Now()

var testDatapoints = []dnsbench.Datapoint{
	{Start: testStart, Duration: 100 * time.Millisecond},
	{Start: testStart.Add(time.Second), Duration: 200 * time.Millisecond},
	{Start: testStart.Add(2 * time.Second), Duration: 300 * time.Millisecond},
	{Start: testStart.Add(3 * time.Second), Duration: 100 * time.Millisecond},
	{Start: testStart.Add(4 * time.Second), Duration: 150 * time.Millisecond},
	{Start: testStart.Add(5 * time.Second), Duration: 200 * time.Millisecond},
	{Start: testStart.Add(6 * time.Second), Duration: 200 * time.Millisecond},
	{Start: testStart.Add(7 * time.Second), Duration: 300 * time.Millisecond},
	{Start: testStart.Add(8 * time.Second), Duration: 350 * time.Millisecond},
	{Start: testStart.Add(9 * time.Second), Duration: 100 * time.Millisecond},
	{Start: testStart.Add(10 * time.Second), Duration: 200 * time.Millisecond},
}

var testErrorDatapoints = []dnsbench.ErrorDatapoint{
	{Start: testStart.Add(2 * time.Second)},
	{Start: testStart.Add(3 * time.Second)},
	{Start: testStart.Add(4 * time.Second)},
	{Start: testStart.Add(5 * time.Second)},
	{Start: testStart.Add(6 * time.Second)},
	{Start: testStart.Add(7 * time.Second)},
}

var testRcodes = map[int]int64{
	0: 8,
	2: 1,
	3: 2,
}

func Test_plotHistogramLatency(t *testing.T) {
	dir := t.TempDir()

	file := dir + "/histogram-latency.svg"
	plotHistogramLatency(file, testDatapoints)

	expected, err := os.ReadFile("testdata/test-histogram-latency.svg")
	require.NoError(t, err)

	actual, err := os.ReadFile(file)
	require.NoError(t, err)

	assert.Equal(t, expected, actual, "generated histogram latency plot does not equal to expected 'test-histogram-latency.png'")
}

func Test_plotBoxPlotLatency(t *testing.T) {
	dir := t.TempDir()

	file := dir + "/boxplot-latency.svg"
	plotBoxPlotLatency(file, "127.0.0.1", testDatapoints)

	expected, err := os.ReadFile("testdata/test-boxplot-latency.svg")
	require.NoError(t, err)

	actual, err := os.ReadFile(file)
	require.NoError(t, err)

	assert.Equal(t, expected, actual, "generated boxplot latency plot does not equal to expected 'test-boxplot-latency.png'")
}

func Test_plotResponses(t *testing.T) {
	dir := t.TempDir()

	file := dir + "/responses-barchart.svg"
	plotResponses(file, testRcodes)

	expected, err := os.ReadFile("testdata/test-responses-barchart.svg")
	require.NoError(t, err)

	actual, err := os.ReadFile(file)
	require.NoError(t, err)

	assert.Equal(t, expected, actual, "generated responses plot does not equal to expected 'test-responses-barchart.png'")
}

func Test_plotLineThroughput(t *testing.T) {
	dir := t.TempDir()

	file := dir + "/throughput-lineplot.svg"
	plotLineThroughput(file, testStart, testDatapoints)

	expected, err := os.ReadFile("testdata/test-throughput-lineplot.svg")
	require.NoError(t, err)

	actual, err := os.ReadFile(file)
	require.NoError(t, err)

	assert.Equal(t, expected, actual, "generated line throughput plot does not equal to expected 'test-throughput-lineplot.png'")
}

func Test_plotLineLatencies(t *testing.T) {
	dir := t.TempDir()

	file := dir + "/latency-lineplot.svg"
	plotLineLatencies(file, testStart, testDatapoints)

	expected, err := os.ReadFile("testdata/test-latency-lineplot.svg")
	require.NoError(t, err)

	actual, err := os.ReadFile(file)
	require.NoError(t, err)

	assert.Equal(t, expected, actual, "generated line latencies plot does not equal to expected 'test-latency-lineplot.png'")
}

func Test_plotErrorRate(t *testing.T) {
	dir := t.TempDir()

	file := dir + "/errorrate-lineplot.svg"
	plotErrorRate(file, testStart, testErrorDatapoints)

	expected, err := os.ReadFile("testdata/test-errorrate-lineplot.svg")
	require.NoError(t, err)

	actual, err := os.ReadFile(file)
	require.NoError(t, err)

	assert.Equal(t, expected, actual, "generated error rate plot does not equal to expected 'test-errorrate-lineplot.png")
}

func Test_numBins(t *testing.T) {
	tests := []struct {
		name   string
		values plotter.Values
		want   int
	}{
		{
			name:   "small dataset",
			values: dataset(25),
			want:   5,
		},
		{
			name:   "medium dataset",
			values: dataset(500),
			want:   15,
		},
		{
			name:   "large dataset",
			values: dataset(2000),
			want:   11,
		},
		{
			name:   "single item dataset",
			values: dataset(1),
			want:   1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, numBins(tt.values))
		})
	}
}

// dataset generates uniformorly distributed dataset.
func dataset(length int) plotter.Values {
	values := make(plotter.Values, length)
	for i := range values {
		values[i] = float64(i)
	}
	return values
}
