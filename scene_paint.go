package main

import "math"

// sceneShade returns the per-cell colour function for one scene.
//
// Every picture is painted in the same four passes, in this order:
//
//	backdrop   sky band, sun or moon, stars
//	ground     floor, water, horizon line - the stage the subject sits on
//	subject    the category painter: the thing the name promises
//	air        the light pass on top: haze, snow, bloom
//
// Each pass only ever writes palette colours, so the painting is already an
// exact mosaic before the quantiser sees it. That is what keeps generated
// artwork clean at any grid size: there is no blend between two bricks for the
// quantiser to resolve into a muddy third colour.
func sceneShade(sc scene) func(x, y, w, h float64) [3]float64 {
	cell := sceneCell(sc)
	return outlineWrap(cell)
}

// sceneCell paints one cell and reports whether the subject pass touched it.
//
// The flag is what the outline pass keys on: a brick-art picture outlines the
// shape, not every colour change inside it, so the recipe has to say where the
// subject ends. Ground and sky are painted before the recipe runs, so anything
// the recipe writes is subject by definition.
func sceneCell(sc scene) func(x, y, w, h float64) ([3]float64, bool) {
	cat := sceneCat(sc.ID)
	recipe := categories[cat].pick(sc.ID)
	m := moods[sc.paletteMood()]
	if toySky, ok := toyMoods[m.name]; ok {
		m.sky = toySky.sky
		m.earth = toySky.earth
	}
	hue := newHue(m)
	seed := uint32(sc.ID)*2654435761 ^ 0x9e3779b9

	return func(x, y, w, hh float64) ([3]float64, bool) {
		p := painter{
			cols: int(math.Round(w)),
			rows: int(math.Round(hh)),
			w:    w,
			h:    hh,
		}
		p.ix = int(x - 0.5)
		p.iy = int(y - 0.5)
		p.u = x / w
		p.v = y / hh
		p.frame = frameFor(aspectOf(w, hh))
		// recipes work in the design box; p.su/p.sv are that view of the cell
		p.su, p.sv = p.frame.toScene(p.u, p.v)

		ctx := sceneCtx{p: &p, h: hue, seed: seed, cat: cat, id: sc.ID, toy: true, kid: true}
		col := ctx.paint(recipe)
		return col, ctx.subject
	}
}

// toyMoods is the children's-set palette table: for every mood it gives the
// sky and the ground a child would actually choose.
//
// The poster moods were tuned for atmosphere, and atmosphere is the opposite
// of a toy. "海洋" put a sand-blue zenith over a teal field, which is a lovely
// watercolour and a terrible brick pattern: a flower pot ended up standing on
// open water. Here the sky keeps the mood's identity (day, sunset, night, snow)
// and the ground is always a ground.
var toyMoods = map[string]struct{ sky, earth []string }{
	"晴日": {[]string{"azure", "lblue"}, []string{"bgreen", "green", "lime", "dgreen"}},
	"日落": {[]string{"orange", "lyellow"}, []string{"bgreen", "green", "dgreen", "olive"}},
	"暮色": {[]string{"dpurple", "coral"}, []string{"dgreen", "green", "olive", "dbrown"}},
	"深夜": {[]string{"dblue", "black"}, []string{"dgreen", "dgreen", "black"}},
	"雪原": {[]string{"lblue", "white"}, []string{"white", "lblue", "lgray"}},
	"樱花": {[]string{"pink", "white"}, []string{"bgreen", "lime", "green"}},
	"海洋": {[]string{"azure", "lblue"}, []string{"bgreen", "green", "lime"}},
	"秋林": {[]string{"lorange", "lyellow"}, []string{"olive", "dgreen", "dtan", "redbrown"}},
	"霓虹": {[]string{"black", "dpurple"}, []string{"dgray", "black", "dblue"}},
	"沙漠": {[]string{"lyellow", "tan"}, []string{"tan", "dtan", "dorange"}},
	"极地": {[]string{"dazure", "lblue"}, []string{"white", "lblue", "sandblue"}},
	"苔原": {[]string{"lblue", "vlgray"}, []string{"olive", "sandgreen", "dgreen"}},
}

