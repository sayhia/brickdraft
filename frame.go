package main

import "math"

// Every scene is drawn in the unit square, but the user's grid can be 4:3 today
// and 1:1 after one click on the size slider. Rather than rewrite each recipe,
// the painter maps the unit square onto a fixed frame and gives the subject a
// margin, so a change of aspect reads as a different crop of the same picture.
const sceneDesignAspect = 1.15

// frame is the artwork box, in screen-normalised units.
type frame struct {
	x0, y0, x1, y1 float64
}

// frameFor fits the design box into the current grid.
//
// The box always spans the full height and is centred horizontally, so the
// subject keeps its proportions instead of being squashed to the grid's aspect
// - the obvious failure mode of drawing straight in screen space.
//
// sceneDesignAspect is the box's nominal width/height. It is only a hint for
// composition (a slightly wide stage reads better than a square one); the
// mapping itself is exact at any grid, which is what keeps a 3:4 picture from
// being drawn as a letterbox.
func frameFor(aspect float64) frame {
	if aspect <= 0 {
		aspect = 1
	}
	// Screen -> design units per cell is uniform in both axes (1/h), so the
	// window is exactly `aspect` wide for a height of 1: a 4:3 grid covers
	// 1.333 x 1 units of the design box, a 3:4 grid 0.75 x 1. It is centred,
	// so a wide grid is a crop of the box's middle and a tall grid is a narrow
	// slice of it - never a stretched copy, which is the one thing a crop must
	// never do.
	w := aspect
	if w < sceneDesignAspect/4 {
		w = sceneDesignAspect / 4 // keeps the box from vanishing on odd grids
	}
	m := (1 - w) / 2
	return frame{x0: m, y0: 0, x1: 1 - m, y1: 1}
}

// toScene maps a screen point into the design box.
func (f frame) toScene(u, v float64) (float64, float64) {
	w, h := f.x1-f.x0, f.y1-f.y0
	if w <= 0 || h <= 0 {
		return u, v
	}
	return (u - f.x0) / w, (v - f.y0) / h
}

// aspectOf returns width/height for a grid.
func aspectOf(w, h float64) float64 {
	if h <= 0 {
		return 1
	}
	return w / h
}

// aspectLabel names a crop the way a photo app would.
func aspectLabel(a float64) string {
	switch {
	case math.Abs(a-1) < 0.02:
		return "1:1"
	case math.Abs(a-4.0/3.0) < 0.05:
		return "4:3"
	case math.Abs(a-3.0/4.0) < 0.05:
		return "3:4"
	case math.Abs(a-0.8) < 0.05:
		return "5:4"
	}
	return "自由"
}
