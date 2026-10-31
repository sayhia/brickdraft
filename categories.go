package main

import "math"

// The subject recipes. Each one draws the thing its category is named after,
// and each takes its proportions and layout from the scene seed, so one recipe
// covers a whole family: a single peak, a three-layer range, a volcano at dusk.
//
// Recipes only write palette colours (ctx.set) plus, sparingly, a tint
// (ctx.blend). Anything derived from a position or an angle must be snapped by
// the recipe itself - see the `step` helper - because an anti-aliased silhouette
// is exactly what leaves grey fringes on a brick mosaic.

// step quantises v to a grid of n divisions. Used to fake dithering and
// stair-stepping on purpose, which reads as "designed" rather than "aliased".
func step(v float64, n float64) float64 { return math.Floor(v*n) / n }

// hashf is a float stream in [0,1) keyed by a seed and an index.
func hashf(seed, i uint32) float64 {
	return float64(hash2(int(seed&0xffff), int(i&0xffff), seed>>16, i>>16)) / 4294967295.0
}

// ------------------------------------------------------------------ natural

// drawMountain paints a range: a dominant peak, a supporting ridge, an optional
// snow line, and a lake or a tree line at the foot.
func drawMountain(c *sceneCtx) {
	if c.toy {
		mountainToy(c)
		return
	}
	p, h := c.p, c.h
	horizon := c.horizon()
	if p.sv > horizon {
		return // the ground pass already painted the plain
	}

	peakX := 0.30 + 0.40*unit(c.seed, 71)
	peakY := 0.10 + 0.22*unit(c.seed, 72)
	// a narrow slope gives a spire, a wide one a massif: the same formula
	// should not always read as a single perfect cone
	slope := 0.18 + 0.26*unit(c.seed, 73)
	snowy := snowMood(h)

	// far ridge: higher, paler, behind everything
	farTop := ridgeY(p.su, horizon, peakX+0.06, peakY-0.02, slope*1.30, c.seed+81)
	if p.sv > farTop {
		c.set(h.earthAt(2))
		if snowy && p.sv < farTop+0.05 {
			c.set(colorOf("white"))
		}
	}

	// main ridge, lit from the left
	mainTop := ridgeY(p.su, horizon, peakX, peakY, slope, c.seed+82)
	if p.sv > mainTop {
		if p.su > peakX+(p.sv-mainTop)*0.30 {
			c.set(h.earthAt(3)) // shadow side
		} else {
			c.set(h.earthAt(1))
		}
		// snow cap: only the band right under the summit, and only some moods
		if snowy {
			cap := 0.10 + 0.10*unit(c.seed, 74)
			if p.sv < mainTop+cap {
				c.set(colorOf("white"))
				// a rock tooth poking through the cap keeps it from reading
				// like a paper cone - but only in the poster style; a child's
				// set wants a clean white cap
				if !c.toy && p.dither(c.seed+87) > 0.90 {
					c.set(h.earthAt(3))
				}
			}
		}
		// striations following the slope, one cell wide
		if !c.toy && p.dither(c.seed+83) > 0.972 {
			c.set(h.earthAt(3))
		}
	}

	// foreground: a lake that mirrors the range, or a tree line
	if c.seed%4 == 0 {
		if p.sv > horizon-0.14 && p.sv <= horizon {
			c.set(mix3(h.earthAt(2), h.dark, 0.30))
			// mirror band under the summit: regular rows, not dither - the
			// toy style keeps every pattern legible
			if math.Abs(p.su-peakX) < 0.16 && p.iy%2 == 0 {
				c.lift(0.22)
			}
		}
		return
	}
	// conifers on the last strip of ground, a sawtooth silhouette
	treeTop := horizon - 0.055 - 0.055*math.Abs(math.Sin(p.su*26+float64(c.seed%5)))
	if p.sv > treeTop && p.sv <= horizon {
		c.set(h.dark)
	}
}

// mountainToy paints one big triangular mountain with a snow cap, a smaller
// peak behind it, and a row of fir trees or a flat lake at its foot.
//
// The poster ridge is a noisy profile with a long slanted shoulder; at toy
// size that shoulder reads as a shark fin, and the dither striations read as
// dirt. A child's set is a triangle you can count the bricks of.
func mountainToy(c *sceneCtx) {
	p, h := c.p, c.h
	horizon := c.horizon()
	if p.sv > horizon {
		return
	}
	snowy := snowMood(h)

	// Three peaks, always three: a child can count them, and a range drawn as
	// one cone is the thing that read as a paper pyramid.
	mainX := 0.36 + 0.16*unit(c.seed, 71)
	mainY := 0.12 + 0.10*unit(c.seed, 72)
	mainW := 0.36 + 0.08*unit(c.seed, 73)

	rock := h.earthAt(1)
	lit := mix3(rock, colorOf("white"), 0.22)
	shade := mix3(rock, colorOf("black"), 0.34)
	if h.night {
		// a black triangle on a black sky is an empty frame: moonlight
		rock = mix3(h.earthAt(1), colorOf("white"), 0.20)
		lit = mix3(rock, colorOf("white"), 0.20)
		shade = mix3(rock, colorOf("black"), 0.40)
	}

	type peak struct {
		x, y, w    float64
		face, body [3]float64
	}
	// back to front: the side peaks first, so the main one overlaps them
	peaks := []peak{
		{mainX - 0.40, mainY + 0.22, mainW * 0.62, shade, shade},
		{mainX + 0.38, mainY + 0.15, mainW * 0.74, shade, rock},
		{mainX, mainY, mainW, lit, rock},
	}
	for _, pk := range peaks {
		if p.sv < pk.y || p.sv > horizon {
			continue
		}
		t := (p.sv - pk.y) / (horizon - pk.y)
		if math.Abs(p.su-pk.x) > pk.w*t {
			continue
		}
		// two flat faces split down the ridge: that single line is what makes
		// a triangle read as a mountain instead of as a pyramid sticker
		ridge := pk.x - pk.w*t*0.18
		if p.su < ridge {
			c.set(pk.face)
		} else {
			c.set(pk.body)
		}
		if snowy {
			// the cap has to keep some width at the summit, or ten cells of
			// cone hold two cells of snow and the peak wears a dot
			if p.sv < pk.y+0.115+0.045*unit(c.seed, 74) {
				c.set(colorOf("white"))
				if p.su > ridge+pk.w*t*0.34 {
					c.set(mix3(colorOf("white"), colorOf("lgray"), 0.60))
				}
			}
		}
	}

	// a short row of firs standing in front of the range. They used to be a
	// hedge along the horizon line; three big ones read as trees.
	n := 3
	for i := 0; i < n; i++ {
		tx := 0.14 + float64(i)*0.72/float64(n-1)
		hgt := 0.115 + 0.030*unit(c.seed, uint32(120+i))
		base := horizon + 0.03
		if p.sv <= base-hgt || p.sv > base {
			continue
		}
		t := (p.sv - (base - hgt)) / hgt
		if math.Abs(p.su-tx) > 0.062*(1-t)+0.006 {
			continue
		}
		// two tones, split at the trunk: a flat green triangle with no far
		// side is the same sticker problem the mountains had
		if p.su < tx-0.008 {
			c.set(mix3(colorOf("bgreen"), h.dark, 0.55))
		} else {
			c.set(h.dark)
		}
	}
}

// snowMood reports whether the palette warrants snow caps.
func snowMood(h *hue) bool {
	switch h.m.name {
	case "雪原", "极地", "晴日", "暮色", "苔原":
		return true
	}
	return false
}

// ridgeY evaluates a ridge profile: a peak with one bumpy shoulder.
func ridgeY(x, base, peakX, peakY, slope float64, seed uint32) float64 {
	d := math.Abs(x - peakX)
	t := clamp01(d / slope)
	v := peakY + (base-peakY)*math.Pow(t, 0.85)
	// shoulder noise, damped near the peak so the summit stays sharp
	bump := math.Sin(x*17+float64(seed%17)) * 0.022 * t
	bump += math.Sin(x*41+float64(seed%29)) * 0.010 * t
	return v + bump
}

// ------------------------------------------------------------------ animals

