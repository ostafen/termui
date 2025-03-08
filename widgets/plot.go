// Copyright 2017 Zack Guo <zack.y.guo@gmail.com>. All rights reserved.
// Use of this source code is governed by a MIT license that can
// be found in the LICENSE file.

package widgets

import (
	"fmt"
	"image"
	"math"

	. "github.com/gizak/termui/v3"
)

// Plot has two modes: line(default) and scatter.
// Plot also has two marker types: braille(default) and dot.
// A single braille character is a 2x4 grid of dots, so using braille
// gives 2x X resolution and 4x Y resolution over dot mode.
type Plot struct {
	Block

	Data       [][]float64
	DataLabels []string
	MaxVal     float64

	LineColors []Color
	AxesColor  Color // TODO
	ShowAxes   bool

	Marker          PlotMarker
	DotMarkerRune   rune
	PlotType        PlotType
	HorizontalScale int
	DrawDirection   DrawDirection // TODO
}

const (
	xAxisLabelsHeight = 1
	yAxisLabelsWidth  = 4
	xAxisLabelsGap    = 2
	yAxisLabelsGap    = 1
)

type PlotType uint

const (
	LineChart PlotType = iota
	ScatterPlot
)

type PlotMarker uint

const (
	MarkerBraille PlotMarker = iota
	MarkerDot
)

type DrawDirection uint

const (
	DrawLeft DrawDirection = iota
	DrawRight
)

func NewPlot() *Plot {
	return &Plot{
		Block:           *NewBlock(),
		LineColors:      Theme.Plot.Lines,
		AxesColor:       Theme.Plot.Axes,
		Marker:          MarkerBraille,
		DotMarkerRune:   DOT,
		Data:            [][]float64{},
		HorizontalScale: 1,
		DrawDirection:   DrawRight,
		ShowAxes:        true,
		PlotType:        LineChart,
	}
}

func (self *Plot) renderBraille(buf *Buffer, drawArea image.Rectangle, maxVal float64) {
	canvas := NewCanvas()
	canvas.Rectangle = drawArea

	switch self.PlotType {
	case ScatterPlot:
		for i, line := range self.Data {
			for j, val := range line {
				height := int((val / maxVal) * float64(drawArea.Dy()-1))
				canvas.SetPoint(
					image.Pt(
						(drawArea.Min.X+(j*self.HorizontalScale))*2,
						(drawArea.Max.Y-height-1)*4,
					),
					SelectColor(self.LineColors, i),
				)
			}
		}
	case LineChart:
		r := self.GetValueRange()

		for i, line := range self.Data {
			previousHeight := int(r.Scale(line[0]) * float64(drawArea.Dy()-1))
			for j, val := range line[1:] {
				height := int(r.Scale(val) * float64(drawArea.Dy()-1))
				canvas.SetLine(
					image.Pt(
						(drawArea.Min.X+(j*self.HorizontalScale))*2,
						(drawArea.Max.Y-previousHeight-1)*4,
					),
					image.Pt(
						(drawArea.Min.X+((j+1)*self.HorizontalScale))*2,
						(drawArea.Max.Y-height-1)*4,
					),
					SelectColor(self.LineColors, i),
				)
				previousHeight = height
			}
		}
	}

	canvas.Draw(buf)
}

func (self *Plot) renderDot(buf *Buffer, drawArea image.Rectangle, maxVal float64) {
	switch self.PlotType {
	case ScatterPlot:
		for i, line := range self.Data {
			for j, val := range line {
				height := int((val / maxVal) * float64(drawArea.Dy()-1))
				point := image.Pt(drawArea.Min.X+(j*self.HorizontalScale), drawArea.Max.Y-1-height)
				if point.In(drawArea) {
					buf.SetCell(
						NewCell(self.DotMarkerRune, NewStyle(SelectColor(self.LineColors, i))),
						point,
					)
				}
			}
		}
	case LineChart:
		for i, line := range self.Data {
			for j := 0; j < len(line) && j*self.HorizontalScale < drawArea.Dx(); j++ {
				val := line[j]
				height := int((val / maxVal) * float64(drawArea.Dy()-1))
				buf.SetCell(
					NewCell(self.DotMarkerRune, NewStyle(SelectColor(self.LineColors, i))),
					image.Pt(drawArea.Min.X+(j*self.HorizontalScale), drawArea.Max.Y-1-height),
				)
			}
		}
	}
}

