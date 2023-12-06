package cmd

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testDatapoints = []Datapoint{
	{Start: time.Now(), Duration: 100},
	{Start: time.Now().Add(time.Second), Duration: 200},
	{Start: time.Now().Add(2 * time.Second), Duration: 300},
	{Start: time.Now().Add(3 * time.Second), Duration: 100},
	{Start: time.Now().Add(4 * time.Second), Duration: 150},
	{Start: time.Now().Add(5 * time.Second), Duration: 200},
	{Start: time.Now().Add(6 * time.Second), Duration: 200},
	{Start: time.Now().Add(7 * time.Second), Duration: 300},
	{Start: time.Now().Add(8 * time.Second), Duration: 350},
	{Start: time.Now().Add(9 * time.Second), Duration: 100},
	{Start: time.Now().Add(10 * time.Second), Duration: 200},
}

var testErrorDatapoints = []ErrorDatapoint{
	{Start: time.Now()},
	{Start: time.Now().Add(2 * time.Second)},
	{Start: time.Now().Add(3 * time.Second)},
	{Start: time.Now().Add(4 * time.Second)},
	{Start: time.Now().Add(5 * time.Second)},
	{Start: time.Now().Add(6 * time.Second)},
	{Start: time.Now().Add(7 * time.Second)},
}

var testRcodes = map[int]int64{
	0: 8,
	2: 1,
	3: 2,
}

func Test_plotHistogramLatency(t *testing.T) {
	dir := t.TempDir()

	file := dir + "/histogram-latency.png"
	plotHistogramLatency(file, testDatapoints)

	expected, err := os.ReadFile("test-histogram-latency.png")
	require.NoError(t, err)

	actual, err := os.ReadFile(file)
	require.NoError(t, err)

	assert.Equal(t, expected, actual, "generated histogram latency plot does not equal to expected 'test-histogram-latency.png'")
}

func Test_plotBoxPlotLatency(t *testing.T) {
	dir := t.TempDir()

	file := dir + "/boxplot-latency.png"
	plotBoxPlotLatency(file, "127.0.0.1", testDatapoints)

	expected, err := os.ReadFile("test-boxplot-latency.png")
	require.NoError(t, err)

	actual, err := os.ReadFile(file)
	require.NoError(t, err)

	assert.Equal(t, expected, actual, "generated boxplot latency plot does not equal to expected 'test-boxplot-latency.png'")
}

func Test_plotResponses(t *testing.T) {
	dir := t.TempDir()

	file := dir + "/responses-barchart.png"
	plotResponses(file, testRcodes)

	expected, err := os.ReadFile("test-responses-barchart.png")
	require.NoError(t, err)

	actual, err := os.ReadFile(file)
	require.NoError(t, err)

	assert.Equal(t, expected, actual, "generated responses plot does not equal to expected 'test-responses-barchart.png'")
}

func Test_plotLineThroughput(t *testing.T) {
	dir := t.TempDir()

	file := dir + "/throughput-lineplot.png"
	plotLineThroughput(file, testDatapoints)

	expected, err := os.ReadFile("test-throughput-lineplot.png")
	require.NoError(t, err)

	actual, err := os.ReadFile(file)
	require.NoError(t, err)

	assert.Equal(t, expected, actual, "generated line throughput plot does not equal to expected 'test-throughput-lineplot.png'")
}

func Test_plotLineLatencies(t *testing.T) {
	dir := t.TempDir()

	file := dir + "/latency-lineplot.png"
	plotLineLatencies(file, testDatapoints)

	expected, err := os.ReadFile("test-latency-lineplot.png")
	require.NoError(t, err)

	actual, err := os.ReadFile(file)
	require.NoError(t, err)

	assert.Equal(t, expected, actual, "generated line latencies plot does not equal to expected 'test-latency-lineplot.png'")
}

func Test_plotErrorRate(t *testing.T) {
	dir := t.TempDir()

	file := dir + "/errorrate-lineplot.png"
	plotErrorRate(file, testErrorDatapoints)

	expected, err := os.ReadFile("test-errorrate-lineplot.png")
	require.NoError(t, err)

	actual, err := os.ReadFile(file)
	require.NoError(t, err)

	assert.Equal(t, expected, actual, "generated error rate plot does not equal to expected 'test-errorrate-lineplot.png")
}