// groundKind says what the stage under the subject is. Colour alone was not
// enough: a potted flower needs a floor and a boat needs water, and the mood
// cannot know which.
type groundKind int

const (
	groundGrass groundKind = iota
	groundSnow
	groundSand
	groundRock
	groundWater
	groundRoad
)

func (c *sceneCtx) groundKind() groundKind {
	name := c.h.m.name
	snow := name == "雪原" || name == "极地"
	sand := name == "沙漠"
	switch c.cat {
	case catCity:
		return groundRoad
	case catVehicle:
		if isCarScene(c) {
			return groundRoad
		}
		// a boat needs water in every mood: a snow-covered sea stranded the
		// hull on a white band that read as a beach in the middle of the frame
		return groundWater
	case catSpace, catPortrait:
		return groundGrass
	}
	switch {
	case snow:
		return groundSnow
	case sand:
		return groundSand
	case name == "秋林":
		return groundRock
	}
	return groundGrass
}

// outlineWrap paints the whole grid once, then rims the subject with one cell
// of dark brick.
//
// A flat fill with no edge is the difference between "a brick mosaic" and "a
// vector sticker": the rim is what every printed brick pattern uses to make a
// shape pop off its background at thumbnail size, and it also hides the ragged
// one-cell staircase along every diagonal.
func outlineWrap(cell func(x, y, w, h float64) ([3]float64, bool)) func(x, y, w, h float64) [3]float64 {
	var (
		cols, rows int
		col        [][3]float64
		in         []bool
	)
	build := func(c, r int) {
		cols, rows = c, r
		col = make([][3]float64, cols*rows)
		in = make([]bool, cols*rows)
		for iy := 0; iy < rows; iy++ {
			for ix := 0; ix < cols; ix++ {
				cc, ss := cell(float64(ix)+0.5, float64(iy)+0.5, float64(cols), float64(rows))
				col[iy*cols+ix] = cc
				in[iy*cols+ix] = ss
			}
		}
		// rim inside the silhouette, so the shape keeps its size
		for iy := 0; iy < rows; iy++ {
			for ix := 0; ix < cols; ix++ {
				i := iy*cols + ix
				if !in[i] {
					continue
				}
				edge := false
				for _, d := range [4][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
					jx, jy := ix+d[0], iy+d[1]
					if jx < 0 || jy < 0 || jx >= cols || jy >= rows {
						continue // the picture crops the shape: no line there
					}
					if !in[jy*cols+jx] {
						edge = true
						break
					}
				}
				if edge {
					col[i] = mix3(col[i], colorOf("black"), 0.66)
				}
			}
		}
	}

	return func(x, y, w, h float64) [3]float64 {
		c, r := int(math.Round(w)), int(math.Round(h))
		if col == nil || cols != c || rows != r {
			build(c, r)
		}
		ix, iy := int(x-0.5), int(y-0.5)
		if ix < 0 || iy < 0 || ix >= cols || iy >= rows {
			// off-grid sample (odd upscales): fall back to a direct paint
			v, _ := cell(x, y, w, h)
			return v
		}
		return col[iy*cols+ix]
	}
}

// sceneCtx carries everything a recipe needs for one cell.
type sceneCtx struct {
	p    *painter
	h    *hue
	seed uint32
	cat  int
	id   int
	// toy is the children's-set style: flat colour bands, chunky subjects,
	// no grain. Gradients, speckle noise and haze are what made the field
	// read as a printed poster rather than as something a child can build.
	toy bool
	// canvas is the colour being built up, already in palette space.
	canvas [3]float64
	// subject reports that the recipe wrote this cell, which is what the
	// outline pass rims.
	subject bool
	// kid switches the whole picture to the sticker card: a character on a
	// flat background, drawn from characters.go.
	kid bool
	// inSubject is true while the recipe is running.
	inSubject bool
}

// sceneZoom is how much the subject is enlarged in the children's-set style.
// A subject that occupies the middle third of the frame is a poster; one that
// runs off the sides is a brick pattern.
const sceneZoom = 1.18