// drawAnimal paints a cat sitting front-on, cropped at the chest the way a
// portrait is: head, two triangular ears, shoulders, a curled tail, a pale
// muzzle and chest, and dark eyes.
//
// Two lessons came out of reviewing the first version of this recipe. A body
// ellipse of nearly the head's size, placed right under the head, merges the
// two into a pill; a cat needs a narrow chest that widens into a haunch. And a
// dark chest patch on a bright cat reads as a hole in the picture, so every
// patch here is a lighter tint of the fur, never the mood's silhouette colour.
func drawAnimal(c *sceneCtx) {
	if c.toy {
		animalToy(c)
		return
	}
	p, h := c.p, c.h
	ground := c.horizon()

	cx := 0.50 + 0.05*(unit(c.seed, 91)-0.5)
	headR := 0.135 + 0.030*unit(c.seed, 92)
	headY := 0.40 + 0.06*unit(c.seed, 93)
	haunchK := 1.45
	// children's sets scale the head up: a big head with big eyes is what
	// makes an animal read as a toy rather than as a cat-shaped shrub
	if c.toy {
		headR = 0.180 + 0.020*unit(c.seed, 92)
		headY = 0.420 + 0.040*unit(c.seed, 93)
		haunchK = 1.28
	}
	// a seated cat: haunch on the ground, chest under the chin
	haunchR := headR * haunchK
	haunchY := ground - haunchR*0.55
	chestR := headR * 1.05

	fur := coatColour(c.seed, h.earthAt(0))
	pale := mix3(fur, colorOf("white"), 0.45)
	// markings darken the fur itself. Mixing towards the mood's silhouette
	// colour looks right on a grey palette and turns a pink cat's stripes olive
	// on a green one, so black is the only safe direction here.
	dark := mix3(fur, colorOf("black"), 0.34)
	ink := colorOf("black")

	tailS := 1.0
	if c.seed%2 == 0 {
		tailS = -1.0
	}
	// tail: a chain of discs running out from the hip and curling up at the
	// tip. An arc hugging the hip reads as a carry handle, a full ring reads as
	// a second animal, and a stub on the ground reads as a foot.
	// tail: a quarter arc hugging the hip, from the floor up to the waist.
	// Constructions tried and rejected: a ring around the hip (a handle), a
	// chain rising beside the body (a cane or a raised paw), a stub on the
	// ground (a foot). An arc that starts at the floor, stays outside the body
	// and tucks back into it at the waist is the one that reads as a tail.
	{
		rc := haunchR * 0.46
		for i := 0; i <= 7; i++ {
			a := float64(i) / 7 * math.Pi / 2
			tx := cx + tailS*(haunchR*0.86+rc*math.Cos(a))
			ty := ground - haunchR*0.10 - rc*math.Sin(a)
			if p.at(tx, ty, haunchR*0.105, haunchR*0.105) {
				c.set(fur)
				if i >= 6 {
					c.set(pale) // the tip
				}
			}
		}
	}
	// haunch, then chest, so the chest sits in front
	if p.at(cx, haunchY+haunchR*0.15, haunchR, haunchR*0.85) {
		c.set(fur)
	}
	if p.at(cx, headY+headR*1.55, chestR*0.88, chestR*0.95) {
		c.set(fur)
	}
	// front legs: two pale columns down the chest
	for _, s := range []float64{-1, 1} {
		if p.at(cx+s*headR*0.42, headY+headR*2.05, headR*0.28, headR*0.56) {
			c.set(pale)
		}
	}
	// a dark seam between them. Without it the chest, the bib and both paws
	// fuse into one pear-shaped blob and the cat reads as a plush toy.
	if math.Abs(p.su-cx) < p.hair()*0.8 && p.sv > headY+headR*1.95 && p.sv < haunchY+haunchR*1.10 {
		c.set(dark)
	}
	// chest bib
	if p.at(cx, headY+headR*1.66, headR*0.50, headR*0.55) {
		c.set(pale)
	}

	// ears before the head, so their bases disappear into the skull. The apex
	// has to clear the crown by only a little: a tall narrow triangle reads as
	// a rabbit, which is what the previous numbers produced.
	earS := 1.0 + 0.16*unit(c.seed, 94)
	for _, s := range []float64{-1, 1} {
		bx := cx + s*headR*0.90 // outer edge of the base
		ix := cx + s*headR*0.08 // inner edge of the base
		ax := cx + s*headR*0.60 // apex, between the two
		// a wider base and a lower apex: the tall thin triangle the first
		// version drew read as a rabbit's ear on a cat
		if p.tri(ax, headY-headR*(1.24+0.08*earS), ix, headY-headR*0.80, bx, headY-headR*0.80) {
			c.set(fur)
		}
		if p.tri(ax, headY-headR*(1.14+0.06*earS), cx+s*headR*0.26, headY-headR*0.82, cx+s*headR*0.74, headY-headR*0.82) {
			c.set(pale) // inner ear
		}
	}

	// head
	if p.at(cx, headY, headR, headR*0.94) {
		c.set(fur)
	}

	// tabby stripes: two short bars across the brow, plus a cheek stripe
	if c.seed%3 != 2 {
		for i := 0; i < 2; i++ {
			y := headY - headR*(0.62-0.16*float64(i))
			if math.Abs(p.sv-y) < headR*0.10 && math.Abs(p.su-cx) < headR*(0.42-0.10*float64(i)) {
				c.set(dark)
			}
		}
	}

	// muzzle: two pale lobes and a small nose
	for _, s := range []float64{-1, 1} {
		if p.at(cx+s*headR*0.22, headY+headR*0.34, headR*0.30, headR*0.24) {
			c.set(pale)
		}
	}
	if p.tri(cx-headR*0.11, headY+headR*0.18, cx+headR*0.11, headY+headR*0.18, cx, headY+headR*0.32) {
		c.set(mix3(fur, colorOf("black"), 0.55))
	}
	// mouth: one cell down from the nose, then a short bar each side. In the
	// toy style the whole mouth is dropped: at 64 cells it is two dark cells
	// under the nose, which reads as a frown, and eyes plus nose already say
	// "cat".
	if !c.toy {
		if math.Abs(p.su-cx) < p.hair()*0.6 && p.sv > headY+headR*0.34 && p.sv < headY+headR*0.46 {
			c.set(ink)
		}
		if math.Abs(p.sv-(headY+headR*0.46)) < p.hairV()*0.6 && math.Abs(p.su-cx) < headR*0.26 {
			c.set(ink)
		}
	}

	// eyes: a dark almond with one highlight cell. The toy style goes bigger:
	// round glossy eyes are the single strongest "this is a friend" cue.
	eyeRx, eyeRy, hiR := headR*0.17, headR*0.20, headR*0.055
	if c.toy {
		eyeRx, eyeRy, hiR = headR*0.21, headR*0.25, headR*0.075
	}
	for _, s := range []float64{-1, 1} {
		ex := cx + s*headR*0.40
		ey := headY - headR*0.10
		if p.at(ex, ey, eyeRx, eyeRy) {
			c.set(ink)
		}
		if p.at(ex-eyeRx*0.28, ey-eyeRy*0.28, hiR, hiR) {
			c.set(colorOf("white"))
		}
	}

	// whiskers: one row a side, inside the head's width, so they read as a line
	// rather than as two rows fusing into a green block. Skipped in the toy
	// style: at 64 cells a whisker is a stray dark dash on the cheek.
	if !c.toy && math.Abs(p.sv-(headY+headR*0.30)) < p.hairV()*0.5 {
		d := math.Abs(p.su - cx)
		if d > headR*0.50 && d < headR*0.92 {
			c.set(dark)
		}
	}
}

// --------------------------------------------------------------------- city

