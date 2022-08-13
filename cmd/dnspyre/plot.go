package dnspyre

import (
	"fmt"
	"image/color"
	"os"
	"sort"

	"github.com/miekg/dns"
	"go-hep.org/x/hep/hplot"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
)

func plotHistogramLatency(file string, times []Datapoint) {
	var values plotter.Values
	for _, v := range times {
		values = append(values, v.Duration)
	}
	p := plot.New()
	p.Title.Text = "Latencies distribution"

	hist, err := plotter.NewHist(values, 16)
	if err != nil {
		panic(err)
	}
	p.X.Label.Text = "Latencies (ms)"
	p.X.Tick.Marker = hplot.Ticks{N: 10, Format: "%.0f"}
	p.Y.Label.Text = "Number of requests"
	p.Y.Tick.Marker = hplot.Ticks{N: 10, Format: "%.0f"}
	hist.FillColor = color.RGBA{R: 175, G: 238, B: 238}
	p.Add(hist)

	if err := p.Save(6*vg.Inch, 6*vg.Inch, file); err != nil {
		fmt.Fprintln(os.Stderr, "Failed to save plot.", err)
	}
}

func plotBoxPlotLatency(file, server string, times []Datapoint) {
	var values plotter.Values
	for _, v := range times {
		values = append(values, v.Duration)
	}
	p := plot.New()
	p.Title.Text = "Latencies distribution"
	p.Y.Label.Text = "Latencies (ms)"
	p.Y.Tick.Marker = hplot.Ticks{N: 10, Format: "%.0f"}
	p.NominalX(server)

	boxplot, err := plotter.NewBoxPlot(vg.Length(120), 0, values)
	if err != nil {
		panic(err)
	}
	boxplot.FillColor = color.RGBA{R: 127, G: 188, B: 165, A: 1}
	p.Add(boxplot)

	if err := p.Save(6*vg.Inch, 6*vg.Inch, file); err != nil {
		fmt.Fprintln(os.Stderr, "Failed to save plot.", err)
	}
}

func plotResponses(file string, rcodes map[int]int64) {
	sortedKeys := make([]int, 0)
	for k := range rcodes {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Ints(sortedKeys)

	colors := []color.Color{
		color.RGBA{R: 122, G: 195, B: 106},
		color.RGBA{R: 241, G: 90, B: 96},
		color.RGBA{R: 90, G: 155, B: 212},
		color.RGBA{R: 250, G: 167, B: 91},
		color.RGBA{R: 158, G: 103, B: 171},
		color.RGBA{R: 206, G: 112, B: 88},
		color.RGBA{R: 215, G: 127, B: 180},
	}
	colors = append(colors, plotutil.DarkColors...)

	p := plot.New()
	p.Title.Text = "Response code distribution"
	p.NominalX("Response codes")

	width := vg.Points(40)

	c := 0
	off := -vg.Length(len(rcodes)/2) * width
	fmt.Println(sortedKeys)
	for _, v := range sortedKeys {
		fmt.Println(v)
		bar, err := plotter.NewBarChart(plotter.Values{float64(rcodes[v])}, width)
		if err != nil {
			panic(err)
		}
		p.Legend.Add(dns.RcodeToString[v], bar)
		bar.Color = colors[c%len(colors)]
		bar.Offset = off
		p.Add(bar)
		c++
		off += width
	}

	p.Y.Label.Text = "Number of requests"
	p.Y.Tick.Marker = hplot.Ticks{N: 10, Format: "%.0f"}
	p.Legend.Top = true

	if err := p.Save(6*vg.Inch, 6*vg.Inch, file); err != nil {
		fmt.Fprintln(os.Stderr, "Failed to save plot.", err)
	}
}

func plotLineThroughput(file string, times []Datapoint) {
	var values plotter.XYs
	m := make(map[int64]int64)

	if len(times) != 0 {
		first := times[0].Start.Unix()

		for _, v := range times {
			unix := v.Start.Unix() - first
			if _, ok := m[unix]; !ok {
				m[unix] = 0
			}
			m[unix]++
		}
	}

	for k, v := range m {
		values = append(values, plotter.XY{X: float64(k), Y: float64(v)})
	}

	sort.SliceStable(values, func(i, j int) bool {
		return values[i].X < values[j].X
	})

	p := plot.New()
	p.Title.Text = "Throughput per second"
	p.X.Label.Text = "Time of test (s)"
	p.X.Tick.Marker = hplot.Ticks{N: 10, Format: "%.0f"}
	p.Y.Label.Text = "Number of requests (per sec)"
	p.Y.Tick.Marker = hplot.Ticks{N: 10, Format: "%.0f"}

	l, err := plotter.NewLine(values)
	l.Color = color.RGBA{R: 255, A: 255}
	if err != nil {
		panic(err)
	}
	p.Add(l)

	if err := p.Save(6*vg.Inch, 6*vg.Inch, file); err != nil {
		fmt.Fprintln(os.Stderr, "Failed to save plot.", err)
	}
}
