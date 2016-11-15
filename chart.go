// Package chartjs simplifies making chartjs.org plots in go.
package chartjs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image/color"
)

// RGBA amends image/color.RGBA to have a MarshalJSON that meets the expectations of chartjs.
type RGBA color.RGBA

// MarshalJSON satisfies the json.Marshaler interface.
func (c RGBA) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"rgba(%d, %d, %d, %.3f)\"", c.R, c.G, c.B, float64(c.A)/255)), nil
}

var chartTypes = [...]string{
	"line",
	"bar",
	"bubble",
}

type chartType int

func (c chartType) MarshalJSON() ([]byte, error) {
	return []byte(`"` + chartTypes[c] + `"`), nil
}

const (
	// Line is a "line" plot
	Line chartType = iota
	// Bar is a "bar" plot
	Bar
	// Bubble is a "bubble" plot
	Bubble
)

// FloatFormat determines how many decimal places are sent in the JSON.
var FloatFormat = "%.2f"

// Values dictates the interface of data to be plotted.
type Values interface {
	// X-axis values. If only these are specified then it must be a Bar plot.
	Xs() []float64
	// Optional Y values.
	Ys() []float64
	// Rs are used for chartType `Bubble`
	Rs() []float64
}

func marshalValuesJSON(v Values) ([]byte, error) {
	xs, ys, rs := v.Xs(), v.Ys(), v.Rs()
	if len(xs) == 0 {
		if len(rs) != 0 {
			return nil, fmt.Errorf("chart: bad format of Values data")
		}
		xs = ys[:len(ys)]
		ys = nil
	}
	buf := bytes.NewBuffer(make([]byte, 0, 8*len(xs)))
	buf.WriteRune('[')
	if len(rs) > 0 {
		if len(xs) != len(ys) || len(xs) != len(rs) {
			return nil, fmt.Errorf("chart: bad format of Values. All axes must be of the same length")
		}
		for i, x := range xs {
			if i > 0 {
				buf.WriteRune(',')
			}
			y, r := ys[i], rs[i]
			_, err := buf.WriteString(fmt.Sprintf(("{\"x\":" + FloatFormat + ",\"y\":" + FloatFormat + ",\"r\":" + FloatFormat + "}"), x, y, r))
			if err != nil {
				return nil, err
			}
		}
	} else if len(ys) > 0 {
		if len(xs) != len(ys) {
			return nil, fmt.Errorf("chart: bad format of Values. X and Y must be of the same length")
		}
		for i, x := range xs {
			if i > 0 {
				buf.WriteRune(',')
			}
			y := ys[i]
			_, err := buf.WriteString(fmt.Sprintf(("{\"x\":" + FloatFormat + ",\"y\":" + FloatFormat + "}"), x, y))
			if err != nil {
				return nil, err
			}
		}

	} else {
		for i, x := range xs {
			if i > 0 {
				buf.WriteRune(',')
			}
			_, err := buf.WriteString(fmt.Sprintf(FloatFormat, x))
			if err != nil {
				return nil, err
			}
		}
	}

	buf.WriteRune(']')
	return buf.Bytes(), nil
}

// Dataset wraps the "dataset" JSON
type Dataset struct {
	Data            Values    `json:"-"`
	Type            chartType `json:"type,omitempty"`
	BackgroundColor *RGBA     `json:"backgroundColor,omitempty"`
	BorderColor     *RGBA     `json:"borderColor,omitempty"`

	// Label indicates the name of the dataset to be shown in the legend.
	Label       string  `json:"label,omitempty"`
	Fill        Bool    `json:"fill,omitempty"`
	LineTension float64 `json:"lineTension,omitempty"`
	PointRadius float64 `json:"pointRadius,omitempty"`
	ShowLine    Bool    `json:"showLine,omitempty"`
	SpanGaps    Bool    `json:"spanGaps,omitempty"`
}

