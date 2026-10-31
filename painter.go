package main

import (
	"image/color"
	"math"
)

// painter is the drawing surface every scene recipe writes through.
//
// Scenes are procedural on purpose. The mosaic pipeline samples the source once
// per cell, so a picture that is painted at cell resolution with hard edges
// survives quantisation perfectly: every sample already *is* a palette colour.
// Resampling a photo would blend two bricks into a third and leave muddy
// outliers along every edge.
//
// Coordinates are normalised: x/w and y/h, both in [0,1). Helpers take the same
// units so a recipe reads like a sketch ("an ellipse at 0.4,0.5").
type painter struct {
	cols, rows int
	w, h       float64
	shade      func(x, y, w, h float64) [3]float64
	// u/v are the cell centre in screen space, su/sv the same point in the
	// artwork's design box (see frame.go). Recipes work in su/sv.
	u, v   float64
	su, sv float64
	frame  frame
	ix     int
	iy     int
}

// hue is the palette a recipe is allowed to use: one mood, resolved to RGB.
type hue struct {
	m        mood
	sky      [][3]float64
	earth    [][3]float64
	warm     [][3]float64
	lit      [][3]float64
	light    [3]float64
	dark     [3]float64
	night    bool
	contrast float64
}

func newHue(m mood) *hue {
	toRGB := func(ids []string) [][3]float64 {
		out := make([][3]float64, len(ids))
		for i, id := range ids {
			out[i] = colorOf(id)
		}
		return out
	}
	h := &hue{
		m:        m,
		sky:      toRGB(m.sky),
		earth:    toRGB(m.earth),
		warm:     toRGB(m.warm),
		lit:      toRGB(m.lit),
		light:    colorOf("white"),
		dark:     colorOf("black"),
		night:    isNightMood(m.name),
		contrast: 1,
	}
	if len(h.sky) == 0 {
		h.sky = [][3]float64{colorOf("lblue")}
	}
	// the darkest member of the earth band is the natural silhouette colour,
	// and in a night mood it has to be the absolute darkest brick we own
	if h.night {
		h.dark = colorOf("black")
	} else {
		h.dark = darkest(h.earth)
	}
	return h
}

func isNightMood(name string) bool {
	return name == "深夜" || name == "霓虹" || name == "暮色"
}

func darkest(list [][3]float64) [3]float64 {
	if len(list) == 0 {
		return colorOf("black")
	}
	best := list[0]
	bestL := 1e9
	for _, c := range list {
		l := 0.2126*c[0] + 0.7152*c[1] + 0.0722*c[2]
		if l < bestL {
			bestL, best = l, c
		}
	}
	return best
}

// ------------------------------------------------ palette access (wrapping)

func (h *hue) earthAt(i int) [3]float64 { return pick(h.earth, i) }
func (h *hue) warmAt(i int) [3]float64  { return pick(h.warm, i) }
func (h *hue) skyAt(i int) [3]float64   { return pick(h.sky, i) }
func (h *hue) litAt(i int) [3]float64   { return pick(h.lit, i) }

func pick(list [][3]float64, i int) [3]float64 {
	if len(list) == 0 {
		return colorOf("lgray")
	}
	if i < 0 {
		i = len(list) + i%len(list)
	}
	return list[i%len(list)]
}

// ramp walks a ramp of swatches and interpolates between neighbours.
//
// The ground used to be four flat bands - one swatch per quarter of the depth -
// and that is what the field read as: a printed stripe. A continuous walk gives
// the quantiser something to resolve into a gradient, and a gradient along the
// ground is what tells the eye that the plane recedes.
func ramp(list [][3]float64, t float64) [3]float64 {
	if len(list) == 0 {
		return colorOf("lgray")
	}
	if len(list) == 1 {
		return list[0]
	}
	t = clamp01(t)
	x := t * float64(len(list)-1)
	i := int(x)
	if i >= len(list)-1 {
		return list[len(list)-1]
	}
	return mix3(list[i], list[i+1], x-float64(i))
}

// luma is perceived brightness, 0..255.
func luma(c [3]float64) float64 { return 0.299*c[0] + 0.587*c[1] + 0.114*c[2] }

// separate keeps a subject readable against its backdrop.
//
// In a pale mood a pale building on a pale sky is a blank picture, and the
// contact sheet had three of those. When the luma gap is too small the colour
// is pushed towards black - or towards white on a dark backdrop - which is the
// move a screen printer makes to keep a shape legible on a limited palette.
func separate(c, bg [3]float64, minGap float64) [3]float64 {
	if math.Abs(luma(c)-luma(bg)) >= minGap {
		return c
	}
	if luma(bg) > 128 {
		return mix3(c, colorOf("black"), 0.34)
	}
	return mix3(c, colorOf("white"), 0.30)
}

// coatColour picks an animal's coat.
//
// A cat is ginger, tabby brown, grey, cream or black in every photograph ever
// taken. Taking the coat from the mood's accent swatch is how the first version
// of this recipe ended up with a pink cat under a pink sky, so the candidates
// here are earth tones and the choice is made for contrast: a white cat painted
// on a snowfield is not a white cat, it is a missing cat.
func coatColour(seed uint32, ground [3]float64) [3]float64 {
	ids := []string{"nougat", "tan", "dbrown", "dgray", "white", "olive", "dtan"}
	gl := luma(ground)
	ok := make([][3]float64, 0, len(ids))
	for _, id := range ids {
		if c := colorOf(id); math.Abs(luma(c)-gl) > 26 {
			ok = append(ok, c)
		}
	}
	if len(ok) == 0 {
		return colorOf("white")
	}
	return ok[seed%uint32(len(ok))]
}