func (c *sceneCtx) paint(recipe drawFn) [3]float64 {
	if c.kid {
		// the sticker card: one flat colour, one ground strip. No sun, no
		// clouds, no horizon haze - the character is the whole picture.
		c.kidStage()
	} else {
		c.canvas = c.backdrop()
		c.ground()
	}
	if recipe != nil {
		su, sv := c.p.su, c.p.sv
		if c.toy && !c.kid {
			// scale about the ground line, so feet stay planted and it is the
			// empty sky that gets cropped
			anchor := c.horizon()
			if c.cat == catSpace || c.cat == catPortrait {
				anchor = 0.52
			}
			c.p.su = 0.5 + (su-0.5)*sceneZoom
			c.p.sv = anchor + (sv-anchor)*sceneZoom
		}
		c.inSubject = true
		recipe(c)
		c.inSubject = false
		c.p.su, c.p.sv = su, sv
	}
	if !c.toy {
		c.air()
	}
	return c.canvas
}

// set paints a brick colour.
func (c *sceneCtx) set(col [3]float64) {
	c.canvas = col
	c.touch()
}

// touch records that the subject pass wrote this cell.
func (c *sceneCtx) touch() {
	if c.inSubject {
		c.subject = true
	}
}

// blend tints the canvas. Reserved for glass, water, haze and light bloom,
// where a tint is the point; solids should use set.
func (c *sceneCtx) blend(col [3]float64, t float64) {
	c.canvas = mix3(c.canvas, col, clamp01(t))
	c.touch()
}

// shadow tints towards the mood's silhouette colour.
func (c *sceneCtx) shadow(t float64) {
	c.canvas = mix3(c.canvas, c.h.dark, clamp01(t))
	c.touch()
}

// lift tints towards white: snow, foam, a highlight.
func (c *sceneCtx) lift(t float64) {
	c.canvas = mix3(c.canvas, colorOf("white"), clamp01(t))
	c.touch()
}

// ---------------------------------------------------------------- backdrop

func (c *sceneCtx) backdrop() [3]float64 {
	p, h := c.p, c.h
	if c.toy {
		return c.toyBackdrop()
	}
	if h.night {
		col := rampAt(h.skyRamp(), clamp01(p.sv*1.06))
		col = c.stars(col)
		return c.nightLight(col)
	}
	if h.m.name == "雪原" || h.m.name == "苔原" {
		// overcast: two flat bands read as cloud without needing a texture
		col := rampAt(h.skyRamp(), clamp01(p.sv*0.92))
		if p.sv < 0.34 && p.jitter(7, c.seed+31) > 0.62 {
			col = h.skyAt(1) // a flat cloud bank
		}
		if p.sv > 0.30 && p.sv < 0.315 {
			col = h.skyAt(len(h.sky) - 1) // its underside line
		}
		return col
	}
	col := rampAt(h.skyRamp(), clamp01(p.sv*1.14))
	return c.sun(col)
}

// sun draws a flat disc and a couple of halo rings, placed off-centre so the
// composition is never symmetric.
func (c *sceneCtx) sun(col [3]float64) [3]float64 {
	p, h := c.p, c.h
	sx := 0.20 + 0.60*unit(c.seed, 11)
	sy := 0.12 + 0.16*unit(c.seed, 12)
	r := 0.045 + 0.020*unit(c.seed, 13)

	if d := math.Hypot(p.su-sx, p.sv-sy); d > r && d < r*1.9 {
		col = layer(col, h.warmAt(1), 0.26)
	}
	if math.Hypot(p.su-sx, p.sv-sy) <= r {
		col = h.warmAt(0)
	}
	return col
}

