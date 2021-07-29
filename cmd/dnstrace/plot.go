package dnstrace

import (
	"fmt"
	"os"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

func plotHistogramLatency(file string, times []datapoint) {
	var values plotter.Values
	for _, v := range times {
		values = append(values, v.duration)
	}
	p := plot.New()
	p.Title.Text = "Latencies distribution"

	hist, err := plotter.NewHist(values, 16)
	if err != nil {
		panic(err)
	}
	p.X.Label.Text = "latencies (ms)"
	p.Y.Label.Text = "count"
	p.Add(hist)

	if err := p.Save(6*vg.Inch, 6*vg.Inch, file); err != nil {
		fmt.Fprintln(os.Stderr, "Failed to save plot.", err)
	}
}

func plotBoxPlotLatency(file, server string, times []datapoint) {
	var values plotter.Values
	for _, v := range times {
		values = append(values, v.duration)
	}
	p := plot.New()
	p.Title.Text = "Latencies distribution"
	p.Y.Label.Text = "latencies (ms)"
	p.NominalX(server)

	boxplot, err := plotter.NewBoxPlot(vg.Length(120), 0, values)
	if err != nil {
		panic(err)
	}
	p.Add(boxplot)

	if err := p.Save(6*vg.Inch, 6*vg.Inch, file); err != nil {
		fmt.Fprintln(os.Stderr, "Failed to save plot.", err)
	}
}

func plotLineLatency(file string, times []datapoint) {
	var values plotter.XYs
	for i, v := range times {
		values = append(values, plotter.XY{X: float64(i), Y: v.duration})
	}
	p := plot.New()
	p.Title.Text = "Latencies progression during benchmark test"
	p.X.Label.Text = "number of requests since the start"
	p.Y.Label.Text = "latencies (ms)"

	l, err := plotter.NewLine(values)
	if err != nil {
		panic(err)
	}
	p.Add(l)

	if err := p.Save(6*vg.Inch, 6*vg.Inch, file); err != nil {
		fmt.Fprintln(os.Stderr, "Failed to save plot.", err)
	}
}
