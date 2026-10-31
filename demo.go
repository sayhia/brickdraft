package main

import (
	"image"
	"image/color"
	"math"
)

// demoScale is how many pixels the built-in scenes occupy per brick cell. The
// artwork is sampled once per cell and expanded by this factor, so downsampling
// it back to the grid is a pure 1:1 copy.
const demoScale = 16

// rampStop is one colour stop of a vertical gradient.
type rampStop struct {
	at float64
	c  [3]float64
}

// rampAt interpolates a gradient at t in [0,1].
func rampAt(stops []rampStop, t float64) [3]float64 {
	if t <= stops[0].at {
		return stops[0].c
	}
	for i := 1; i < len(stops); i++ {
		if t < stops[i].at {
			u := (t - stops[i-1].at) / (stops[i].at - stops[i-1].at)
			return mix3(stops[i-1].c, stops[i].c, u)
		}
	}
	return stops[len(stops)-1].c
}

// paintScene samples shade once per grid cell, at the cell centre, and blows
// each sample up into a scale x scale block.
//
// Hard edges matter here: an anti-aliased edge would survive the downsample as
// a half-and-half cell, and since the brick palette has no in-between hue the
// quantiser turns those into muddy outliers (a grey brick on the rim of the
// sun). Sampling per cell keeps the artwork on palette colours everywhere.
func paintScene(cols, rows, scale int, shade func(x, y, w, h float64) [3]float64) image.Image {
	if cols < 2 {
		cols = 2
	}
	if rows < 2 {
		rows = 2
	}
	if scale < 1 {
		scale = 1
	}

	w, h := cols*scale, rows*scale
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	fw, fh := float64(cols), float64(rows)

	for cy := 0; cy < rows; cy++ {
		y := float64(cy) + 0.5
		for cx := 0; cx < cols; cx++ {
			x := float64(cx) + 0.5
			c := shade(x, y, fw, fh)
			px := color.RGBA{
				R: uint8(clampF(math.Round(c[0]), 0, 255)),
				G: uint8(clampF(math.Round(c[1]), 0, 255)),
				B: uint8(clampF(math.Round(c[2]), 0, 255)),
				A: 255,
			}
			x0, y0 := cx*scale, cy*scale
			for dy := 0; dy < scale; dy++ {
				row := img.Pix[(y0+dy)*img.Stride : (y0+dy)*img.Stride+w*4]
				for dxi := 0; dxi < scale; dxi++ {
					o := (x0 + dxi) * 4
					row[o], row[o+1], row[o+2], row[o+3] = px.R, px.G, px.B, px.A
				}
			}
		}
	}
	return img
}

// colorOf returns the RGB of a palette entry by id, so the demos always use
// real brick colours.
func colorOf(id string) [3]float64 {
	for _, c := range classicColors {
		if c.ID == id {
			return [3]float64{float64(c.R), float64(c.G), float64(c.B)}
		}
	}
	return [3]float64{128, 128, 128}
}

func mix3(a, b [3]float64, t float64) [3]float64 {
	return [3]float64{lerp(a[0], b[0], t), lerp(a[1], b[1], t), lerp(a[2], b[2], t)}
}

// shade blends colour c towards k by amount t.
func shade(c, k [3]float64, t float64) [3]float64 { return mix3(c, k, t) }

func ease(t float64) float64 {
	t = clampF(t, 0, 1)
	return t * t * (3 - 2*t)
}

func lerp(a, b, t float64) float64 { return a + (b-a)*t }

func smoothstep(edge0, edge1, x float64) float64 {
	if edge1 == edge0 {
		return 0
	}
	t := clampF((x-edge0)/(edge1-edge0), 0, 1)
	return t * t * (3 - 2*t)
}

// ------------------------------------------------------------------ shapes

// inEllipse reports whether (x,y) lies inside an axis-aligned ellipse.
func inEllipse(x, y, cx, cy, rx, ry float64) bool {
	dx, dy := (x-cx)/rx, (y-cy)/ry
	return dx*dx+dy*dy <= 1
}

// inRotEllipse is inEllipse with the ellipse rotated by rot radians.
func inRotEllipse(x, y, cx, cy, rx, ry, rot float64) bool {
	dx, dy := x-cx, y-cy
	cs, sn := math.Cos(-rot), math.Sin(-rot)
	u, v := dx*cs-dy*sn, dx*sn+dy*cs
	return (u*u)/(rx*rx)+(v*v)/(ry*ry) <= 1
}

// inTri reports whether a point is inside the triangle abc.
func inTri(px, py, ax, ay, bx, by, cx, cy float64) bool {
	d1 := (px-bx)*(ay-by) - (ax-bx)*(py-by)
	d2 := (px-cx)*(by-cy) - (bx-cx)*(py-cy)
	d3 := (px-ax)*(cy-ay) - (cx-ax)*(py-ay)
	neg := d1 < 0 || d2 < 0 || d3 < 0
	pos := d1 > 0 || d2 > 0 || d3 > 0
	return !(neg && pos)
}

// inBox reports whether a point is inside a rectangle.
func inBox(x, y, x0, y0, x1, y1 float64) bool {
	return x >= x0 && x <= x1 && y >= y0 && y <= y1
}

// inRoundRect reports whether a point is inside a rounded rectangle.
func inRoundRect(x, y, x0, y0, x1, y1, r float64) bool {
	if x < x0 || x > x1 || y < y0 || y > y1 {
		return false
	}
	cx := clampF(x, x0+r, x1-r)
	cy := clampF(y, y0+r, y1-r)
	dx, dy := x-cx, y-cy
	return dx*dx+dy*dy <= r*r
}

// hash2 is a cheap deterministic hash used for stars, lit windows and texture.
// The third argument salts the stream so several independent fields (say stars
// and window lights) can be drawn from the same cell without correlating.
func hash2(x, y int, salt ...uint32) uint32 {
	h := uint32(x)*374761393 + uint32(y)*668265263
	if len(salt) > 0 {
		h += (salt[0] + 1) * 2654435761
	}
	h = (h ^ (h >> 13)) * 1274126177
	return h ^ (h >> 16)
}