// drawCity paints a skyline: three depth layers of towers, lit windows and a
// couple of antennae. Layer colours get darker towards the foreground.
func drawCity(c *sceneCtx) {
	if c.toy {
		cityToy(c)
		return
	}
	p, h := c.p, c.h
	ground := c.horizon()
	if p.sv > ground {
		return
	}

	// three passes, far to near, each with its own tower rhythm
	for layer := 0; layer < 3; layer++ {
		tone := h.earthAt(2 + layer)
		if h.night {
			tone = h.dark
		}
		baseY := ground - 0.02*float64(2-layer)
		seed := c.seed + uint32(layer)*17
		// towers are laid out on a coarse column grid so edges stay vertical
		colW := 0.075 - float64(layer)*0.018
		col := int(math.Floor(p.su / colW))
		if col < 0 {
			continue
		}
		ix := uint32(col)
		top := baseY - 0.14 - 0.34*hashf(seed, ix)
		if col%3 == 0 {
			top -= 0.06 // an occasional spire
		}
		// leave gaps between towers
		inset := colW * 0.12
		x0 := float64(col)*colW + inset
		x1 := x0 + colW - inset*2
		if p.sv < top || p.su < x0 || p.su > x1 {
			continue
		}
		c.set(tone)

		// windows: a lattice that only lands on some cells
		if p.sv > top+0.035 && p.sv < baseY-0.02 && p.su > x0+0.012 && p.su < x1-0.012 {
			inRow := p.iy%2 == 0
			inCol := p.ix%2 == 0
			if inRow && inCol && hashf(seed+9, ix) > 0.35 {
				lit := p.dither(seed + 33)
				if lit > 0.30 {
					c.set(h.litAt(0))
				}
				if lit > 0.72 {
					c.set(h.litAt(1))
				}
			}
		}

		// roofs catch a rim of light in the daylight moods
		if p.sv < top+p.hairV()*1.2 && !h.night {
			c.blend(colorOf("white"), 0.25)
		}
		// antenna
		if col%3 == 0 && math.Abs(p.su-(x0+x1)/2) < p.hair()*0.55 &&
			p.sv > top-0.07 && p.sv < top {
			c.set(h.dark)
			if h.night && p.sv < top-0.055 {
				c.set(colorOf("red"))
			}
		}
	}
}

// cityToy paints the children's-set skyline: three chunky buildings with big
// square windows, and a paler pair behind them for depth.
//
// The poster recipe lays one narrow tower every 7% of the width, which at 64
// cells is a row of thin vertical bars - a barcode, not a city. A child's set
// is four buildings you can count, each with windows the size of a brick.
func cityToy(c *sceneCtx) {
	p, h := c.p, c.h
	ground := c.horizon()
	if p.sv > ground {
		return
	}
	skyAtGround := h.skyAt(len(h.sky) - 1)

	// far row, paler, taller: depth without detail. Slots are fixed so the
	// skyline always reads as a street instead of a pile.
	for i := 0; i < 2; i++ {
		w := 0.19 + 0.04*unit(c.seed, uint32(200+i))
		cx := 0.34 + 0.34*float64(i) + 0.05*unit(c.seed, uint32(210+i))
		top := ground - 0.32 - 0.12*unit(c.seed, uint32(220+i))
		if p.box(cx-w/2, top, cx+w/2, ground-0.015) {
			c.set(separate(mix3(h.earthAt(2), skyAtGround, 0.30), skyAtGround, 18))
		}
	}

	// near row: three buildings the child can count, on three slots
	for i := 0; i < 3; i++ {
		w := 0.175 + 0.045*unit(c.seed, uint32(300+i))
		cx := 0.195 + 0.305*float64(i) + 0.035*unit(c.seed, uint32(310+i))
		top := ground - 0.15 - 0.22*unit(c.seed, uint32(320+i))
		// one tower has to break the roofline, or the skyline is a hedgerow
		if i == 1 && c.seed%2 == 0 {
			top -= 0.14
		}
		if !p.box(cx-w/2, top, cx+w/2, ground) {
			continue
		}
		// pastel concrete, the way a toy city is painted: warm sand, mint,
		// powder blue. The first version drew every tower in the same grey
		// and the skyline read as an office block.
		wall := colorOf([]string{"tan", "white", "sandgreen", "lblue", "nougat", "vlgray"}[c.seed%6])
		if h.night {
			wall = mix3(h.dark, colorOf("sandblue"), 0.30)
		}
		wall = separate(wall, skyAtGround, 26)
		c.set(wall)
		// a darker cornice, one cell high, so the roof has an edge
		if p.sv < top+p.hairV()*1.1 {
			c.set(mix3(wall, colorOf("black"), 0.30))
		}
		// door on the ground floor
		if p.box(cx-0.030, ground-0.095, cx+0.030, ground) {
			c.set(mix3(wall, colorOf("black"), 0.55))
		}
		// two columns of big windows, one floor at a time. They used to be
		// speckle; a window has to be at least three cells across to read as a
		// window rather than as dirt on the wall.
		for wy := top + 0.045; wy < ground-0.125; wy += 0.062 {
			for _, s := range []float64{-1, 1} {
				wx := cx + s*w*0.215
				if p.box(wx-0.030, wy, wx+0.030, wy+0.036) {
					if h.night {
						c.set(h.litAt(0))
					} else {
						c.set(mix3(wall, colorOf("dblue"), 0.55))
					}
					if p.box(wx-0.030, wy, wx+0.030, wy+0.009) {
						c.set(colorOf("white")) // sill
					}
				}
			}
		}
	}

	// one antenna on the middle roof, with a red lamp at night
	cx := 0.195 + 0.305 + 0.035*unit(c.seed, 311)
	top := ground - 0.17 - 0.19*unit(c.seed, 321)
	if math.Abs(p.su-cx) < p.hair()*1.1 && p.sv > top-0.07 && p.sv < top {
		c.set(h.dark)
		if h.night && p.sv < top-0.055 {
			c.set(colorOf("red"))
		}
	}
}

// --------------------------------------------------------- structures, food