// toyBackdrop is the children's-set sky: two or three flat colour bands, one
// big flat sun, and one or two puffy clouds. No gradient, no halo, no haze -
// a band the child can fill in one colour is the whole point.
func (c *sceneCtx) toyBackdrop() [3]float64 {
	p, h := c.p, c.h
	// a portrait is always a bright card. The night palettes put the face in a
	// black frame, which reads as a mugshot rather than as a portrait.
	if c.cat == catPortrait {
		ids := []string{"lblue", "pink", "lyellow", "sandgreen", "lavender"}
		bg := mix3(colorOf(ids[c.seed%uint32(len(ids))]), colorOf("white"), 0.34)
		if p.sv > 0.66 {
			bg = mix3(bg, colorOf("white"), 0.40)
		}
		return bg
	}
	// One flat sky of one real brick colour. A ramp between azure and white
	// gives stripes of washing-powder blue, and stepping that ramp still gave
	// three bands of in-between colour: the sky is the place where a wrong
	// colour is loudest, so it is a single swatch.
	col := h.skyAt(0)
	// space and portraits use one flat sky: a band boundary across the middle
	// of the frame cuts a planet in half and reads as a horizon on an empty
	// black field
	if c.cat == catSpace || c.cat == catPortrait {
		if h.night {
			return c.stars(col)
		}
		return col
	}
	// one flat sky. A second band was tried twice - full height and as a strip
	// above the horizon - and both read as a wall behind the subject rather
	// than as distance. A brick pattern has a sky and a ground, and the seam
	// between them is the horizon.

	if h.night {
		col = c.stars(col)
		return c.nightLight(col)
	}
	// overcast snow moods: one flat cloud bank with a hard edge. The 7-cell
	// dither blocks the poster style used read as a broken texture.
	if h.m.name == "雪原" || h.m.name == "苔原" {
		return col
	}
	// a potted flower owns the middle of the frame, where the sun would land
	if c.cat == catBotanical {
		return c.toyClouds(col)
	}
	return c.toySun(c.toyClouds(col))
}

// toySun paints one big flat disc. It used to carry a chunky halo ring, but a
// ring the same hue as the disc reads as a doughnut floating in the sky; a
// child's sticker sun is a plain circle.
func (c *sceneCtx) toySun(col [3]float64) [3]float64 {
	p, h := c.p, c.h
	sx := 0.18 + 0.64*unit(c.seed, 11)
	sy := 0.10 + 0.14*unit(c.seed, 12)
	r := 0.080 + 0.025*unit(c.seed, 13)
	if math.Hypot(p.su-sx, p.sv-sy) <= r {
		return h.warmAt(0)
	}
	return col
}

// toyClouds lays one or two clouds built from three flat ellipses each.
//
// They live in the top corners only: a cloud drifting over the middle of the
// frame lands on the subject's head, and a cloud that touches an ear is the
// single most "generated-looking" thing in the sheet.
func (c *sceneCtx) toyClouds(col [3]float64) [3]float64 {
	p, h := c.p, c.h
	// a cloud is white brick, not a tint of the sky - and on a white sky it is
	// not a cloud at all, so it simply is not drawn
	if luma(h.skyAt(0)) > 208 {
		return col
	}
	cloudCol := colorOf("white")
	n := 1 + int(c.seed%2)
	for i := 0; i < n; i++ {
		cx := 0.14 + 0.14*unit(c.seed, uint32(61+i))
		if i == 1 || (n == 1 && c.seed%2 == 1) {
			cx = 0.72 + 0.14*unit(c.seed, uint32(61+i))
		}
		cy := 0.09 + 0.07*unit(c.seed, uint32(71+i))
		// three lobes: a big one between two smaller ones
		if p.at(cx, cy, 0.050, 0.026) ||
			p.at(cx-0.044, cy+0.011, 0.030, 0.018) ||
			p.at(cx+0.044, cy+0.011, 0.032, 0.018) {
			return cloudCol
		}
	}
	return col
}

// stars is a sparse field that thins out towards the horizon. The densities are
// deliberately low: a night sky reads as empty, and at 64 cells a 2% fill turns
// the background into confetti.
func (c *sceneCtx) stars(col [3]float64) [3]float64 {
	p := c.p
	if p.sv > 0.46 {
		return col
	}
	// stars thin out as they approach the horizon
	if p.sv > 0.30 && p.dither(c.seed+7) < (p.sv-0.30)/0.16 {
		return col
	}
	switch {
	case p.dither(c.seed+5) > 0.9992:
		return colorOf("white") // the occasional bright one
	case p.dither(c.seed+3) > 0.9955:
		if p.dither(c.seed+4) > 0.90 {
			return colorOf("white")
		}
		return c.h.litAt(0)
	}
	return col
}

