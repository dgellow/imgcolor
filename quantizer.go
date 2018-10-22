package main

import (
	"image"
	"math"
	"sort"
)

// A color's RGBA method returns values in the range [0, 65535]
const maxColor = 65535

type entry struct {
	r     int
	g     int
	b     int
	count int
}

type quantizer struct {
	scale     float64
	image     image.Image
	shift     uint
	bins      int
	mean      int
	histogram []entry
}

func NewQuantizer(img image.Image, shift uint, scale float64) quantizer {
	if scale <= 0 {
		scale = 1
	}

	bins := (maxColor >> shift) + 1

	return quantizer{
		scale: scale,
		image: img,
		// Shifting 65535 by e.g: 14 reduces it to the range [0, 3].
		shift:     shift,
		bins:      bins,
		mean:      (1 << (shift - 1)) / 2,
		histogram: []entry{},
	}
}

func (q *quantizer) Quantize() {
	bounds := q.image.Bounds()

	var multiDimHistogram = make([][][]int, q.bins)
	// initialize
	for r := range multiDimHistogram {
		multiDimHistogram[r] = make([][]int, q.bins)
		for g := range multiDimHistogram[r] {
			multiDimHistogram[r][g] = make([]int, q.bins)
		}
	}

	// multi dimension histogram, makes it easy to count bin combinations
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := q.image.At(x, y).RGBA()
			if a > 0 {
				multiDimHistogram[r>>q.shift][g>>q.shift][b>>q.shift]++
			}
		}
	}

	// flatten multi dimensional histogram in a smaller uni dimensional one that
	// only contains found bin combinations.
	for r := range multiDimHistogram {
		for g := range multiDimHistogram[r] {
			for b := range multiDimHistogram[r][g] {
				count := multiDimHistogram[r][g][b]
				if count > 0 {
					q.histogram = append(q.histogram, entry{r, g, b, count})
				}
			}
		}
	}
}

func (q quantizer) Less(i int, j int) bool {
	e, f := q.histogram[i], q.histogram[j]
	return e.count < f.count
}

func (q quantizer) Len() int {
	return len(q.histogram)
}

func (q quantizer) Swap(i int, j int) {
	e, f := q.histogram[i], q.histogram[j]
	q.histogram[i], q.histogram[j] = f, e
}

// MostFrequent tries to get up to n most frequent entries in the histogram. A
// result length of n isn't guaranteed as the histogram can have less than n entries.
func (q quantizer) MostFrequent(n int) []entry {
	if n <= 0 {
		n = 1
	}
	if l := len(q.histogram); l < n {
		n = l
	}

	sort.Sort(sort.Reverse(q))

	res := make([]entry, n)
	for i := 0; i < n; i++ {
		res[i] = q.applyScale(q.histogram[i])
	}
	return res
}

func (q quantizer) applyScale(e entry) entry {
	return entry{
		r:     int(math.Round(float64(e.r<<q.shift+q.mean) / float64(maxColor) * q.scale)),
		g:     int(math.Round(float64(e.g<<q.shift+q.mean) / float64(maxColor) * q.scale)),
		b:     int(math.Round(float64(e.b<<q.shift+q.mean) / float64(maxColor) * q.scale)),
		count: e.count,
	}
}