// drawStructure paints a building: a gabled cottage, a tower or a lighthouse,
// with a door, windows and a lit roof.
func drawStructure(c *sceneCtx) {
	p, h := c.p, c.h
	ground := c.horizon()
	if p.sv > ground {
		return
	}

	kind := c.seed % 3
	// fixed brick colours rather than the mood's earth ramp: in the snow moods
	// the ramp is white on white and the house disappears into the sky
	walls := colorOf([]string{"tan", "sandgreen", "nougat", "lgray", "white"}[c.seed%5])
	roof := colorOf([]string{"red", "dorange", "dazure", "dbrown", "coral"}[(c.seed/3)%5])
	if h.night {
		walls = mix3(walls, colorOf("dblue"), 0.50)
	}
	// a building the same tone as the sky behind it is a missing building:
	// this is what turned half the night towers into black rectangles
	walls = separate(walls, h.skyAt(len(h.sky)-1), 26)

	switch kind {
	case 0: // cottage, wide and low, with a steep toy roof
		w := 0.26 + 0.06*unit(c.seed, 31)
		cx := 0.50
		top := ground - 0.24
		if p.box(cx-w, top, cx+w, ground) {
			c.set(walls)
		}
		if p.tri(cx, top-0.24, cx-w-0.05, top, cx+w+0.05, top) {
			c.set(roof)
		}
		// chimney
		if p.box(cx+w*0.45, top-0.26, cx+w*0.60, top-0.12) {
			c.set(mix3(roof, colorOf("black"), 0.55))
		}
		// door
		if p.box(cx-0.032, ground-0.105, cx+0.032, ground) {
			c.set(mix3(walls, colorOf("black"), 0.62))
		}
		// a white knob, one cell, because a door without one is a hole
		if p.box(cx+0.014, ground-0.055, cx+0.022, ground-0.045) {
			c.set(colorOf("white"))
		}
		// two windows, lit at night
		for _, s := range []float64{-1, 1} {
			if p.box(cx+s*w*0.62-0.028, top+0.07, cx+s*w*0.62+0.028, top+0.14) {
				c.set(h.litAt(0))
				if p.box(cx+s*w*0.62-0.028, top+0.07, cx+s*w*0.62+0.028, top+0.085) {
					c.set(mix3(walls, colorOf("black"), 0.45)) // sill
				}
			}
		}
	case 1: // tower: wider than a pylon, with a plinth so it sits on the ground
		w := 0.13 + 0.035*unit(c.seed, 32)
		cx := 0.50
		top := ground - 0.44
		if p.box(cx-w, top, cx+w, ground) {
			c.set(walls)
			if !c.toy && p.iy%2 == 0 {
				c.blend(h.dark, 0.16) // masonry courses, horizontal
			}
		}
		// plinth
		if p.box(cx-w*1.22, ground-0.09, cx+w*1.22, ground) {
			c.set(mix3(walls, colorOf("black"), 0.30))
		}
		// battlements: teeth four cells wide, so they read as a castle crown
		// rather than as a row of spikes
		if p.box(cx-w-0.035, top-0.045, cx+w+0.035, top) && (p.ix/4)%2 == 0 {
			c.set(h.dark)
		}
		// two windows with a warm glow
		for _, wy := range []float64{ground - 0.34, ground - 0.20} {
			if p.box(cx-0.030, wy-0.05, cx+0.030, wy) {
				c.set(h.litAt(0))
				if p.box(cx-0.030, wy-0.05, cx+0.030, wy-0.035) {
					c.set(mix3(walls, colorOf("black"), 0.45)) // lintel
				}
			}
		}
		// a door at the foot, so the tower is somewhere you could walk into
		if p.box(cx-0.035, ground-0.09, cx+0.035, ground) {
			c.set(mix3(walls, colorOf("black"), 0.60))
		}
		// flag on a pole: the one thing that makes a grey box read as a castle
		if math.Abs(p.su-(cx+w*0.55)) < p.hair() && p.sv > top-0.16 && p.sv < top-0.04 {
			c.set(colorOf("dgray"))
		}
		if p.tri(cx+w*0.55, top-0.155, cx+w*0.55+0.10, top-0.125, cx+w*0.55, top-0.095) {
			c.set(colorOf("red"))
		}
	case 2: // lighthouse: tapered bands, lantern room and beam
		cx := 0.46
		base := ground
		top := ground - 0.56
		hw := 0.075 + 0.030*(p.sv-top)/(base-top)
		if p.sv >= top && p.sv <= base && math.Abs(p.su-cx) <= hw {
			if int((p.sv-top)/0.075)%2 == 0 {
				c.set(colorOf("white"))
			} else {
				c.set(roof) // the stripe has to be a real colour: the mood's
				// third accent is white in the sunny palettes, and white on
				// white is an unstriped tower
			}
			if p.su > cx+hw*0.35 {
				c.shadow(0.30)
			}
		}
		// lantern room and roof
		if p.sv > top-0.06 && p.sv < top && math.Abs(p.su-cx) <= 0.045 {
			c.set(h.litAt(0))
		}
		if p.tri(cx, top-0.10, cx-0.055, top-0.06, cx+0.055, top-0.06) {
			c.set(h.dark)
		}
		// beam, only when the mood is dark enough to show it
		if h.night && p.su > cx+0.05 && p.sv < top+0.10 {
			axis := top - (p.su-cx)*0.22
			if math.Abs(p.sv-axis) < 0.035 {
				c.blend(h.litAt(0), 0.55)
			}
		}
	}
}

// drawFood paints a dessert: a cake slice or layered tart on a plate, with a
// cherry. Warm moods only, but any palette can do it.
func drawFood(c *sceneCtx) {
	p, h := c.p, c.h
	ground := c.horizon()
	if c.toy {
		foodToy(c, ground)
		return
	}
	if p.sv > ground {
		return
	}

	plateY := ground - 0.015
	cx := 0.50

	// plate
	if p.at(cx, plateY, 0.34, 0.045) {
		c.set(colorOf("white"))
	}
	// cake body: a trapezoid, wider at the bottom
	topY := ground - 0.30
	botY := ground - 0.03
	if p.sv > topY && p.sv < botY {
		halfW := 0.13 + 0.07*(p.sv-topY)/(botY-topY)
		if math.Abs(p.su-cx) < halfW {
			band := int((p.sv - topY) / 0.055)
			switch band {
			case 0:
				c.set(h.warmAt(3)) // frosting
			case 1:
				c.set(h.earthAt(1)) // filling
			default:
				c.set(h.warmAt(1)) // sponge
			}
			// a shadow on the right third - poster style only; the toy cake
			// stays flat so every band is one plate colour
			if !c.toy && p.su > cx+halfW*0.25 {
				c.shadow(0.16)
			}
		}
	}
	// cherry with a highlight
	if p.at(cx, topY-0.030, 0.045, 0.038) {
		c.set(h.warmAt(0))
	}
	if p.at(cx-0.012, topY-0.042, 0.014, 0.012) {
		c.set(colorOf("white"))
	}
	// steam or a candle flame, one cell wide - the dither speckle reads as
	// noise in the toy style, so only the poster keeps it
	if !c.toy && p.sv < topY-0.06 && math.Abs(p.su-cx) < p.hair()*0.9 && p.dither(c.seed+44) > 0.72 {
		c.set(h.warmAt(1))
	}
}

// foodToy paints a three-tier birthday cake: stacked slabs, frosting on every
// tier, a cherry on top.
//
// A single trapezoid with horizontal bands reads as a sand pile, which is what
// the first toy cake looked like. Stacking three tiers of different widths is
// the shape every child already knows from a birthday.
func foodToy(c *sceneCtx, ground float64) {
	p := c.p
	cx := 0.50
	// a cake is brown sponge and white frosting; the mood accent turned it
	// grey-olive, which is the colour of a rock cake
	sponge := colorOf("nougat")
	crumb := mix3(colorOf("nougat"), colorOf("dtan"), 0.60)
	frost := colorOf("white")
	trim := colorOf("pink")
	if c.seed%2 == 1 {
		frost, trim = colorOf("pink"), colorOf("white")
	}
	plate := colorOf("white")

	// plate: a wide white disc with a shadow ring under it
	if p.at(cx, ground-0.010, 0.35, 0.052) {
		c.set(plate)
	}

	// three tiers, bottom to top. Each is a slab with a frosting band on top.
	type tier struct {
		halfW float64
		y0    float64 // top of the tier
		y1    float64 // bottom of the tier
	}
	tiers := []tier{
		{0.30, ground - 0.20, ground - 0.030},
		{0.225, ground - 0.32, ground - 0.205},
		{0.155, ground - 0.42, ground - 0.325},
	}
	for _, t := range tiers {
		if p.sv < t.y0 || p.sv >= t.y1 || math.Abs(p.su-cx) >= t.halfW {
			continue
		}
		c.set(sponge)
		// the sponge is two layers: a cake slice always shows a seam
		if p.sv > t.y0+0.052 {
			c.set(crumb)
		}
		// frosting cap, two cells thick, with cream dripping down the side
		if p.sv < t.y0+0.030 {
			c.set(frost)
		} else if p.sv < t.y0+0.056 {
			d := math.Abs(math.Mod(p.su-cx+0.045, 0.090) - 0.045)
			if d > 0.029 {
				c.set(trim)
			}
		}
	}

	// cherry and its stem, standing on the top tier
	topY := ground - 0.42
	if p.at(cx, topY-0.035, 0.042, 0.038) {
		c.set(colorOf("red"))
	}
	if p.at(cx-0.012, topY-0.046, 0.013, 0.012) {
		c.set(colorOf("white"))
	}
	if math.Abs(p.su-(cx+0.008)) < p.hair()*1.1 && p.sv > topY-0.095 && p.sv < topY-0.070 {
		c.set(colorOf("darkred"))
	}
}

// ---------------------------------------------------------------- vehicles

// drawVehicle paints a boat (hull, sail, mast) on water, or a car when the seed
// asks for one.
func drawVehicle(c *sceneCtx) {
	p, h := c.p, c.h
	water := c.horizon()

	if isCarScene(c) {
		drawCar(c, water)
		return
	}
	if c.toy {
		boatToy(c, water)
		return
	}

	cx := 0.46 + 0.10*unit(c.seed, 51)
	if p.sv > water-0.02 && p.sv < water+0.05 && math.Abs(p.su-cx) < 0.18 {
		// hull: a shallow trapezoid
		halfW := 0.18 * (1 - math.Abs(p.sv-(water+0.015))/0.035)
		if math.Abs(p.su-cx) < halfW {
			c.set(h.dark)
		}
	}
	// mast, two cells wide so it survives the quantiser at any grid size
	mid := 2 * p.hair()
	if math.Abs(p.su-cx) < mid && p.sv > water-0.34 && p.sv < water-0.01 {
		c.set(h.dark)
	}
	// main sail, and a jib on the other side of the mast
	if p.tri(cx+mid, water-0.33, cx+mid, water-0.04, cx+0.20, water-0.04) {
		c.set(h.warmAt(0))
	}
	if p.tri(cx-mid, water-0.26, cx-mid, water-0.05, cx-0.12, water-0.05) {
		c.set(h.warmAt(2))
	}
	// waterline shadow
	if p.sv > water && p.sv < water+0.06 && math.Abs(p.su-cx) < 0.20 && p.iy%2 == 1 {
		c.shadow(0.25)
	}
}