// nightLight is the moon (or, for the neon mood, a wash of accent light).
func (c *sceneCtx) nightLight(col [3]float64) [3]float64 {
	p, h := c.p, c.h
	if h.m.name == "霓虹" {
		glow := math.Hypot((p.su-0.5)/0.70, (p.sv-0.40)/0.50)
		if glow < 1 {
			return layer(col, h.warmAt(0), 0.18*(1-glow))
		}
		return col
	}
	// space scenes paint their own moons, and a portrait already has a halo
	// behind the head - a third light in the corner just competes
	if c.cat == catSpace || c.cat == catPortrait {
		return col
	}
	// keep the moon out of the middle - over the subject it reads as a hat - and
	// inside the visible window: on a tall grid the frame crops the design box,
	// so a moon placed at 0.12 of the box lands outside the picture and shows up
	// as a smear in the corner.
	r := 0.062 + 0.020*unit(c.seed, 23)
	fw := p.frame.x1 - p.frame.x0
	if fw <= 0 {
		fw = 1
	}
	mx := p.frame.x0 + fw*(0.13+0.14*unit(c.seed, 21))
	if c.seed%2 == 0 {
		mx = p.frame.x0 + fw*(0.73+0.14*unit(c.seed, 21))
	}
	my := 0.10 + 0.08*unit(c.seed, 22)
	if p.at(mx, my, r, r) {
		// a full disc. A crescent at this size is five cells of white with a
		// bite out of it, which reads as a stray mark, a hook or a bird.
		// One grey cell reads as a crater and keeps the disc from being a dot.
		if p.at(mx+r*0.32, my+r*0.26, r*0.26, r*0.24) {
			return h.skyAt(len(h.sky) - 1)
		}
		return colorOf("white")
	}
	return col
}

// ------------------------------------------------------------------ ground

// ground lays the stage: horizon, water or floor, a vignette at the bottom.
func (c *sceneCtx) ground() {
	p := c.p
	switch c.cat {
	case catSpace, catPortrait:
		return // no ground plane in these compositions
	}
	horizon := c.horizon()
	if p.sv <= horizon {
		return
	}
	depth := clamp01((p.sv - horizon) / (1 - horizon))

	if c.toy {
		c.toyGround(depth)
		return
	}

	switch c.cat {
	case catVehicle:
		c.water(depth, horizon)
	case catCity:
		c.street(depth)
	case catStructure, catBotanical, catFood, catAnimal:
		c.floor(depth, horizon)
	default:
		c.plain(depth, horizon)
	}

	if depth > 0.88 && p.dither(c.seed+51) > 0.55 {
		c.shadow(0.35)
	}
}

// toyGround is the children's-set stage: two bands of real brick colours with
// a hard edge between them, and nothing in between.
//
// Both bands used to be interpolated positions on the earth ramp, which meant
// the grass came out as three stripes of in-between green that no LEGO set
// actually contains. Picking two swatches gives a field the child builds with
// two lots of bricks and can count.
func (c *sceneCtx) toyGround(depth float64) {
	far := depth < 0.52
	switch c.groundKind() {
	case groundRoad:
		if far {
			c.set(colorOf("lgray"))
		} else {
			c.set(colorOf("dgray"))
		}
	case groundWater:
		if far {
			c.set(colorOf("azure"))
		} else {
			c.set(colorOf("dteal"))
		}
	case groundSnow:
		if far {
			c.set(colorOf("white"))
		} else {
			c.set(colorOf("lblue"))
		}
	case groundSand:
		if far {
			c.set(colorOf("tan"))
		} else {
			c.set(colorOf("dtan"))
		}
	case groundRock:
		if far {
			c.set(colorOf("olive"))
		} else {
			c.set(colorOf("dgreen"))
		}
	default: // grass
		if far {
			c.set(colorOf("bgreen"))
		} else {
			c.set(colorOf("green"))
		}
	}
}

// horizon is the waterline / floor line for this composition. Portrait and
// space scenes have none, which is why they returned early.
func (c *sceneCtx) horizon() float64 {
	if c.kid {
		return kidFoot
	}
	switch c.cat {
	case catCity:
		return 0.78 + 0.06*unit(c.seed, 63)
	case catVehicle:
		return 0.62 + 0.08*unit(c.seed, 62)
	case catStructure:
		return 0.74 + 0.07*unit(c.seed, 64)
	case catBotanical, catFood:
		return 0.70 + 0.08*unit(c.seed, 65)
	case catAnimal:
		return 0.66 + 0.08*unit(c.seed, 66)
	default:
		return 0.70 + 0.06*unit(c.seed, 61)
	}
}