func (self *Plot) plotAxes(buf *Buffer, maxVal float64) {
	// draw origin cell
	buf.SetCell(
		NewCell(BOTTOM_LEFT, NewStyle(ColorWhite)),
		image.Pt(self.Inner.Min.X+yAxisLabelsWidth, self.Inner.Max.Y-xAxisLabelsHeight-1),
	)
	// draw x axis line
	for i := yAxisLabelsWidth + 1; i < self.Inner.Dx(); i++ {
		buf.SetCell(
			NewCell(HORIZONTAL_DASH, NewStyle(ColorWhite)),
			image.Pt(i+self.Inner.Min.X, self.Inner.Max.Y-xAxisLabelsHeight-1),
		)
	}
	// draw y axis line
	for i := 0; i < self.Inner.Dy()-xAxisLabelsHeight-1; i++ {
		buf.SetCell(
			NewCell(VERTICAL_DASH, NewStyle(ColorWhite)),
			image.Pt(self.Inner.Min.X+yAxisLabelsWidth, i+self.Inner.Min.Y),
		)
	}
	// draw x axis labels
	// draw 0
	buf.SetString(
		"0",
		NewStyle(ColorWhite),
		image.Pt(self.Inner.Min.X+yAxisLabelsWidth, self.Inner.Max.Y-1),
	)

	if len(self.DataLabels) == 0 {
		// draw rest
		for x := self.Inner.Min.X + yAxisLabelsWidth + (xAxisLabelsGap)*self.HorizontalScale + 1; x < self.Inner.Max.X-1; {
			label := fmt.Sprintf(
				"%d",
				(x-(self.Inner.Min.X+yAxisLabelsWidth)-1)/(self.HorizontalScale)+1,
			)

			buf.SetString(
				label,
				NewStyle(ColorWhite),
				image.Pt(x, self.Inner.Max.Y-1),
			)
			x += (len(label) + xAxisLabelsGap) * self.HorizontalScale
		}
	} else {
		xAxisLabelsGap := ((self.Inner.Max.X - self.Inner.Min.X) - yAxisLabelsWidth) / (len(self.DataLabels) - 1)

		x := self.Inner.Min.X + yAxisLabelsWidth
		for _, label := range self.DataLabels {
			buf.SetString(
				label,
				NewStyle(ColorWhite),
				image.Pt(x, self.Inner.Max.Y-1),
			)

			labelLen := len(label)

			x += xAxisLabelsGap
			if x+labelLen >= self.Inner.Max.X {
				x = self.Inner.Max.X - labelLen
			}
		}
	}

	yTicks := self.yTicks()
	// draw y axis labels
	//verticalScale := maxVal / float64(self.Inner.Dy()-xAxisLabelsHeight-1)
	_yAxisLabelsGap := (self.Inner.Dy() - xAxisLabelsHeight) / (len(yTicks) - 1)
	for i := range yTicks {
		height := self.Inner.Max.Y - (i * (_yAxisLabelsGap)) - 2
		if height <= 0 {
			height = 1
		}

		buf.SetString(
			fmt.Sprintf("%.2f", yTicks[i]), //float64(i)*verticalScale*(yAxisLabelsGap+1)),
			NewStyle(ColorWhite),
			image.Pt(self.Inner.Min.X, height),
		)
	}
}

func (self *Plot) Draw(buf *Buffer) {
	self.Block.Draw(buf)

	maxVal := self.MaxVal
	if maxVal == 0 {
		maxVal, _ = GetMaxFloat64From2dSlice(self.Data)
	}

	if self.ShowAxes {
		self.plotAxes(buf, maxVal)
	}

	drawArea := self.Inner
	if self.ShowAxes {
		drawArea = image.Rect(
			self.Inner.Min.X+yAxisLabelsWidth+1, self.Inner.Min.Y,
			self.Inner.Max.X, self.Inner.Max.Y-xAxisLabelsHeight-1,
		)
	}

	switch self.Marker {
	case MarkerBraille:
		self.renderBraille(buf, drawArea, maxVal)
	case MarkerDot:
		self.renderDot(buf, drawArea, maxVal)
	}
}

func (self *Plot) yAxisLabelWidth() int {
	return 0
}

const MaxYTicks = 10 // TODO: make this configurable

func (self *Plot) yTicks() []float64 {
	r := self.GetValueRange()
	if r.IsConstant() {
		return []float64{
			math.Inf(1),
			r.Min,
			math.Inf(-1),
		}
	}

	ticks := make([]float64, MaxYTicks)
	gap := (r.Max - r.Min) / float64(len(ticks)-1)
	for i := range ticks {
		ticks[i] = r.Min + float64(i)*gap
	}
	return ticks
}

func (self *Plot) GetValueRange() ValueRange {
	if len(self.Data) == 0 {
		return ValueRange{}
	}

	max, min := self.Data[0][0], self.Data[0][0]
	for _, slice := range self.Data {
		for _, val := range slice {
			if val > max {
				max = val
			} else if val < min {
				min = val
			}
		}
	}

	return ValueRange{
		Min: min,
		Max: max,
	}
}

type ValueRange struct {
	Min float64
	Max float64
}

func (r *ValueRange) Scale(x float64) float64 {
	if r.IsConstant() {
		return 0.5
	}
	return (x - r.Min) / (r.Max - r.Min)
}

func (r *ValueRange) IsConstant() bool {
	return r.Max == r.Min
}