// isCarScene reports whether this vehicle scene is a car rather than a boat.
// The stage under the subject has to agree with it, or the car drives on water.
func isCarScene(c *sceneCtx) bool { return c.seed%4 == 3 }

// boatToy paints a sailboat big enough to read as a toy: a deep hull with a
// white deck line, a thick mast and two flat sails.
//
// The poster boat has a hull four cells tall and a mast one cell wide, which
// at toy size is a stick stuck in a pond.
func boatToy(c *sceneCtx, water float64) {
	p, h := c.p, c.h
	cx := 0.47 + 0.06*unit(c.seed, 51)
	deck := water - 0.055

	// the hull has to separate from the near water band, and the sails from the
	// sky: in a pale mood a white sail on a white sky is a bare mast
	hull := colorOf("redbrown")
	if c.seed%2 == 1 {
		hull = colorOf("dazure")
	}
	sail := separate(h.warmAt(0), h.skyAt(len(h.sky)-1), 24)
	jib := separate(h.warmAt(2), h.skyAt(len(h.sky)-1), 20)
	if p.sv >= deck && p.sv <= water+0.02 {
		t := (p.sv - deck) / (water + 0.02 - deck)
		if math.Abs(p.su-cx) < 0.235-0.105*t {
			c.set(hull)
			if p.sv < deck+p.hairV()*1.3 {
				c.set(colorOf("white")) // deck line
			}
		}
	}
	// mast: three cells wide, so it survives any grid
	if math.Abs(p.su-cx) < p.hair()*1.6 && p.sv > 0.20 && p.sv < deck {
		c.set(hull)
	}
	mastX := cx + p.hair()*1.6
	// mainsail, then a paler jib on the other side. Both are wide triangles:
	// a narrow one at 64 cells is a flag on a pole.
	if p.tri(mastX, 0.215, mastX, deck-0.005, cx+0.245, deck-0.005) {
		c.set(sail)
	}
	if p.tri(mastX-2*p.hair()*1.6, 0.285, mastX-2*p.hair()*1.6, deck-0.005, cx-0.165, deck-0.005) {
		c.set(jib)
	}
	// a pennant at the masthead
	if p.tri(mastX, 0.195, mastX+0.055, 0.222, mastX, 0.248) {
		c.set(h.warmAt(3))
	}
}

// drawCar paints a small car in profile.
func drawCar(c *sceneCtx, road float64) {
	p := c.p
	cx := 0.50
	bodyTop := road - 0.16
	cabin := road - 0.26
	// a car in a child's set is a primary colour with a window you can see
	// through. Taking the paint from the mood accent gave a white car on a
	// white road in half the palettes.
	body := colorOf([]string{"red", "azure", "lyellow", "orange", "dgreen", "purple"}[c.seed/4%6])
	roof := mix3(body, colorOf("white"), 0.22)
	if p.round(cx-0.22, bodyTop, cx+0.22, road-0.04, 0.035) {
		c.set(body)
	}
	if p.round(cx-0.13, cabin, cx+0.10, bodyTop+0.01, 0.03) {
		c.set(roof)
	}
	// windows
	if p.box(cx-0.10, cabin+0.02, cx-0.01, bodyTop-0.01) ||
		p.box(cx+0.01, cabin+0.02, cx+0.08, bodyTop-0.01) {
		c.set(colorOf("lblue"))
	}
	// a headlight band and a wheel arch, so the body is not a flat slab
	if p.box(cx+0.155, bodyTop+0.020, cx+0.215, bodyTop+0.045) {
		c.set(colorOf("lyellow"))
	}
	// wheels
	for _, s := range []float64{-1, 1} {
		if p.at(cx+s*0.13, road-0.035, 0.045, 0.045) {
			c.set(colorOf("black"))
		}
		if p.at(cx+s*0.13, road-0.035, 0.018, 0.018) {
			c.set(colorOf("lgray"))
		}
	}
	// headlight
	if p.box(cx+0.20, road-0.12, cx+0.225, road-0.095) {
		c.set(colorOf("lyellow"))
	}
}

// animalToy paints the children's-set cat: one big round head, one pear of a
// body, two paws, two ears and a tail that leaves the silhouette.
//
// The layered recipe above builds a cat out of a dozen overlapping ellipses in
// the same colour. At 64 cells the layers stop being layers and become one
// white pear with a grey band across the middle, which is exactly what a child
// cannot read as a cat. Flat shapes with gaps between them are the fix.
func animalToy(c *sceneCtx) {
	p := c.p
	ground := c.horizon()
	cx := 0.50 + 0.03*(unit(c.seed, 91)-0.5)
	headR := 0.185 + 0.020*unit(c.seed, 92)
	headY := 0.335 + 0.030*unit(c.seed, 93)

	fur := coatColour(c.seed, colorOf("bgreen"))
	pale := mix3(fur, colorOf("white"), 0.55)
	dark := mix3(fur, colorOf("black"), 0.32)
	ink := colorOf("black")
	tailS := 1.0
	if c.seed%2 == 0 {
		tailS = -1.0
	}

	// tail first: a fat arc from the floor curving up beside the hip, so it
	// reads as a tail and not as a handle or a raised paw
	for i := 0; i <= 9; i++ {
		t := float64(i) / 9
		a := t * math.Pi * 0.72
		// the base has to start inside the body, or the tail is a hook
		// floating beside the cat
		tx := cx + tailS*(headR*0.74+headR*0.54*math.Sin(a))
		ty := ground - headR*0.10 - headR*0.95*t*t
		if p.at(tx, ty, headR*0.15, headR*0.15) {
			c.set(fur)
			if i >= 8 {
				c.set(pale)
			}
		}
	}

	// body: a pear, narrower than the head and wider at the floor
	bodyTop := headY + headR*0.62
	bodyMid := (bodyTop + ground) / 2
	bodyH := (ground - bodyTop) / 2
	if p.at(cx, bodyMid, headR*0.86, bodyH+headR*0.10) {
		c.set(fur)
	}
	// bib and paws in the lighter tone, as separate shapes with fur between
	if p.at(cx, bodyMid+headR*0.18, headR*0.42, bodyH*0.72) {
		c.set(pale)
	}
	for _, s := range []float64{-1, 1} {
		if p.at(cx+s*headR*0.44, ground-headR*0.14, headR*0.27, headR*0.18) {
			c.set(pale)
		}
	}

	// ears: wide triangles whose bases disappear into the skull
	for _, s := range []float64{-1, 1} {
		if p.tri(cx+s*headR*0.52, headY-headR*1.30, cx+s*headR*0.12, headY-headR*0.72,
			cx+s*headR*0.92, headY-headR*0.72) {
			c.set(fur)
		}
		if p.tri(cx+s*headR*0.50, headY-headR*1.12, cx+s*headR*0.28, headY-headR*0.76,
			cx+s*headR*0.74, headY-headR*0.76) {
			c.set(pale)
		}
	}
	// head: a circle, drawn last so it sits over the ears and the neck
	if p.at(cx, headY, headR, headR*0.94) {
		c.set(fur)
	}

	// a pale muzzle patch, so the face is not two eyes floating on a ball
	if p.at(cx, headY+headR*0.42, headR*0.44, headR*0.30) {
		c.set(pale)
	}
	// nose: a small down triangle
	if p.tri(cx-headR*0.11, headY+headR*0.30, cx+headR*0.11, headY+headR*0.30, cx, headY+headR*0.46) {
		c.set(mix3(fur, colorOf("red"), 0.55))
	}
	// two forehead stripes: the one marking that still survives at this size
	for i := 0; i < 2; i++ {
		y := headY - headR*(0.66-0.14*float64(i))
		if math.Abs(p.sv-y) < headR*0.075 && math.Abs(p.su-cx) < headR*(0.34-0.08*float64(i)) {
			c.set(dark)
		}
	}
	// eyes: round, wide apart, with a highlight each
	for _, s := range []float64{-1, 1} {
		ex := cx + s*headR*0.40
		ey := headY - headR*0.06
		if p.at(ex, ey, headR*0.20, headR*0.24) {
			c.set(ink)
		}
		if p.at(ex-headR*0.05, ey-headR*0.06, headR*0.07, headR*0.07) {
			c.set(colorOf("white"))
		}
	}
}