// water is a reflective plane: darker with depth, with horizontal glints.
func (c *sceneCtx) water(depth, horizon float64) {
	p, h := c.p, c.h
	if p.sv < horizon+p.hairV()*1.5 {
		c.set(h.earthAt(0)) // the waterline itself
		return
	}
	col := ramp(h.earth, 0.95-0.72*depth)
	// the far water mirrors the sky, the near water is the mood's own colour
	col = mix3(col, h.skyAt(len(h.sky)-1), 0.26*(1-depth)*(1-depth))
	c.set(col)
	band := uint32(depth * 12)
	if p.dither(c.seed+7+band) > 0.90 && p.iy%2 == 0 {
		c.blend(h.litAt(0), 0.30)
	}
}

// floor is a studio sweep or tabletop: a plane that darkens towards the front
// and catches a little sky light at the back.
func (c *sceneCtx) floor(depth, horizon float64) {
	h := c.h
	col := ramp(h.earth, 0.78-0.56*depth)
	col = mix3(col, h.skyAt(len(h.sky)-1), 0.14*(1-depth))
	if c.p.jitter(7, c.seed+82) > 0.90 {
		col = mix3(col, h.dark, 0.10) // grain in the surface, not noise
	}
	c.set(col)
}

// street is asphalt with a faint reflection.
func (c *sceneCtx) street(depth float64) {
	h := c.h
	c.set(mix3(h.dark, ramp(h.earth, 0.42-0.30*depth), 0.34))
	if c.p.dither(c.seed+91) > 0.972 {
		c.blend(h.litAt(0), 0.22)
	}
}

// plain is grass, sand, snow or rock depending on the mood.
func (c *sceneCtx) plain(depth, horizon float64) {
	p, h := c.p, c.h
	snowy := h.m.name == "雪原" || h.m.name == "极地"
	if snowy {
		// packed snow: two or three broad drifts. Static noise reads as a
		// broken renderer; a 9-cell clump reads as a drift.
		c.set(ramp(h.earth, 0.30+0.50*depth))
		if p.jitter(9, c.seed+102) > 0.66 {
			c.set(h.earthAt(2))
		}
		if p.jitter(14, c.seed+103) > 0.86 {
			c.set(h.earthAt(3))
		}
		return
	}
	// The plane recedes: at the horizon it is washed towards the sky, at the
	// front it is the deep end of the earth ramp. Two mixes do more for depth
	// than any amount of outline.
	col := ramp(h.earth, 0.80-0.64*depth)
	col = mix3(col, h.skyAt(len(h.sky)-1), 0.22*(1-depth)*(1-depth))
	if p.jitter(6, c.seed+101) > 0.74 {
		col = mix3(col, h.litAt(0), 0.13) // patches of second growth
	}
	c.set(col)
}

// air is the final pass: haze, weather, bloom.
func (c *sceneCtx) air() {
	p, h := c.p, c.h

	// distance haze just above the horizon line. It has to be a smooth blend:
	// gating it on a dither leaves a row of speckles that reads as a rendering
	// fault rather than as air.
	if c.cat != catSpace && c.cat != catPortrait {
		if d := math.Abs(p.sv - c.horizon()); d < 0.045 {
			c.blend(h.skyAt(len(h.sky)-1), 0.22*(1-d/0.045))
		}
	}

	switch h.m.name {
	case "雪原":
		// only the top third: snow drifting into a horizontal band across the
		// whole frame looks like a broken texture, not weather
		if p.sv < 0.35 && p.dither(c.seed+121) > 0.982 {
			c.set(colorOf("white"))
		}
	case "海洋":
		if p.dither(c.seed+122) > 0.988 {
			c.blend(colorOf("white"), 0.55)
		}
	}

	if h.night && p.su > 0.55 && p.sv < 0.35 && p.dither(c.seed+131) > 0.997 {
		c.blend(h.litAt(1), 0.6) // bokeh, top right
	}
}