// skyRamp turns the mood's sky colours into a gradient ramp, zenith first.
func (h *hue) skyRamp() []rampStop {
	n := len(h.sky)
	out := make([]rampStop, n)
	for i, c := range h.sky {
		var at float64
		if n > 1 {
			at = float64(i) / float64(n-1)
		}
		out[i] = rampStop{at * 0.98, c}
	}
	return out
}

// ------------------------------------------------------- colour arithmetic

// layer paints colour c over the current colour with the given coverage.
func layer(under, over [3]float64, t float64) [3]float64 {
	if t <= 0 {
		return under
	}
	if t >= 1 {
		return over
	}
	return mix3(under, over, t)
}

// dither is a stable 0..1 noise value for a cell, used for star fields,
// brick texture and foliage. It never leaves the palette: callers gate on it.
func (p *painter) dither(salt uint32) float64 {
	return float64(hash2(p.ix, p.iy, salt)%1000) / 1000
}

// jitter returns a value in [0,1) that varies at a coarser scale than the cell,
// for texture that should read as clumps rather than static.
func (p *painter) jitter(block int, salt uint32) float64 {
	if block < 1 {
		block = 1
	}
	return float64(hash2(p.ix/block, p.iy/block, salt)%1000) / 1000
}

// --------------------------------------------------------------- geometry

// The helpers below take the unit square; the painter keeps a cached copy of
// the cell so they can be called as methods without passing coordinates.

func (p *painter) at(cx, cy, rx, ry float64) bool {
	dx, dy := (p.su-cx)/rx, (p.sv-cy)/ry
	return dx*dx+dy*dy <= 1
}

func (p *painter) box(x0, y0, x1, y1 float64) bool {
	return p.su >= x0 && p.su <= x1 && p.sv >= y0 && p.sv <= y1
}

func (p *painter) round(x0, y0, x1, y1, r float64) bool {
	return inRoundRect(p.su, p.sv, x0, y0, x1, y1, r)
}

func (p *painter) tri(ax, ay, bx, by, cx, cy float64) bool {
	return inTri(p.su, p.sv, ax, ay, bx, by, cx, cy)
}

func (p *painter) rot(cx, cy, rx, ry, rot float64) bool {
	return inRotEllipse(p.su, p.sv, cx, cy, rx, ry, rot)
}

// hair and hairV return the size of one cell in design units - the unit the
// recipes draw in, not screen space.
//
// This matters because the frame crops the design box: on a 3:4 grid the window
// is 0.75 units wide, so a cell covers 0.75/cols and not 1/cols. Using the
// screen fraction made every hairline 30% too thick on tall grids and too thin
// on wide ones, which showed up as features that vanished at some sizes.
func (p *painter) hair() float64 {
	if p.cols <= 0 {
		return 0.01
	}
	w := p.frame.x1 - p.frame.x0
	if w <= 0 {
		w = 1
	}
	return w / float64(p.cols)
}

func (p *painter) hairV() float64 {
	if p.rows <= 0 {
		return 0.01
	}
	h := p.frame.y1 - p.frame.y0
	if h <= 0 {
		h = 1
	}
	return h / float64(p.rows)
}

// clamp01 keeps a float in the unit interval.
func clamp01(v float64) float64 { return clampF(v, 0, 1) }

// toRGBA converts a float colour to the image type used by paintScene.
func toRGBA(c [3]float64) color.RGBA {
	return color.RGBA{
		R: uint8(clampF(c[0], 0, 255)),
		G: uint8(clampF(c[1], 0, 255)),
		B: uint8(clampF(c[2], 0, 255)),
		A: 255,
	}
}

// drawFn paints the subject of a scene, in place, onto the canvas.
//
// The recipe runs once per cell after the backdrop and ground are laid down. It
// writes through ctx.set / ctx.blend and reads its own noise through ctx.p:
// no return value, no allocation per cell.
type drawFn func(c *sceneCtx)

// unit maps a seed + salt to a stable value in [0,1). Two calls with different
// salts are independent, which is how one scene id yields a dozen layout
// decisions (peak position, sun height, ear size) without a config struct.
func unit(seed, salt uint32) float64 {
	return float64(hash2(int(seed%65536), int(salt%65536), seed/65536+1)) / 4294967295.0
}

// categories is filled in categories.go; the order is referenced by sceneCat.
var categories []category

func init() {
	categories = []category{
		{name: "大自然", word: catWord{"暖暖的", "太阳"}, subs: natureSubjects},
		{name: "小动物", word: catWord{"安静的", "小猫"}, subs: animalSubjects},
		// city, structures and planets are photograph-only labels now: the
		// generated library is characters, and a skyline is not a character.
		{name: "城市", word: catWord{"灯火里的", "城市"}, draw: drawCity},
		{name: "建筑", word: catWord{"海边的", "小屋"}, draw: drawStructure},
		{name: "甜甜的", word: catWord{"甜甜的", "蛋糕"}, subs: sweetSubjects},
		{name: "小车车", word: catWord{"驶过的", "小车"}, subs: vehicleSubjects},
		{name: "小花小草", word: catWord{"窗台的", "花束"}, subs: plantSubjects},
		{name: "星球", word: catWord{"遥远的", "星球"}, draw: drawSpace},
		{name: "小伙伴", word: catWord{"微笑的", "小孩"}, subs: peopleSubjects},
	}
}