// --------------------------------------------------------------- botanical

// drawBotanical paints a flowering branch: a vase or stem, leaves, and a
// cluster of petals. The petal count and hue come from the seed.
func drawBotanical(c *sceneCtx) {
	p, h := c.p, c.h
	ground := c.horizon()
	if c.toy {
		botanicalToy(c, ground)
		return
	}
	cx := 0.50

	// vase
	if p.tri(cx-0.10, ground-0.20, cx+0.10, ground-0.20, cx+0.065, ground) ||
		p.box(cx-0.065, ground-0.30, cx+0.065, ground-0.14) {
		c.set(h.earthAt(3))
	}
	// stem, two cells wide so it survives the quantiser
	if math.Abs(p.su-cx) < p.hair()*1.2 && p.sv < ground-0.26 && p.sv > 0.10 {
		c.set(h.earthAt(0))
	}
	// two branches
	for _, s := range []float64{-1, 1} {
		if p.rot(cx+s*0.09, ground-0.36-s*0.02, 0.10, p.hair()*1.5, s*0.6) {
			c.set(h.earthAt(0))
		}
	}
	// leaves
	for i := 0; i < 4; i++ {
		ly := ground - 0.24 - float64(i)*0.090
		lx := cx + float64(1-2*(i%2))*0.080
		if p.rot(lx, ly, 0.058, 0.024, float64(1-2*(i%2))*0.5) {
			c.set(h.earthAt(1))
		}
	}
	// flowers: rings of petals around a core
	petal := h.warmAt(0)
	core := h.warmAt(1)
	headY := 0.13 + 0.05*unit(c.seed, 61)
	for _, hx := range []float64{cx - 0.10, cx + 0.02, cx + 0.13} {
		hy := headY + 0.045*float64(int(hx*100)%3)
		for i := 0; i < 6; i++ {
			a := float64(i) * math.Pi / 3
			px, py := hx+math.Cos(a)*0.045, hy+math.Sin(a)*0.045
			if p.rot(px, py, 0.036, 0.020, a) {
				c.set(petal)
			}
		}
		if p.at(hx, hy, 0.020, 0.020) {
			c.set(core)
		}
	}
}

// botanicalToy paints a potted flower: a terracotta pot, a thick stem, two
// leaves and one big daisy.
//
// The poster branch puts three small flower heads at 0.13 of the frame, so they
// land against the sky, crop on tall grids and read as kites. One big head,
// low enough to sit on the stem, is the toy answer.
func botanicalToy(c *sceneCtx, ground float64) {
	p, h := c.p, c.h
	cx := 0.50
	pot := mix3(colorOf("dtan"), colorOf("red"), 0.28)
	pot = separate(pot, ramp(h.earth, 0.30), 20)
	// a stem is green in every mood: the earth ramp is white in the snow
	// palettes, and a white stem on a white wall is a floating flower
	stem := mix3(colorOf("green"), colorOf("bgreen"), 0.35)

	// pot: a trapezoid, wider at the rim
	rimY := ground - 0.24
	for _, seg := range [][2]float64{{rimY, ground - 0.20}, {ground - 0.20, ground}} {
		wide := 0.130
		if seg[0] > rimY+0.001 {
			wide = 0.100
		}
		if p.sv >= seg[0] && p.sv < seg[1] && math.Abs(p.su-cx) < wide {
			c.set(pot)
		}
	}
	// rim band
	if p.sv >= rimY && p.sv < rimY+p.hairV()*1.2 && math.Abs(p.su-cx) < 0.118 {
		c.set(mix3(pot, colorOf("white"), 0.28))
	}

	// stem: two cells, from the pot to the flower
	if math.Abs(p.su-cx) < p.hair()*1.4 && p.sv > 0.30 && p.sv < rimY {
		c.set(stem)
	}
	// two leaves on the stem, mirrored: one leaf higher than the other read
	// as a windmill rather than as a plant
	if p.rot(cx-0.062, ground-0.30, 0.052, 0.021, 0.55) ||
		p.rot(cx+0.062, ground-0.30, 0.052, 0.021, -0.55) {
		c.set(mix3(stem, colorOf("lime"), 0.40))
	}
	// one big daisy: five round petals on a ring, spaced so the gaps survive
	// the quantiser, around a fat yellow core.
	//
	// Radial petals made a starfish, which is what the first toy daisy looked
	// like. Five is the count that keeps a gap at this radius; six on the same
	// ring touch and the flower becomes a gear.
	fx, fy := cx, 0.285
	petal := flowerPetal(c.seed, h.skyAt(len(h.sky)-1))
	// five lobes in polar space: r(angle) pinches between petals. Separate
	// petal ellipses were sampled into horizontal dashes at 48 cells, because
	// a six-cell circle is not a circle in a grid. A lobe function is.
	if d := math.Hypot(p.su-fx, p.sv-fy); d > 0.001 && d < 0.118 {
		th := math.Atan2(p.sv-fy, p.su-fx)
		if d < 0.112*(0.56+0.44*math.Abs(math.Cos(2.5*th))) {
			c.set(petal)
		}
	}
	if p.at(fx, fy, 0.046, 0.044) {
		c.set(colorOf("dorange"))
	}
	if p.at(fx, fy, 0.024, 0.022) {
		c.set(colorOf("lyellow"))
	}
	// three seed dots in the core: the one detail that says "sunflower"
	for i := 0; i < 3; i++ {
		a := float64(i) * 2 * math.Pi / 3
		if p.at(fx+math.Cos(a)*0.020, fy+math.Sin(a)*0.018, 0.008, 0.008) {
			c.set(colorOf("dorange"))
		}
	}
}

// flowerPetal picks a petal colour that is actually visible against the sky.
// White petals on a white wall turn into a grey lollipop, so the candidates are
// the flowers a garden actually has and the pick is made on contrast.
func flowerPetal(seed uint32, sky [3]float64) [3]float64 {
	ids := []string{"red", "coral", "dpink", "lyellow", "lorange", "purple", "white"}
	sl := luma(sky)
	for i := 0; i < len(ids); i++ {
		c := colorOf(ids[(int(seed)+i)%len(ids)])
		if math.Abs(luma(c)-sl) > 34 {
			return c
		}
	}
	return colorOf("red")
}

// ------------------------------------------------------------------ space
// drawSpace paints a planet in deep sky. Space scenes skip the ground plane
// entirely, so without a variation on composition a hundred of them would look
// like one picture recoloured; the seed picks one of four structures (ring,
// bare crescent, giant close-up, moonlit) and then varies size, tilt and moon
// count inside it.
func drawSpace(c *sceneCtx) {
	if !c.toy {
		nebula(c)
	}
	switch int(unit(c.seed, 40)*4) % 4 {
	case 1:
		spaceCrescent(c)
	case 2:
		spaceGiant(c)
	case 3:
		spaceMoonlit(c)
	default:
		spaceRinged(c)
	}
	// the odd bright star with a cross flare
	if c.p.dither(c.seed+47) > 0.9985 {
		c.set(colorOf("white"))
	}
}

// nebula is a faint diagonal haze behind the subject. It is a smooth blend, not
// a dither: speckling the sky with single lighter cells reads as sensor noise,
// while a soft band reads as dust.
func nebula(c *sceneCtx) {
	p := c.p
	d := (p.sv - (0.58 - 0.45*p.su)) / 0.30
	fall := math.Exp(-d * d)
	if fall < 0.20 {
		return
	}
	c.blend(c.h.litAt(0), 0.10*fall)
}