// MarshalJSON implements json.Marshaler interface.
func (d Dataset) MarshalJSON() ([]byte, error) {
	o, err := marshalValuesJSON(d.Data)
	// avoid recursion by creating an alias.
	type alias Dataset
	buf, err := json.Marshal(alias(d))
	if err != nil {
		return nil, err
	}
	// replace '}' with ',' to continue struct
	if len(buf) > 0 {
		buf[len(buf)-1] = ','
	}
	buf = append(buf, []byte(`"data":`)...)
	buf = append(buf, o...)
	buf = append(buf, '}')
	return buf, nil
}

// Data wraps the "data" JSON
type Data struct {
	Datasets []Dataset `json:"datasets"`
	Labels   []string  `json:"labels"`
}

type axisType int

var axisTypes = []string{
	"category",
	"linear",
	"logarithmic",
	"time",
	"radialLinear",
}

const (
	// Category is a categorical axis (this is the default),
	// used for bar plots.
	Category axisType = iota
	// Linear axis should be use for scatter plots.
	Linear
	// Log axis
	Log
	// Time axis
	Time
	// Radial axis
	Radial
)

func (t axisType) MarshalJSON() ([]byte, error) {
	return []byte("\"" + axisTypes[t] + "\""), nil
}

type axisPosition int

const (
	// Bottom puts the axis on the bottom (used for Y-axis)
	Bottom axisPosition = iota + 1
	// Top puts the axis on the bottom (used for Y-axis)
	Top
	// Left puts the axis on the bottom (used for X-axis)
	Left
	// Right puts the axis on the bottom (used for X-axis)
	Right
)

var axisPositions = []string{
	"",
	"bottom",
	"top",
	"left",
	"right",
}

func (p axisPosition) MarshalJSON() ([]byte, error) {
	return []byte("\"" + axisPositions[p] + "\""), nil
}

// Bool is a convenience typedef for pointer to bool so that we can differentiate between unset
// and false.
type Bool *bool

var t = true
var f = false

var True = Bool(&t)
var False = Bool(&f)

// Axis corresponds to 'scale' in chart.js lingo.
type Axis struct {
	Type      axisType     `json:"type"`
	Position  axisPosition `json:"position,omitempty"`
	Label     string       `json:"label,omitempty"`
	ID        string       `json:"id,omitempty"`
	GridLines Bool         `json:"gridLine,omitempty"`
	Stacked   Bool         `json:"stacked,omitempty"`

	// need to differentiate between false and empty to use a pointer
	Display Bool `json:"display,omitempty"`
}

// Axes holds the X and Y axies. Its simpler to use Chart.AddXAxis, Chart.AddYAxis.
type Axes struct {
	XAxes []Axis `json:"xAxes,omitempty"`
	YAxes []Axis `json:"yAxes,omitempty"`
}

// AddX adds a X-Axis.
func (a *Axes) AddX(x Axis) {
	a.XAxes = append(a.XAxes, x)
}

// AddY adds a Y-Axis.
func (a *Axes) AddY(y Axis) {
	a.YAxes = append(a.YAxes, y)
}

// Option wraps the chartjs "option"
type Option struct {
	Responsive          Bool `json:"responsive,omitempty"`
	MaintainAspectRatio Bool `json:"maintainAspectRatio,omitempty"`
}

// Options wraps the chartjs "options"
type Options struct {
	Option
	Scales Axes `json:"scales,omitempty"`
}

type Chart struct {
	Type    chartType `json:"type"`
	Label   string    `json:"label,omitempty"`
	Data    Data      `json:"data,omitempty"`
	Options Options   `json:"options,omitempty"`
}

// AddDataset adds a dataset to the chart.
func (c *Chart) AddDataset(d Dataset) {
	c.Data.Datasets = append(c.Data.Datasets, d)
}

// AddXAxis adds an x-axis to the chart.
func (c *Chart) AddXAxis(x Axis) {
	c.Options.Scales.XAxes = append(c.Options.Scales.XAxes, x)
}

// AddYAxis adds an y-axis to the chart.
func (c *Chart) AddYAxis(y Axis) {
	c.Options.Scales.YAxes = append(c.Options.Scales.YAxes, y)
}