// disc paints a banded planet with a terminator. light is +1 when the light
// comes from the left, -1 when from the right.
//
// The mood's earth ramp is a silhouette ramp: its first colour is nearly black
// and the rest are saturated, so using it raw gives a planet striped in two
// unrelated hues. Mixing the ramp's own steps against each other keeps the
// bands in one family, which is what a gas giant looks like.
func disc(c *sceneCtx, cx, cy, r, light float64) {
	p, h := c.p, c.h
	dx, dy := (p.su-cx)/r, (p.sv-cy)/r
	if dx*dx+dy*dy > 1 {
		return
	}
	deep := mix3(h.earthAt(1), h.dark, 0.35)
	mid := mix3(h.earthAt(1), h.earthAt(2), 0.55)
	lit := mix3(h.earthAt(2), colorOf("white"), 0.34)
	if c.toy {
		// two saturated brick colours, alternating. Stepping one hue into
		// three greys gave a beach ball; two colours out of the box gives the
		// striped gas giant a child draws.
		ids := []string{"coral", "dazure", "lorange", "lime", "purple", "lyellow", "red", "mblue"}
		base := colorOf(ids[c.seed%uint32(len(ids))])
		other := colorOf(ids[(c.seed/3+3)%uint32(len(ids))])
		lit = mix3(base, colorOf("white"), 0.30)
		mid = base
		deep = other
	}

	bands := 3 + float64(c.seed%4)
	phase := float64(c.seed%11) * 0.37
	t := clamp01((p.sv - (cy - r)) / (2 * r))
	b := math.Sin(t*math.Pi*bands + phase)
	c.set(lit)
	switch {
	case b > 0.55:
		c.set(mid)
	case b < -0.55:
		c.set(deep)
	}
	// terminator: the far limb falls into shadow
	lx := (p.su - cx) / r * light
	if lx > 0.35 {
		c.shadow(0.30)
	}
	if lx > 0.68 {
		c.shadow(0.55)
	}
	// a polar cap. Only on a planet small enough to show its whole pole: on a
	// cropped close-up the cap lands in the middle of the picture and turns the
	// disc into a snow-capped hill.
	if t < 0.10 && r < 0.36 {
		c.set(colorOf("white"))
	}
}

// ringTone is the pale dust of a ring: light enough to separate from the sky,
// grey enough not to compete with the planet's bands.
func ringTone(c *sceneCtx) [3]float64 {
	if c.toy {
		// a ring is rock and ice, and in a child's set that is sand colour.
		// The mood ramp gave a grey wire loop.
		return mix3(colorOf("tan"), colorOf("white"), 0.35)
	}
	return mix3(c.h.earthAt(1), colorOf("white"), 0.66)
}

// ringQ returns the cell's position in the ring's own flattened coordinate
// system (1 = the planet's limb) and whether it sits on the near half, which
// has to be painted after the planet. tilt rotates the ring in the picture
// plane, squash flattens it into the planet's equatorial plane.
func ringQ(c *sceneCtx, cx, cy, r, tilt, squash float64) (float64, bool) {
	p := c.p
	ca, sa := math.Cos(tilt), math.Sin(tilt)
	ru := (p.su-cx)*ca + (p.sv-cy)*sa
	rv := -(p.su-cx)*sa + (p.sv-cy)*ca
	return math.Hypot(ru/(r*1.72), rv/(r*1.72*squash)), rv > 0
}

// ring paints a wide dust band with a Cassini gap; the disc goes on top of its
// far half, which is what sells the ring as going around the planet.
func ring(c *sceneCtx, cx, cy, r, tilt, squash float64) {
	q, near := ringQ(c, cx, cy, r, tilt, squash)
	in := 1.24 + 0.12*unit(c.seed, 44)
	const width = 0.46
	if q < in || q > in+width {
		return
	}
	f := (q - in) / width
	tone := ringTone(c)
	if f > 0.56 && f < 0.72 { // the gap: dust thins out to near sky
		tone = mix3(c.h.earthAt(2), c.h.dark, 0.40)
	}
	c.set(tone)
	if !near {
		c.shadow(0.45)
	}
}

func moon(c *sceneCtx, x, y, r float64) {
	if !c.p.at(x, y, r, r) {
		return
	}
	c.set(mix3(c.h.earthAt(1), colorOf("white"), 0.62))
	c.shadow(0.22)
	if c.p.at(x-r*0.35, y-r*0.30, r*0.34, r*0.34) {
		c.lift(0.45)
	}
}

func spaceRinged(c *sceneCtx) {
	cx := 0.44 + 0.12*unit(c.seed, 41)
	cy := 0.47 + 0.08*unit(c.seed, 42)
	r := 0.19 + 0.07*unit(c.seed, 43)
	tilt := -0.32 + 0.55*unit(c.seed, 45)
	squash := 0.26 + 0.26*unit(c.seed, 46)
	ring(c, cx, cy, r, tilt, squash)
	// moons stay clear of the disc: a grey ball half-buried in the limb reads
	// as a lump on the planet rather than as a moon
	if mx, my, mr := 0.13+0.10*unit(c.seed, 51), 0.15+0.12*unit(c.seed, 52), 0.032+0.014*unit(c.seed, 53); clearOf(mx, my, mr, cx, cy, r) {
		moon(c, mx, my, mr)
	}
	if mx, my, mr := 0.84+0.08*unit(c.seed, 55), 0.74+0.12*unit(c.seed, 56), 0.026+0.012*unit(c.seed, 57); clearOf(mx, my, mr, cx, cy, r) {
		moon(c, mx, my, mr)
	}
	disc(c, cx, cy, r, 1)
}

// clearOf reports whether a moon at (x,y) keeps its distance from a planet, so
// the two shapes never intersect.
func clearOf(x, y, r, pcx, pcy, pr float64) bool {
	return math.Hypot(x-pcx, y-pcy) > pr+r*2.2
}

func spaceCrescent(c *sceneCtx) {
	cx := 0.34 + 0.30*unit(c.seed, 41)
	cy := 0.44 + 0.10*unit(c.seed, 42)
	r := 0.25 + 0.09*unit(c.seed, 43)
	// a thin lit limb on the dark side keeps the silhouette readable. Skipped in
	// the toy style: a 3%-wider ring is one cell of dashed white outline, which
	// reads as a drawing artifact rather than as a lit edge.
	if !c.toy && c.p.at(cx, cy, r*1.03, r*1.03) {
		c.set(mix3(c.h.earthAt(0), colorOf("white"), 0.72))
	}
	disc(c, cx, cy, r, 1-2*float64(c.seed%2))
	for i := 0; i < 3; i++ {
		x := 0.10 + 0.80*unit(c.seed, uint32(70+i))
		y := 0.12 + 0.40*unit(c.seed, uint32(80+i))
		mr := 0.020 + 0.018*unit(c.seed, uint32(90+i))
		if clearOf(x, y, mr, cx, cy, r) {
			moon(c, x, y, mr)
		}
	}
}

func spaceGiant(c *sceneCtx) {
	// a close-up: the disc is wider than the frame and runs off the bottom,
	// which reads as scale rather than as a sticker in the middle. No ring
	// here - a ring at this size turns into a line across the picture.
	cx := 0.30 + 0.40*unit(c.seed, 41)
	cy := 0.86 + 0.24*unit(c.seed, 42)
	r := 0.42 + 0.14*unit(c.seed, 43)
	disc(c, cx, cy, r, 1)
	for i := 0; i < 2; i++ {
		moon(c, 0.12+0.76*unit(c.seed, uint32(96+i)), 0.10+0.20*unit(c.seed, uint32(99+i)), 0.018+0.014*unit(c.seed, uint32(102+i)))
	}
}

func spaceMoonlit(c *sceneCtx) {
	// a small planet high up, one big moon crowding the corner
	disc(c, 0.28+0.20*unit(c.seed, 41), 0.26+0.10*unit(c.seed, 42), 0.12+0.06*unit(c.seed, 43), 1)
	mx := 0.66 + 0.22*unit(c.seed, 44)
	my := 0.72 + 0.20*unit(c.seed, 45)
	mr := 0.20 + 0.12*unit(c.seed, 46)
	if c.p.at(mx, my, mr, mr) {
		c.set(mix3(c.h.earthAt(1), colorOf("white"), 0.55))
		c.shadow(0.50)
		// the lit limb, and a few craters so the disc is not a flat blob.
		// The toy style wants fixed craters - dither speckle reads as dirt.
		if !c.p.at(mx-mr*0.22, my+mr*0.22, mr*0.80, mr*0.80) {
			c.lift(0.40)
		}
		if c.toy {
			if c.p.at(mx+mr*0.30, my-mr*0.20, mr*0.16, mr*0.16) ||
				c.p.at(mx-mr*0.25, my+mr*0.35, mr*0.12, mr*0.12) {
				c.shadow(0.35)
			}
		} else if c.p.dither(c.seed+33) > 0.94 {
			c.shadow(0.35)
		}
	}
	moon(c, 0.16+0.20*unit(c.seed, 47), 0.62+0.24*unit(c.seed, 48), 0.030+0.016*unit(c.seed, 49))
}

// ---------------------------------------------------------------- portrait

// drawPortrait paints a front-on bust against the sky: a solid silhouette, a
// warm rim light down one side, a halo behind the head.
//
// Three attempts led here. A lit face with eyes and a mouth read as a mask at 64
// cells; a face with features snapped to the cell grid read as two black dots
// and a bar; a profile with a modelled nose read as an egg with a highlight,
// because a nose is one cell and one cell of shape does not survive. What does
// survive is outline plus light: the bust is one solid colour, and the rim tells
// you which way it is lit.
func drawPortrait(c *sceneCtx) {
	p := c.p
	if c.toy {
		portraitToy(c)
		return
	}
	cx := 0.50 + 0.02*(unit(c.seed, 51)-0.5)
	hy := 0.40 + 0.02*unit(c.seed, 52)
	hr := 0.140 + 0.018*unit(c.seed, 53)

	ink := mix3(colorOf("black"), c.h.earthAt(1), 0.10)
	hair := mix3(ink, c.h.dark, 0.30)
	cloth := mix3(ink, c.h.earthAt(1), 0.45)
	rim := c.h.warmAt(0)

	// hair: a slightly wider mass than the skull, with a lower edge at the nape,
	// so the head is not a perfect circle.
	//
	// These are plain functions of (su, sv), not painter methods: the rim pass
	// has to ask about neighbouring points, and p.at always answers for the cell
	// being painted.
	ell := func(su, sv, ecx, ecy, rx, ry float64) bool {
		dx, dy := (su-ecx)/rx, (sv-ecy)/ry
		return dx*dx+dy*dy <= 1
	}
	long := c.seed%2 == 0
	head := func(su, sv float64) bool {
		if ell(su, sv, cx, hy, hr*0.92, hr*1.06) {
			return true
		}
		if ell(su, sv, cx, hy-hr*0.22, hr*1.10, hr*0.94) && sv < hy-hr*0.10 {
			return true
		}
		// long hair falls past the jaw on both sides
		return long && math.Abs(su-cx) < hr*1.02 && sv > hy && sv < hy+hr*1.35
	}
	body := func(su, sv float64) bool {
		if sv > hy+hr*0.90 && sv < hy+hr*1.50 && math.Abs(su-cx) < hr*0.44 {
			return true // neck
		}
		return sv > hy+hr*1.35 && inRoundRect(su, sv, cx-0.30, hy+hr*1.35, cx+0.30, hy+hr*4.2, hr*0.85)
	}
	inside := func(su, sv float64) bool { return head(su, sv) || body(su, sv) }

	// halo behind the head
	if d := math.Hypot((p.su-cx)/0.40, (p.sv-hy)/0.42); d < 1 {
		c.blend(c.h.warmAt(1), 0.18*(1-d))
	}
	if !inside(p.su, p.sv) {
		return
	}

	c.set(cloth)
	if head(p.su, p.sv) {
		c.set(ink)
		if ell(p.su, p.sv, cx, hy-hr*0.22, hr*1.10, hr*0.94) || (long && math.Abs(p.su-cx) < hr*1.02 && p.sv < hy+hr*1.35) {
			c.set(hair)
		}
	}

	// rim light: one side only, sampled along the light direction. A purely
	// horizontal probe only catches near-vertical edges - the crown is nearly
	// horizontal, so it stayed dark and the rim read as a stick down the neck.
	step := p.hair() * 1.15
	edge := false
	for _, d := range [][2]float64{{-1, 0}, {-1, -1}, {0, -1}, {-1, 1}} {
		if !inside(p.su+d[0]*step, p.sv+d[1]*step) {
			edge = true
			break
		}
	}
	// only the head is rimmed: the shoulder's top edge is horizontal, and a
	// rim there draws a bright bar across the bust
	if edge && head(p.su, p.sv) {
		c.set(rim)
	}
}

// portraitToy paints a front-on smiley face: big head, hair, two round eyes,
// a red smile and rosy cheeks.
//
// The poster portrait is a silhouette with a rim light, which at toy size is a
// shadow person with a dotted yellow edge - the opposite of inviting. A face
// with two eyes and a smile is the one thing a child can always read.
func portraitToy(c *sceneCtx) {
	p := c.p
	cx, cy := 0.50, 0.44
	rx, ry := 0.215, 0.235

	skins := []string{"nougat", "tan", "dtan", "dbrown", "lgray"}
	skin := colorOf(skins[c.seed%uint32(len(skins))])
	hairs := []string{"black", "dbrown", "dtan"}
	hair := colorOf(hairs[(c.seed/3)%uint32(len(hairs))])
	// the shirt is the one place a portrait can carry a strong colour, and the
	// mood accent is the same yellow in half the palettes. It also has to stay
	// darker than the pastel card behind it, or the shoulders dissolve.
	shirt := colorOf([]string{"red", "dazure", "purple", "dteal", "orange", "coral"}[(c.seed/5)%6])
	ink := mix3(colorOf("black"), hair, 0.10)

	// shoulders, then neck, then head: the head has to sit in front
	if p.round(cx-0.36, cy+ry*0.86, cx+0.36, 1.30, 0.14) {
		c.set(shirt)
	}
	if p.box(cx-0.062, cy+ry*0.52, cx+0.062, cy+ry*0.96) {
		c.set(mix3(skin, hair, 0.30))
	}
	for _, s := range []float64{-1, 1} {
		if p.at(cx+s*rx*0.97, cy+ry*0.10, 0.036, 0.048) {
			c.set(skin)
		}
	}
	if p.at(cx, cy, rx, ry) {
		c.set(skin)
	}

	// hair: a bowl-cut cap over the crown, cut straight across the brow. Locks
	// with gaps between them read as fangs on a pale face, which is exactly
	// what the first version did.
	if p.at(cx, cy-ry*0.44, rx*1.04, ry*0.74) && p.sv < cy-ry*0.24 {
		c.set(hair)
	}
	// long hair, on every other seed
	if c.seed%2 == 0 {
		if p.box(cx-rx*1.08, cy-ry*0.44, cx-rx*0.74, cy+ry*1.10) ||
			p.box(cx+rx*0.74, cy-ry*0.44, cx+rx*1.08, cy+ry*1.10) {
			c.set(hair)
		}
	}

	// eyes, with one highlight cell each
	for _, s := range []float64{-1, 1} {
		ex, ey := cx+s*rx*0.36, cy-ry*0.02
		if p.at(ex, ey, 0.031, 0.036) {
			c.set(ink)
		}
		if p.at(ex-0.012, ey-0.015, 0.013, 0.013) {
			c.set(colorOf("white"))
		}
	}
	// smile: the lower half of a ring, one cell thick
	if p.sv > cy+ry*0.16 {
		if d := math.Hypot(p.su-cx, p.sv-(cy+ry*0.14)); d > 0.086 && d < 0.102 {
			c.set(mix3(colorOf("darkred"), ink, 0.25))
		}
	}
	// cheeks
	for _, s := range []float64{-1, 1} {
		if p.at(cx+s*rx*0.64, cy+ry*0.30, 0.038, 0.030) {
			c.set(mix3(skin, colorOf("red"), 0.32))
		}
	}
}
