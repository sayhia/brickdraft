package main

import "math"

// This file is the children's library: 33 sticker characters drawn from one
// shared kit.
//
// The scene recipes in categories.go paint places - a range, a skyline, a
// lighthouse. A place has no face, and a face is the only thing a three-year-old
// looks for in a picture. That is why the landscape set kept reading as "a nice
// poster, but not for me": there was nothing in it to greet.
//
// So every picture here is one character, centred, with the same proportions:
// a head that is nearly half the height of the frame, two round eyes with a
// highlight, a small U of a mouth, and two blush ovals. The kit below draws
// that face, and each recipe only decides ears, a body and a colour. Ten lines
// per character is enough because the face carries the likeness.

// --------------------------------------------------------------- card layout

// The numbers below are the card. They are constants rather than per-recipe
// parameters because the whole point of the set is that every card has the same
// proportions: a child should be able to put two of them side by side and see
// the same friend at the same size, the way a sticker book works.
const (
	kidCX     = 0.50
	kidHeadY  = 0.375
	kidHeadR  = 0.205
	kidFoot   = 0.790
	kidGround = 0.735
)

// kidStageColours maps a mood to the card and the ground strip under the feet.
//
// Every entry is a brick, never a mix: a card colour that is not a real brick is
// the one thing a mosaic app cannot reproduce, and it reads as a pastel wash
// instead of a build.
var kidStageColours = map[string][2]string{
	"晴日": {"lblue", "bgreen"},
	"日落": {"lorange", "dgreen"},
	"暮色": {"lavender", "green"},
	"深夜": {"dblue", "dgreen"},
	"雪原": {"vlgray", "white"},
	"樱花": {"pink", "bgreen"},
	"海洋": {"lblue", "sandblue"},
	"秋林": {"tan", "olive"},
	"霓虹": {"dpurple", "dgray"},
	"沙漠": {"lyellow", "tan"},
	"极地": {"sandblue", "white"},
	"苔原": {"sandgreen", "olive"},
}

// kidStage paints the card: one flat colour above the ground strip, one below,
// with a slightly darker third band at the very bottom for the feet to stand on.
func (c *sceneCtx) kidStage() {
	p := c.p
	st, ok := kidStageColours[c.h.m.name]
	if !ok {
		st = kidStageColours["晴日"]
	}
	card := colorOf(st[0])
	ground := colorOf(st[1])

	if p.sv < kidGround {
		c.set(card)
		return
	}
	c.set(ground)
	if p.sv > kidGround+0.16 {
		c.set(mix3(ground, colorOf("black"), 0.18))
	}
}

// ------------------------------------------------------------------ shapes

// disc fills an ellipse and reports whether the cell landed inside it, so a
// recipe can say "paint the body, and if you did, paint the patch on top".
func (c *sceneCtx) disc(x, y, rx, ry float64, col [3]float64) bool {
	if c.p.at(x, y, rx, ry) {
		c.set(col)
		return true
	}
	return false
}

func (c *sceneCtx) rect(x0, y0, x1, y1 float64, col [3]float64) bool {
	if c.p.box(x0, y0, x1, y1) {
		c.set(col)
		return true
	}
	return false
}

func (c *sceneCtx) tri3(ax, ay, bx, by, dx, dy float64, col [3]float64) bool {
	if c.p.tri(ax, ay, bx, by, dx, dy) {
		c.set(col)
		return true
	}
	return false
}

// rrect is a rounded rectangle, the workhorse for car bodies and cakes.
func (c *sceneCtx) rrect(x0, y0, x1, y1, r float64, col [3]float64) bool {
	if c.p.round(x0, y0, x1, y1, r) {
		c.set(col)
		return true
	}
	return false
}

// kidTorso lays the body under an animal head.
//
// The body starts inside the head and is nearly as wide as it: a head of radius
// r sitting on a body of 0.66r, the two only just touching, reads as two beads
// on a string - which is exactly how the first rabbit and panda came out. A
// child's animal is one silhouette with a face on it.
func (c *sceneCtx) kidTorso(fur, belly [3]float64, width, bellyW float64) {
	top := kidHeadY + kidHeadR*0.50
	bot := kidFoot + 0.016
	mid := (top + bot) / 2
	ry := (bot - top) / 2
	rx := kidHeadR * width
	c.disc(kidCX, mid, rx, ry, fur)
	if bellyW > 0 {
		c.disc(kidCX, mid+kidHeadR*0.10, rx*bellyW, ry*0.72, belly)
	}
}

// kidNeck draws one darker cell along the underside of the head.
//
// Head and body are the same colour, so without this line they fuse into a
// single egg. The outline pass cannot help: both halves are subject, and the
// pass only rims the outside of the silhouette. One darker cell under the chin
// is what tells the eye where the head ends.
func (c *sceneCtx) kidNeck(cy, r float64, fur [3]float64) {
	y := cy + r*0.96
	if math.Abs(c.p.sv-y) < c.p.hairV()*0.60 && math.Abs(c.p.su-kidCX) < r*0.66 {
		c.set(mix3(fur, colorOf("black"), 0.38))
	}
}

// ------------------------------------------------------------------- faces

// kidFace is the sticker face. Every character owns one; the eyes and the mouth
// are what make a grey blob a friend.
type kidFace struct {
	cx, cy float64
	rx, ry float64
	ink    [3]float64
	white  [3]float64
	blush  [3]float64
	// eyeDX and eyeDY are offsets from the head centre, in head radii, so the
	// same face works on a cat and on a bus.
	eyeDX, eyeDY float64
	eyeR         float64
	// mouthW is half the mouth width; mouthY is the centre of the U.
	mouthW, mouthY float64
	// noBlush is for faces whose cheeks are already a colour of their own.
	noBlush bool
	// noMouth leaves the mouth to the recipe (a beak, a snout, a grille).
	noMouth bool
}

func newFace(cx, cy, rx, ry float64) kidFace {
	return kidFace{
		cx: cx, cy: cy, rx: rx, ry: ry,
		ink:    colorOf("black"),
		white:  colorOf("white"),
		blush:  mix3(colorOf("coral"), colorOf("white"), 0.30),
		eyeDX:  rx * 0.40,
		eyeDY:  -ry * 0.04,
		eyeR:   rx * 0.255,
		mouthW: rx * 0.21,
		mouthY: cy + ry*0.40,
	}
}

func (f kidFace) draw(c *sceneCtx) {
	p := c.p
	for _, s := range []float64{-1, 1} {
		ex, ey := f.cx+s*f.eyeDX, f.cy+f.eyeDY
		if p.at(ex, ey, f.eyeR, f.eyeR*1.18) {
			c.set(f.ink)
			if p.at(ex-f.eyeR*0.26, ey-f.eyeR*0.44, f.eyeR*0.36, f.eyeR*0.36) {
				c.set(f.white)
			}
		}
	}
	if !f.noMouth {
		// a U, one cell thick. Two cells is not a mouth: every face in the
		// first draft wore a black moustache, because each stroke of the U was
		// drawn at +-one full cell, which is two cells wide.
		cell := p.hair() * 0.52
		cellV := p.hairV() * 0.52
		base := f.mouthY + cellV*1.7
		if math.Abs(p.sv-base) < cellV && math.Abs(p.su-f.cx) < f.mouthW+cell {
			c.set(f.ink)
		}
		for _, s := range []float64{-1, 1} {
			if math.Abs(p.su-(f.cx+s*f.mouthW)) < cell && p.sv > f.mouthY && p.sv <= base {
				c.set(f.ink)
			}
		}
	}
	if !f.noBlush {
		for _, s := range []float64{-1, 1} {
			if p.at(f.cx+s*f.rx*0.68, f.cy+f.ry*0.36, f.rx*0.15, f.ry*0.10) {
				c.set(f.blush)
			}
		}
	}
}

// dot is a round highlight used on noses and cherries.
func (c *sceneCtx) dot(x, y, r float64, col [3]float64) {
	if c.p.at(x, y, r, r) {
		c.set(col)
	}
}

// ---------------------------------------------------------------- animals

// charCat is the sticker cat: pointed ears, a body the same colour as the head,
// a pale belly, and a tail that curls off the right hip.
func charCat(c *sceneCtx) {
	p := c.p
	cx, cy, r := kidCX, kidHeadY, kidHeadR
	fur := colorOf([]string{"lorange", "dgray", "white", "nougat", "lgray"}[c.seed%5])
	pale := mix3(fur, colorOf("white"), 0.66)
	if luma(fur) > 225 {
		pale = mix3(fur, colorOf("pink"), 0.35)
	}

	// tail: four fat discs stepping up from the hip, drawn first so the body
	// covers where it starts
	for i := 0; i < 4; i++ {
		t := float64(i) / 3
		tx := cx + r*(0.72+0.30*t)
		ty := kidFoot - r*(0.30+0.85*t)
		c.disc(tx, ty, r*0.13, r*0.13, fur)
	}
	c.kidTorso(fur, pale, 1.14, 0.50)
	for _, s := range []float64{-1, 1} {
		c.disc(cx+s*r*0.62, kidFoot-r*0.03, r*0.26, r*0.13, pale)
	}
	// ears
	for _, s := range []float64{-1, 1} {
		c.tri3(cx+s*r*0.50, cy-r*1.34, cx+s*r*0.10, cy-r*0.66, cx+s*r*0.94, cy-r*0.66, fur)
		c.tri3(cx+s*r*0.48, cy-r*1.10, cx+s*r*0.24, cy-r*0.74, cx+s*r*0.74, cy-r*0.74, colorOf("pink"))
	}
	c.disc(cx, cy, r, r*0.94, fur)

	// muzzle: a pale patch under the eyes so the nose has somewhere to sit
	c.disc(cx, cy+cy*0.0+r*0.30, r*0.26, r*0.18, pale)
	f := newFace(cx, cy, r, r*0.94)
	f.draw(c)
	// nose: a small triangle between the eyes and the mouth
	c.tri3(cx-r*0.09, cy+r*0.16, cx+r*0.09, cy+r*0.16, cx, cy+r*0.29, mix3(colorOf("coral"), colorOf("red"), 0.45))
	c.kidNeck(cy, r, fur)
	// two whiskers, one cell each: at this size they read as a line, not fur
	for _, s := range []float64{-1, 1} {
		if math.Abs(p.sv-(cy+r*0.36)) < p.hairV()*0.55 && math.Abs(p.su-(cx+s*r*0.66)) < r*0.20 {
			c.set(mix3(fur, colorOf("black"), 0.45))
		}
	}
}

// charDog: floppy ears hanging past the cheeks, a wide muzzle, a tongue.
func charDog(c *sceneCtx) {
	cx, cy, r := kidCX, kidHeadY, kidHeadR
	fur := colorOf([]string{"nougat", "dtan", "tan", "dbrown", "lorange"}[c.seed%5])
	pale := mix3(fur, colorOf("white"), 0.62)

	// body and a wagging tail
	c.disc(cx+r*0.92, kidFoot-r*0.74, r*0.14, r*0.14, fur)
	c.kidTorso(fur, pale, 1.16, 0.50)
	for _, s := range []float64{-1, 1} {
		c.disc(cx+s*r*0.64, kidFoot-r*0.03, r*0.26, r*0.13, pale)
	}
	// ears: long rounded flaps, outside the head so the silhouette reads "dog"
	for _, s := range []float64{-1, 1} {
		c.rrect(cx+s*r*0.68-r*0.30, cy-r*0.72, cx+s*r*0.68+r*0.30, cy+r*0.44, r*0.28, mix3(fur, colorOf("black"), 0.18))
	}
	c.disc(cx, cy, r, r*0.94, fur)
	// muzzle
	c.disc(cx, cy+r*0.34, r*0.40, r*0.30, pale)
	c.disc(cx, cy+r*0.20, r*0.13, r*0.10, colorOf("black"))
	f := newFace(cx, cy, r, r*0.94)
	f.mouthY = cy + r*0.52
	f.mouthW = r * 0.16
	f.draw(c)
	// tongue
	c.disc(cx, cy+r*0.66, r*0.11, r*0.12, colorOf("coral"))
	c.kidNeck(cy, r, fur)
}

// charRabbit: two tall ears, a round body, a fluffy tail.
func charRabbit(c *sceneCtx) {
	cx, cy, r := kidCX, kidHeadY, kidHeadR
	fur := colorOf([]string{"white", "lgray", "tan", "pink"}[c.seed%4])
	// fluffy tail, body, pale belly, feet
	c.disc(cx-r*0.84, kidFoot-r*0.40, r*0.19, r*0.19, colorOf("white"))
	c.kidTorso(fur, mix3(fur, colorOf("pink"), 0.18), 1.20, 0.46)
	for _, s := range []float64{-1, 1} {
		c.disc(cx+s*r*0.66, kidFoot-r*0.03, r*0.27, r*0.13, mix3(fur, colorOf("pink"), 0.18))
	}
	// ears: long and thick, pink inside. Thin ears made the rabbit read as a
	// stick with two antennae.
	for _, s := range []float64{-1, 1} {
		ex := cx + s*r*0.34
		c.disc(ex, cy-r*0.92, r*0.42, r*0.58, fur)
		c.disc(ex, cy-r*0.92, r*0.26, r*0.42, colorOf("pink"))
	}
	c.disc(cx, cy, r, r*0.94, fur)
	f := newFace(cx, cy, r, r*0.94)
	f.draw(c)
	// nose and two front teeth
	c.disc(cx, cy+r*0.26, r*0.09, r*0.07, colorOf("coral"))
	c.rect(cx-r*0.11, cy+r*0.52, cx-0.002, cy+r*0.66, colorOf("white"))
	c.rect(cx+0.002, cy+r*0.52, cx+r*0.11, cy+r*0.66, colorOf("white"))
	c.kidNeck(cy, r, fur)
}

// charPanda: white head, black ears and eye patches. The patches are what make
// a panda a panda, so they are big.
func charPanda(c *sceneCtx) {
	cx, cy, r := kidCX, kidHeadY, kidHeadR
	white := colorOf("white")
	black := colorOf("black")
	c.kidTorso(white, white, 1.12, 0)
	// a black band across the shoulders, then arms and legs. A white body on a
	// white head is one capsule no matter how the proportions are tuned; the
	// band is what gives the panda a torso.
	c.rect(cx-r*1.12, kidHeadY+r*0.52, cx+r*1.12, kidHeadY+r*0.74, black)
	for _, s := range []float64{-1, 1} {
		c.disc(cx+s*r*1.16, kidFoot-r*0.42, r*0.18, r*0.30, black)
		c.disc(cx+s*r*0.56, kidFoot-r*0.02, r*0.26, r*0.13, black)
	}
	for _, s := range []float64{-1, 1} {
		c.disc(cx+s*r*0.66, cy-r*0.78, r*0.28, r*0.26, black)
	}
	c.disc(cx, cy, r, r*0.94, white)
	// eye patches, tilted so the face is not a template
	for _, s := range []float64{-1, 1} {
		c.disc(cx+s*r*0.40, cy-r*0.02, r*0.26, r*0.30, black)
	}
	// eyes inside the patches, smaller than the shared face
	f := newFace(cx, cy, r, r*0.94)
	f.ink = white
	f.white = mix3(white, colorOf("lblue"), 0.5)
	f.blush = mix3(colorOf("pink"), white, 0.25)
	f.eyeR = r * 0.13
	f.draw(c)
	c.disc(cx, cy+r*0.30, r*0.10, r*0.08, black)
	c.kidNeck(cy, r, white)
}

// charBear: round ears on top of the head, a pale muzzle, a stout body.
func charBear(c *sceneCtx) {
	cx, cy, r := kidCX, kidHeadY, kidHeadR
	fur := colorOf([]string{"nougat", "dorange", "dtan", "redbrown"}[c.seed%4])
	pale := mix3(fur, colorOf("tan"), 0.60)
	c.kidTorso(fur, pale, 1.18, 0.50)
	for _, s := range []float64{-1, 1} {
		c.disc(cx+s*r*0.66, kidFoot-r*0.03, r*0.27, r*0.13, pale)
	}
	for _, s := range []float64{-1, 1} {
		c.disc(cx+s*r*0.80, cy-r*0.98, r*0.30, r*0.30, fur)
		c.disc(cx+s*r*0.80, cy-r*0.98, r*0.16, r*0.16, mix3(fur, colorOf("black"), 0.35))
	}
	c.disc(cx, cy, r, r*0.94, fur)
	c.disc(cx, cy+r*0.32, r*0.34, r*0.26, pale)
	f := newFace(cx, cy, r, r*0.94)
	f.draw(c)
	c.disc(cx, cy+r*0.24, r*0.11, r*0.08, colorOf("black"))
	c.kidNeck(cy, r, fur)
}

// charChick: one yellow egg with a beak. No ears, no neck, no arms - a chick is
// a blob, and a blob is the easiest thing in the world to read.
func charChick(c *sceneCtx) {
	cx := kidCX
	body := colorOf("lyellow")
	legs := colorOf("lorange")
	// legs first, so the body sits on them
	for _, s := range []float64{-1, 1} {
		c.rect(cx+s*0.075-0.012, kidFoot-0.030, cx+s*0.075+0.012, kidFoot+0.020, legs)
	}
	c.disc(cx, 0.560, kidHeadR*0.92, 0.235, body)
	// wings
	for _, s := range []float64{-1, 1} {
		c.disc(cx+s*0.185, 0.610, 0.052, 0.080, mix3(body, colorOf("lorange"), 0.30))
	}
	f := newFace(cx, 0.430, kidHeadR*0.92, kidHeadR*0.94)
	f.eyeR = kidHeadR * 0.24
	f.noMouth = true
	f.draw(c)
	// beak: a diamond wide enough to read at 32 cells
	c.tri3(cx-0.062, 0.492, cx+0.062, 0.492, cx, 0.548, legs)
	c.rect(cx-0.030, 0.516, cx+0.030, 0.524, mix3(legs, colorOf("black"), 0.28))
	// a tuft on the crown
	c.tri3(cx-0.014, 0.200, cx+0.014, 0.200, cx+0.004, 0.150, mix3(body, colorOf("lorange"), 0.45))
}

// charDuck: yellow, a wide flat bill, a wing.
func charDuck(c *sceneCtx) {
	cx := kidCX
	body := colorOf("lyellow")
	bill := colorOf("orange")
	for _, s := range []float64{-1, 1} {
		c.rect(cx+s*0.080-0.014, kidFoot-0.026, cx+s*0.080+0.014, kidFoot+0.022, bill)
	}
	c.disc(cx, 0.575, kidHeadR*0.98, 0.240, body)
	c.disc(cx-0.190, 0.600, 0.058, 0.090, mix3(body, colorOf("tan"), 0.45))
	f := newFace(cx, 0.420, kidHeadR*0.94, kidHeadR*0.94)
	f.noMouth = true
	f.draw(c)
	// bill: a wide flat rounded bar, not a triangle - a triangle is a beak
	c.rrect(cx-0.080, 0.470, cx+0.080, 0.520, 0.020, bill)
	c.rect(cx-0.060, 0.492, cx+0.060, 0.500, mix3(bill, colorOf("black"), 0.30))
}

// charFrog: eyes on top of the head, a wide smile, a pale belly.
func charFrog(c *sceneCtx) {
	cx := kidCX
	skin := colorOf("bgreen")
	pale := mix3(skin, colorOf("white"), 0.55)
	c.disc(cx, 0.600, 0.230, 0.210, skin)
	c.disc(cx, 0.620, 0.150, 0.150, pale)
	for _, s := range []float64{-1, 1} {
		c.disc(cx+s*0.180, kidFoot-0.070, 0.070, 0.048, skin)
	}
	// eye domes, drawn above the body
	for _, s := range []float64{-1, 1} {
		ex := cx + s*0.115
		c.disc(ex, 0.360, 0.086, 0.086, skin)
		c.disc(ex, 0.362, 0.058, 0.058, colorOf("white"))
		c.disc(ex, 0.366, 0.032, 0.034, colorOf("black"))
		c.dot(ex-0.014, 0.352, 0.014, colorOf("white"))
	}
	// mouth: a wide flat smile across the face
	c.rect(cx-0.100, 0.470, cx+0.100, 0.492, mix3(skin, colorOf("black"), 0.62))
	c.rect(cx-0.118, 0.455, cx-0.098, 0.492, mix3(skin, colorOf("black"), 0.62))
	c.rect(cx+0.098, 0.455, cx+0.118, 0.492, mix3(skin, colorOf("black"), 0.62))
	// cheeks
	for _, s := range []float64{-1, 1} {
		c.disc(cx+s*0.150, 0.440, 0.030, 0.020, mix3(colorOf("coral"), colorOf("white"), 0.25))
	}
}

// charPig: a snout with two nostrils. Nothing else in the set has one, so the
// snout alone identifies the animal.
func charPig(c *sceneCtx) {
	cx, cy, r := kidCX, kidHeadY, kidHeadR
	skin := colorOf("pink")
	dark := mix3(skin, colorOf("coral"), 0.55)
	c.kidTorso(skin, mix3(skin, colorOf("white"), 0.35), 1.16, 0.50)
	for _, s := range []float64{-1, 1} {
		c.disc(cx+s*r*0.66, kidFoot-r*0.03, r*0.27, r*0.13, dark)
	}
	for _, s := range []float64{-1, 1} {
		c.disc(cx+s*r*0.72, cy-r*0.70, r*0.30, r*0.26, dark)
		c.disc(cx+s*r*0.72, cy-r*0.70, r*0.17, r*0.14, mix3(dark, colorOf("black"), 0.20))
	}
	c.disc(cx, cy, r, r*0.94, skin)
	f := newFace(cx, cy, r, r*0.94)
	f.noMouth = true
	f.draw(c)
	// snout
	c.disc(cx, cy+r*0.30, r*0.36, r*0.27, dark)
	for _, s := range []float64{-1, 1} {
		c.disc(cx+s*r*0.12, cy+r*0.30, r*0.06, r*0.09, mix3(dark, colorOf("black"), 0.55))
	}
	c.kidNeck(cy, r, skin)
}

// charLion: a mane of triangles around the head. The mane is a ring, not a
// circle, or the lion reads as a sunflower.
func charLion(c *sceneCtx) {
	cx, cy, r := kidCX, kidHeadY, kidHeadR
	body := colorOf("lorange")
	mane := colorOf("dorange")
	pale := mix3(colorOf("tan"), colorOf("white"), 0.25)
	c.kidTorso(body, pale, 1.12, 0.50)
	for _, s := range []float64{-1, 1} {
		c.disc(cx+s*r*0.62, kidFoot-r*0.03, r*0.26, r*0.13, pale)
	}
	// mane: twelve wedges on a ring
	for i := 0; i < 12; i++ {
		a := float64(i) * math.Pi / 6
		mx := cx + math.Cos(a)*r*1.02
		my := cy + math.Sin(a)*r*1.02
		c.disc(mx, my, r*0.34, r*0.34, mane)
	}
	c.disc(cx, cy, r, r*0.94, body)
	for _, s := range []float64{-1, 1} {
		c.disc(cx+s*r*0.62, cy-r*0.70, r*0.14, r*0.14, body)
	}
	c.disc(cx, cy+r*0.34, r*0.30, r*0.22, pale)
	f := newFace(cx, cy, r, r*0.94)
	f.draw(c)
	c.tri3(cx-r*0.09, cy+r*0.24, cx+r*0.09, cy+r*0.24, cx, cy+r*0.36, mix3(colorOf("redbrown"), colorOf("black"), 0.4))
	c.kidNeck(cy, r, body)
}

// charPenguin: a black body with a white front. The shape does all the work.
func charPenguin(c *sceneCtx) {
	cx := kidCX
	black := mix3(colorOf("black"), colorOf("dblue"), 0.30)
	white := colorOf("white")
	beak := colorOf("orange")
	for _, s := range []float64{-1, 1} {
		c.disc(cx+s*0.085, kidFoot-0.010, 0.055, 0.030, beak)
	}
	c.disc(cx, 0.580, 0.205, 0.235, black)
	c.disc(cx, 0.610, 0.140, 0.180, white)
	// flippers
	for _, s := range []float64{-1, 1} {
		c.disc(cx+s*0.195, 0.580, 0.048, 0.110, black)
	}
	f := newFace(cx, 0.400, 0.200, 0.205)
	f.noMouth = true
	f.blush = mix3(colorOf("coral"), white, 0.20)
	f.draw(c)
	c.tri3(cx-0.048, 0.470, cx+0.048, 0.470, cx, 0.512, beak)
}

// ---------------------------------------------------------------- friends

// charGirl: a bob and two pigtails, a dress. The dress is the triangle that
// says "girl" faster than any amount of hair does.
func charGirl(c *sceneCtx) {
	cx := kidCX
	skin := colorOf([]string{"nougat", "tan", "mnougat", "dtan"}[c.seed%4])
	hair := colorOf([]string{"dbrown", "black", "redbrown", "dorange"}[c.seed%4])
	dress := colorOf([]string{"coral", "pink", "lavender", "azure", "lyellow"}[c.seed%5])
	dressDark := mix3(dress, colorOf("black"), 0.18)

	// legs
	for _, s := range []float64{-1, 1} {
		c.rect(cx+s*0.065-0.022, 0.700, cx+s*0.065+0.022, 0.780, skin)
	}
	// dress: a trapezoid, wide at the hem
	if c.p.sv > 0.540 && c.p.sv < 0.720 {
		half := 0.075 + (c.p.sv-0.540)*0.62
		if c.p.box(cx-half, 0.540, cx+half, 0.720) {
			c.set(dress)
		}
	}
	c.rect(cx-0.085, 0.706, cx+0.085, 0.722, dressDark)
	// arms
	for _, s := range []float64{-1, 1} {
		c.rect(cx+s*0.115-0.020, 0.560, cx+s*0.115+0.020, 0.660, skin)
	}
	// head
	r := kidHeadR
	cy := kidHeadY + 0.020
	// pigtails
	for _, s := range []float64{-1, 1} {
		c.disc(cx+s*r*1.02, cy+r*0.24, r*0.24, r*0.30, hair)
		c.disc(cx+s*r*1.02, cy+r*0.24, r*0.12, r*0.16, colorOf("coral"))
	}
	c.disc(cx, cy, r, r*0.96, skin)
	// bob: a cap whose hem sits just above the eyes, plus two side locks.
	// The first version centred the cap on the head, so its hem covered the
	// eyes and every girl came out as a brown blob with a face painted on.
	c.disc(cx, cy-r*0.74, r*1.02, r*0.38, hair)
	c.rect(cx-r*1.02, cy-r*0.60, cx-r*0.74, cy+r*0.34, hair)
	c.rect(cx+r*0.74, cy-r*0.60, cx+r*1.02, cy+r*0.34, hair)
	f := newFace(cx, cy, r, r*0.96)
	f.draw(c)
}

// charBoy: a cap of hair, a t-shirt, shorts and two legs.
func charBoy(c *sceneCtx) {
	cx := kidCX
	skin := colorOf([]string{"nougat", "tan", "mnougat", "dtan"}[c.seed%4])
	hair := colorOf([]string{"black", "dbrown", "dtan", "redbrown"}[c.seed%4])
	shirt := colorOf([]string{"azure", "red", "lyellow", "bgreen", "orange", "purple"}[c.seed%6])
	shorts := colorOf([]string{"dazure", "dgreen", "dbrown", "purple"}[c.seed%4])

	for _, s := range []float64{-1, 1} {
		c.rect(cx+s*0.060-0.024, 0.700, cx+s*0.060+0.024, 0.782, skin)
	}
	c.rect(cx-0.120, 0.640, cx+0.120, 0.716, shorts)
	c.rrect(cx-0.115, 0.530, cx+0.115, 0.660, 0.030, shirt)
	for _, s := range []float64{-1, 1} {
		c.rect(cx+s*0.140-0.022, 0.545, cx+s*0.140+0.022, 0.640, skin)
	}
	r := kidHeadR
	cy := kidHeadY + 0.020
	c.disc(cx, cy, r, r*0.96, skin)
	// hair: a cap above the eyes and one small quiff
	c.disc(cx, cy-r*0.76, r*1.00, r*0.36, hair)
	c.rect(cx-r*0.96, cy-r*0.50, cx+r*0.96, cy-r*0.42, hair)
	c.tri3(cx+r*0.10, cy-r*1.02, cx+r*0.44, cy-r*0.74, cx+r*0.02, cy-r*0.70, hair)
	f := newFace(cx, cy, r, r*0.96)
	f.draw(c)
}

// charPrincess: a crown of three points and a wide gown. Every child in the
// target age recognises the crown before they recognise the face.
func charPrincess(c *sceneCtx) {
	cx := kidCX
	skin := colorOf([]string{"nougat", "tan", "mnougat"}[c.seed%3])
	hair := colorOf([]string{"dbrown", "black", "lyellow", "redbrown"}[c.seed%4])
	gown := colorOf([]string{"pink", "lavender", "coral", "lblue"}[c.seed%4])
	gownDark := mix3(gown, colorOf("black"), 0.16)
	r := kidHeadR

	// gown: wide at the hem, with a darker band
	if c.p.sv > 0.520 && c.p.sv < 0.790 {
		half := 0.070 + (c.p.sv-0.520)*0.66
		if c.p.box(cx-half, 0.520, cx+half, 0.790) {
			c.set(gown)
			if c.p.sv > 0.740 {
				c.set(gownDark)
			}
		}
	}
	// sleeves
	for _, s := range []float64{-1, 1} {
		c.disc(cx+s*0.140, 0.570, 0.045, 0.055, gown)
		c.disc(cx+s*0.140, 0.625, 0.028, 0.028, skin)
	}
	cy := kidHeadY + 0.010
	// long hair behind the shoulders
	for _, s := range []float64{-1, 1} {
		c.rrect(cx+s*0.150-0.045, cy-r*0.60, cx+s*0.150+0.045, cy+r*1.05, 0.040, hair)
	}
	c.disc(cx, cy, r, r*0.96, skin)
	c.disc(cx, cy-r*0.76, r*1.02, r*0.36, hair)
	c.rect(cx-r*1.02, cy-r*0.62, cx-r*0.76, cy+r*0.16, hair)
	c.rect(cx+r*0.76, cy-r*0.62, cx+r*1.02, cy+r*0.16, hair)
	f := newFace(cx, cy, r, r*0.96)
	f.draw(c)
	// crown: three points over the fringe
	c.rect(cx-r*0.60, cy-r*1.02, cx+r*0.60, cy-r*0.86, colorOf("gold"))
	for i := -1; i <= 1; i++ {
		px := cx + float64(i)*r*0.42
		c.tri3(px-r*0.20, cy-r*1.00, px+r*0.20, cy-r*1.00, px, cy-r*1.34, colorOf("gold"))
	}
	c.dot(cx, cy-r*1.08, r*0.07, colorOf("red"))
}

// charHero: a cape, a mask, a chest star. Superhero logic is pure shape.
func charHero(c *sceneCtx) {
	cx := kidCX
	skin := colorOf([]string{"nougat", "tan", "mnougat"}[c.seed%3])
	suit := colorOf([]string{"dazure", "red", "purple", "dgreen"}[c.seed%4])
	cape := colorOf([]string{"red", "coral", "dazure"}[c.seed%3])
	r := kidHeadR
	cy := kidHeadY + 0.020

	// cape behind: two wide triangles from the shoulders
	c.tri3(cx-0.260, 0.560, cx-0.060, 0.560, cx-0.180, 0.790, cape)
	c.tri3(cx+0.260, 0.560, cx+0.060, 0.560, cx+0.180, 0.790, cape)
	// body and legs
	for _, s := range []float64{-1, 1} {
		c.rect(cx+s*0.060-0.026, 0.690, cx+s*0.060+0.026, 0.782, suit)
	}
	c.rrect(cx-0.125, 0.520, cx+0.125, 0.700, 0.030, suit)
	// belt
	c.rect(cx-0.125, 0.640, cx+0.125, 0.664, colorOf("lyellow"))
	// star on the chest
	c.disc(cx, 0.575, 0.042, 0.042, colorOf("lyellow"))
	// arms on the hips
	for _, s := range []float64{-1, 1} {
		c.rect(cx+s*0.150-0.024, 0.545, cx+s*0.150+0.024, 0.650, suit)
		c.disc(cx+s*0.150, 0.655, 0.026, 0.026, skin)
	}
	c.disc(cx, cy, r, r*0.96, skin)
	// hair, kept above the mask
	c.disc(cx, cy-r*0.80, r*1.00, r*0.34, colorOf("black"))
	c.rect(cx-r*1.00, cy-r*0.62, cx-r*0.76, cy-r*0.06, colorOf("black"))
	c.rect(cx+r*0.76, cy-r*0.62, cx+r*1.00, cy-r*0.06, colorOf("black"))
	// mask: a band across the eyes
	c.rect(cx-r*0.94, cy-r*0.34, cx+r*0.94, cy-r*0.02, suit)
	for _, s := range []float64{-1, 1} {
		c.tri3(cx+s*r*0.92, cy-r*0.34, cx+s*r*1.14, cy-r*0.46, cx+s*r*0.94, cy-r*0.06, suit)
	}
	f := newFace(cx, cy, r, r*0.96)
	f.blush = mix3(colorOf("coral"), colorOf("white"), 0.25)
	f.draw(c)
	// the mask clipped the eyes white, so re-draw them on top
	for _, s := range []float64{-1, 1} {
		ex, ey := cx+s*f.eyeDX, cy+f.eyeDY
		c.disc(ex, ey, f.eyeR, f.eyeR*1.18, colorOf("white"))
		c.disc(ex, ey, f.eyeR*0.52, f.eyeR*0.62, colorOf("black"))
		c.dot(ex-f.eyeR*0.26, ey-f.eyeR*0.30, f.eyeR*0.22, colorOf("white"))
	}
}

// charRobot: a square head, an antenna, two square eyes. A robot is the only
// character allowed to have a rectangle for a head, and that is the joke.
func charRobot(c *sceneCtx) {
	cx := kidCX
	metal := colorOf([]string{"lgray", "sandblue", "vlgray"}[c.seed%3])
	trim := colorOf([]string{"dazure", "red", "lime", "orange"}[c.seed%4])
	dark := mix3(metal, colorOf("black"), 0.45)

	// legs and feet
	for _, s := range []float64{-1, 1} {
		c.rect(cx+s*0.075-0.030, 0.690, cx+s*0.075+0.030, 0.775, dark)
		c.rect(cx+s*0.075-0.048, 0.762, cx+s*0.075+0.048, 0.790, metal)
	}
	// body
	c.rrect(cx-0.145, 0.520, cx+0.145, 0.700, 0.020, metal)
	c.rect(cx-0.145, 0.596, cx+0.145, 0.616, trim)
	// buttons
	c.disc(cx-0.070, 0.660, 0.016, 0.016, colorOf("red"))
	c.disc(cx, 0.660, 0.016, 0.016, colorOf("lyellow"))
	c.disc(cx+0.070, 0.660, 0.016, 0.016, colorOf("bgreen"))
	// arms
	for _, s := range []float64{-1, 1} {
		c.rect(cx+s*0.170-0.024, 0.530, cx+s*0.170+0.024, 0.660, dark)
		c.disc(cx+s*0.170, 0.672, 0.030, 0.030, trim)
	}
	// head
	c.rrect(cx-0.170, 0.180, cx+0.170, 0.470, 0.040, metal)
	c.rect(cx-0.170, 0.180, cx+0.170, 0.206, trim)
	// antenna
	c.rect(cx-0.010, 0.120, cx+0.010, 0.184, dark)
	c.disc(cx, 0.112, 0.026, 0.026, colorOf("red"))
	// eyes: square, with a highlight
	for _, s := range []float64{-1, 1} {
		ex := cx + s*0.078
		c.rect(ex-0.042, 0.260, ex+0.042, 0.330, colorOf("white"))
		c.rect(ex-0.028, 0.272, ex+0.028, 0.320, colorOf("dazure"))
		c.rect(ex-0.020, 0.278, ex+0.000, 0.292, colorOf("white"))
	}
	// mouth grille
	for i := 0; i < 3; i++ {
		x := cx - 0.060 + float64(i)*0.045
		c.rect(x, 0.380, x+0.022, 0.412, dark)
	}
	// cheeks
	for _, s := range []float64{-1, 1} {
		c.disc(cx+s*0.120, 0.392, 0.020, 0.014, colorOf("coral"))
	}
}

// charSnowman: three balls, a carrot, a scarf, a hat. The most-built thing in
// the world, and the easiest possible picture.
func charSnowman(c *sceneCtx) {
	cx := kidCX
	snow := colorOf("white")
	shade := mix3(colorOf("lblue"), colorOf("white"), 0.40)
	coal := colorOf("black")
	carrot := colorOf("orange")

	c.disc(cx, 0.700, 0.190, 0.120, snow)
	c.disc(cx, 0.700, 0.190, 0.120, snow)
	// stick arms
	for _, s := range []float64{-1, 1} {
		if math.Abs(c.p.sv-(0.570-0.010)) < 0.012 && math.Abs(c.p.su-(cx+s*0.200)) < 0.075 {
			c.set(colorOf("dbrown"))
		}
	}
	// middle ball
	c.disc(cx, 0.560, 0.150, 0.120, snow)
	c.disc(cx, 0.560, 0.150, 0.120, snow)
	// scarf
	c.rect(cx-0.115, 0.470, cx+0.115, 0.505, colorOf("red"))
	c.rect(cx+0.060, 0.500, cx+0.100, 0.580, colorOf("red"))
	// head
	c.disc(cx, 0.350, 0.155, 0.140, snow)
	c.disc(cx, 0.350, 0.155, 0.140, snow)
	f := newFace(cx, 0.345, 0.155, 0.140)
	f.ink = coal
	f.white = shade
	f.blush = mix3(colorOf("coral"), colorOf("white"), 0.20)
	f.draw(c)
	// carrot nose
	c.tri3(cx-0.020, 0.390, cx+0.020, 0.390, cx+0.070, 0.408, carrot)
	// hat
	c.rect(cx-0.130, 0.150, cx+0.130, 0.186, coal)
	c.rrect(cx-0.085, 0.040, cx+0.085, 0.155, 0.020, coal)
	c.rect(cx-0.085, 0.108, cx+0.085, 0.126, colorOf("darkred"))
	// buttons
	c.disc(cx, 0.560, 0.016, 0.016, coal)
	c.disc(cx, 0.610, 0.016, 0.016, coal)
}

// charClown: a cone hat with a pompom, a big red nose, a ruff. Everything is
// round on purpose; a clown is the one adult allowed to be a cartoon.
func charClown(c *sceneCtx) {
	cx := kidCX
	skin := colorOf("white")
	r := kidHeadR
	cy := kidHeadY + 0.035

	// body: three colour blocks
	c.rrect(cx-0.140, 0.550, cx+0.140, 0.760, 0.040, colorOf("dazure"))
	c.rect(cx-0.140, 0.620, cx+0.140, 0.660, colorOf("lyellow"))
	c.rect(cx-0.140, 0.700, cx+0.140, 0.740, colorOf("red"))
	// ruff
	for i := -2; i <= 2; i++ {
		c.disc(cx+float64(i)*0.055, 0.520, 0.045, 0.035, colorOf("lyellow"))
	}
	// head
	c.disc(cx, cy, r*0.94, r*0.90, skin)
	// hair tufts
	for _, s := range []float64{-1, 1} {
		c.disc(cx+s*r*0.98, cy+r*0.10, r*0.24, r*0.24, colorOf("coral"))
	}
	// hat: a cone tilting right, with a pompom
	c.tri3(cx-r*0.62, cy-r*0.72, cx+r*0.62, cy-r*0.72, cx+r*0.30, cy-r*1.44, colorOf("red"))
	c.rect(cx-r*0.66, cy-r*0.78, cx+r*0.66, cy-r*0.68, colorOf("lyellow"))
	c.disc(cx+r*0.30, cy-r*1.50, r*0.14, r*0.14, colorOf("lime"))
	// face
	f := newFace(cx, cy, r*0.94, r*0.90)
	f.noBlush = true
	f.noMouth = true
	f.draw(c)
	// the big nose: a red ball between the eyes and the mouth
	c.disc(cx, cy+r*0.18, r*0.20, r*0.17, colorOf("red"))
	c.dot(cx-r*0.06, cy+r*0.12, r*0.06, colorOf("coral"))
	// painted smile, wide and thin
	c.rect(cx-r*0.34, cy+r*0.44, cx+r*0.34, cy+r*0.48, colorOf("red"))
	c.rect(cx-r*0.38, cy+r*0.38, cx-r*0.32, cy+r*0.48, colorOf("red"))
	c.rect(cx+r*0.32, cy+r*0.38, cx+r*0.38, cy+r*0.48, colorOf("red"))
}

// --------------------------------------------------------------- vehicles

// The vehicles are drawn head-on, not in profile.
//
// A side view is the better drawing and the worse sticker: the windscreen is a
// letterbox, so the face has to be squeezed into twenty cells and turns into a
// smudge, and the vehicle ends up a third the height of the card. Head-on, the
// windscreen fills the frame and the eyes sit in it at the same size as every
// animal's. Tayo and Thomas both work this way for the same reason.

// charCar: a bubble car, head-on. The windscreen is the face.
func charCar(c *sceneCtx) {
	cx := kidCX
	body := colorOf([]string{"red", "azure", "lyellow", "bgreen", "orange", "purple"}[c.seed%6])
	trim := mix3(body, colorOf("black"), 0.22)
	glass := colorOf("lblue")
	dark := mix3(body, colorOf("black"), 0.55)

	for _, s := range []float64{-1, 1} {
		c.rrect(cx+s*0.215-0.070, 0.700, cx+s*0.215+0.070, 0.792, 0.028, colorOf("black"))
	}
	c.rrect(cx-0.270, 0.395, cx+0.270, 0.742, 0.075, body)
	// roof
	c.rrect(cx-0.220, 0.318, cx+0.220, 0.420, 0.055, trim)
	// windscreen
	c.rrect(cx-0.205, 0.415, cx+0.205, 0.632, 0.055, glass)
	// face
	f := newFace(cx, 0.480, 0.205, 0.110)
	f.eyeDX = 0.092
	f.eyeDY = 0.0
	f.eyeR = 0.050
	f.mouthY = 0.575
	f.mouthW = 0.048
	f.noBlush = true
	f.draw(c)
	// headlights, then the grille under the mouth
	for _, s := range []float64{-1, 1} {
		c.disc(cx+s*0.185, 0.668, 0.058, 0.038, colorOf("lyellow"))
	}
	c.rect(cx-0.080, 0.650, cx+0.080, 0.690, dark)
	c.rect(cx-0.270, 0.712, cx+0.270, 0.736, dark)
	// wing mirrors
	for _, s := range []float64{-1, 1} {
		c.rect(cx+s*0.272-0.022, 0.470, cx+s*0.272+0.022, 0.510, trim)
	}
}

// charBus: a bus, head-on. A destination sign and a route number do the work of
// saying "bus" that the shape alone cannot.
func charBus(c *sceneCtx) {
	cx := kidCX
	body := colorOf([]string{"lyellow", "red", "azure", "orange"}[c.seed%4])
	trim := mix3(body, colorOf("black"), 0.28)
	glass := colorOf("lblue")
	dark := mix3(body, colorOf("black"), 0.55)

	for _, s := range []float64{-1, 1} {
		c.rrect(cx+s*0.215-0.065, 0.700, cx+s*0.215+0.065, 0.792, 0.026, colorOf("black"))
	}
	c.rrect(cx-0.275, 0.300, cx+0.275, 0.745, 0.055, body)
	// destination blind
	c.rect(cx-0.210, 0.330, cx+0.210, 0.378, colorOf("black"))
	for i := 0; i < 3; i++ {
		c.rect(cx-0.150+float64(i)*0.110, 0.344, cx-0.090+float64(i)*0.110, 0.362, colorOf("lyellow"))
	}
	// big windscreen
	c.rrect(cx-0.225, 0.395, cx+0.225, 0.650, 0.035, glass)
	f := newFace(cx, 0.470, 0.220, 0.120)
	f.eyeDX = 0.098
	f.eyeDY = 0.0
	f.eyeR = 0.052
	f.mouthY = 0.572
	f.mouthW = 0.050
	f.noBlush = true
	f.draw(c)
	// headlights and bumper
	for _, s := range []float64{-1, 1} {
		c.disc(cx+s*0.200, 0.682, 0.056, 0.036, colorOf("lyellow"))
	}
	c.rect(cx-0.275, 0.706, cx+0.275, 0.734, trim)
	c.rect(cx-0.120, 0.690, cx+0.120, 0.706, dark)
}

// charTrain: a tank engine head-on: boiler face, funnel, cowcatcher.
func charTrain(c *sceneCtx) {
	cx := kidCX
	body := colorOf([]string{"red", "dazure", "bgreen", "orange"}[c.seed%4])
	trim := colorOf("lyellow")
	dark := mix3(body, colorOf("black"), 0.38)
	smoke := mix3(colorOf("white"), colorOf("lgray"), 0.30)
	black := colorOf("black")

	// wheels
	for _, s := range []float64{-1, 1} {
		c.disc(cx+s*0.215, 0.735, 0.070, 0.062, black)
		c.disc(cx+s*0.215, 0.735, 0.028, 0.028, colorOf("lgray"))
	}
	// cowcatcher
	c.tri3(cx-0.280, 0.760, cx+0.280, 0.760, cx, 0.660, dark)
	// boiler: a big round face
	c.disc(cx, 0.500, 0.235, 0.215, body)
	c.disc(cx, 0.500, 0.185, 0.170, mix3(body, colorOf("white"), 0.10))
	// funnel and steam
	c.rect(cx-0.070, 0.230, cx+0.070, 0.300, dark)
	c.rect(cx-0.095, 0.212, cx+0.095, 0.244, trim)
	c.disc(cx, 0.166, 0.055, 0.045, smoke)
	c.disc(cx-0.050, 0.104, 0.038, 0.032, smoke)
	// face
	f := newFace(cx, 0.480, 0.200, 0.185)
	f.eyeDX = 0.088
	f.eyeDY = 0.0
	f.eyeR = 0.048
	f.mouthY = 0.570
	f.mouthW = 0.046
	f.noBlush = true
	f.draw(c)
	// buffer beam and lamps
	c.rect(cx-0.245, 0.680, cx+0.245, 0.706, trim)
	for _, s := range []float64{-1, 1} {
		c.disc(cx+s*0.195, 0.640, 0.040, 0.034, colorOf("lyellow"))
	}
}

// charPlane: a plane head-on: nose face, propeller, wings out to the sides.
func charPlane(c *sceneCtx) {
	cx := kidCX
	body := colorOf([]string{"white", "lyellow", "red", "azure"}[c.seed%4])
	accent := colorOf([]string{"red", "dazure", "orange", "purple"}[c.seed%4])
	glass := colorOf("lblue")
	metal := colorOf("dgray")

	// wings, drawn behind the fuselage
	c.rrect(cx-0.330, 0.520, cx-0.130, 0.610, 0.030, accent)
	c.rrect(cx+0.130, 0.520, cx+0.330, 0.610, 0.030, accent)
	// tail fin peeking above
	c.tri3(cx-0.045, 0.230, cx+0.045, 0.230, cx, 0.130, accent)
	// fuselage: a fat oval
	c.disc(cx, 0.510, 0.215, 0.240, body)
	// cockpit window = face
	c.disc(cx, 0.450, 0.165, 0.140, glass)
	f := newFace(cx, 0.450, 0.165, 0.130)
	f.eyeDX = 0.072
	f.eyeDY = 0.0
	f.eyeR = 0.044
	f.mouthY = 0.525
	f.mouthW = 0.040
	f.noBlush = true
	f.draw(c)
	// propeller: a hub and two blades
	c.disc(cx, 0.690, 0.030, 0.030, metal)
	c.rect(cx-0.190, 0.700, cx+0.190, 0.722, metal)
	c.rect(cx-0.190, 0.660, cx-0.170, 0.740, metal)
	c.rect(cx+0.170, 0.660, cx+0.190, 0.740, metal)
	// landing gear
	for _, s := range []float64{-1, 1} {
		c.disc(cx+s*0.115, 0.770, 0.048, 0.036, colorOf("black"))
	}
}

// charBoat: a sailboat seen from the side - the one vehicle that keeps its
// profile, because a boat head-on is a wall.
func charBoat(c *sceneCtx) {
	cx := kidCX
	hull := colorOf([]string{"red", "dazure", "orange", "dgreen"}[c.seed%4])
	sail := colorOf([]string{"white", "lyellow", "pink", "lblue"}[c.seed%4])
	trim := colorOf("white")

	// mast and two sails
	c.rect(cx-0.016, 0.180, cx+0.016, 0.620, colorOf("dbrown"))
	c.tri3(cx+0.018, 0.190, cx+0.018, 0.560, cx+0.255, 0.560, sail)
	c.tri3(cx-0.018, 0.250, cx-0.018, 0.560, cx-0.180, 0.560, mix3(sail, colorOf("lblue"), 0.28))
	c.tri3(cx+0.018, 0.180, cx+0.130, 0.204, cx+0.018, 0.232, colorOf("red"))
	// hull
	if c.p.sv > 0.590 && c.p.sv < 0.740 {
		t := (c.p.sv - 0.590) / 0.150
		half := 0.275 - 0.090*t
		if c.p.box(cx-half, 0.590, cx+half, 0.740) {
			c.set(hull)
		}
	}
	c.rect(cx-0.278, 0.588, cx+0.278, 0.610, trim)
	// face on the hull
	f := newFace(cx, 0.640, 0.110, 0.062)
	f.eyeDX = 0.048
	f.eyeDY = -0.004
	f.eyeR = 0.026
	f.mouthY = 0.688
	f.mouthW = 0.026
	f.noBlush = true
	f.draw(c)
}

// charDigger: tracks, cab, arm. The arm is the whole animal.
func charDigger(c *sceneCtx) {
	cx := kidCX
	body := colorOf([]string{"lyellow", "lorange", "orange"}[c.seed%3])
	dark := mix3(body, colorOf("black"), 0.36)
	metal := colorOf("dgray")

	// tracks
	c.rrect(cx-0.265, 0.690, cx+0.155, 0.792, 0.045, colorOf("black"))
	for i := 0; i < 4; i++ {
		c.disc(cx-0.200+float64(i)*0.105, 0.741, 0.030, 0.030, metal)
	}
	// body and cab
	c.rrect(cx-0.235, 0.560, cx+0.090, 0.700, 0.030, body)
	c.rrect(cx-0.190, 0.380, cx+0.030, 0.580, 0.028, body)
	c.rrect(cx-0.165, 0.402, cx+0.005, 0.500, 0.018, colorOf("lblue"))
	// arm: two segments and a bucket
	c.rect(cx+0.060, 0.400, cx+0.230, 0.436, dark)
	c.rect(cx+0.196, 0.420, cx+0.300, 0.520, dark)
	c.tri3(cx+0.250, 0.520, cx+0.330, 0.520, cx+0.290, 0.630, metal)
	// face on the cab window
	f := newFace(cx-0.080, 0.440, 0.080, 0.048)
	f.eyeDX = 0.034
	f.eyeDY = 0.0
	f.eyeR = 0.019
	f.mouthY = 0.478
	f.mouthW = 0.018
	f.noBlush = true
	f.draw(c)
}

// charFire: a fire engine head-on, ladder on the roof, beacon flashing.
func charFire(c *sceneCtx) {
	cx := kidCX
	body := colorOf("red")
	trim := colorOf("lyellow")
	glass := colorOf("lblue")
	metal := colorOf("lgray")
	dark := mix3(body, colorOf("black"), 0.40)

	for _, s := range []float64{-1, 1} {
		c.rrect(cx+s*0.210-0.066, 0.705, cx+s*0.210+0.066, 0.792, 0.026, colorOf("black"))
	}
	// ladder on the roof, seen end-on: two rails and rungs
	for i := 0; i < 5; i++ {
		c.rect(cx-0.170+float64(i)*0.075, 0.250, cx-0.150+float64(i)*0.075, 0.302, metal)
	}
	c.rect(cx-0.190, 0.268, cx+0.190, 0.284, metal)
	// cab
	c.rrect(cx-0.265, 0.300, cx+0.265, 0.748, 0.055, body)
	// windscreen
	c.rrect(cx-0.215, 0.360, cx+0.215, 0.640, 0.045, glass)
	f := newFace(cx, 0.435, 0.210, 0.115)
	f.eyeDX = 0.094
	f.eyeDY = 0.0
	f.eyeR = 0.050
	f.mouthY = 0.545
	f.mouthW = 0.046
	f.noBlush = true
	f.draw(c)
	// beacon and headlights
	c.rrect(cx-0.075, 0.286, cx+0.075, 0.316, 0.012, dark)
	c.disc(cx, 0.280, 0.040, 0.026, colorOf("lyellow"))
	for _, s := range []float64{-1, 1} {
		c.disc(cx+s*0.190, 0.672, 0.058, 0.038, trim)
	}
	c.rect(cx-0.265, 0.712, cx+0.265, 0.740, mix3(body, colorOf("white"), 0.30))
}

// ----------------------------------------------------------------- sweets

// charCake: three tiers with icing, a cherry, a candle. The face goes on the
// middle tier, which is the only flat area big enough.
func charCake(c *sceneCtx) {
	cx := kidCX
	sponge := colorOf("nougat")
	crumb := mix3(colorOf("nougat"), colorOf("dtan"), 0.55)
	frost := colorOf("white")
	trim := colorOf("pink")
	if c.seed%2 == 1 {
		frost, trim = colorOf("pink"), colorOf("white")
	}
	// plate
	c.disc(cx, 0.782, 0.270, 0.026, colorOf("lgray"))
	// three tiers, narrow at the top
	type tier struct{ y0, y1, hw float64 }
	tiers := []tier{{0.470, 0.540, 0.115}, {0.540, 0.640, 0.165}, {0.640, 0.760, 0.215}}
	for _, t := range tiers {
		if c.p.sv < t.y0 || c.p.sv >= t.y1 || math.Abs(c.p.su-cx) >= t.hw {
			continue
		}
		c.set(sponge)
		if c.p.sv > t.y0+0.028 {
			c.set(crumb)
		}
		if c.p.sv < t.y0+0.024 {
			c.set(frost)
		}
	}
	// drips down each tier's edge
	for _, t := range tiers {
		if c.p.sv > t.y0+0.024 && c.p.sv < t.y0+0.048 {
			d := math.Abs(math.Mod(c.p.su-cx+0.050, 0.100) - 0.050)
			if d > 0.032 {
				c.set(trim)
			}
		}
	}
	// cherry and candle
	c.rect(cx-0.008, 0.360, cx+0.008, 0.470, trim)
	c.disc(cx, 0.404, 0.026, 0.026, colorOf("orange"))
	c.disc(cx, 0.378, 0.016, 0.020, colorOf("lyellow"))
	// face on the middle tier
	f := newFace(cx, 0.588, 0.100, 0.040)
	f.eyeDX = 0.042
	f.eyeDY = -0.004
	f.eyeR = 0.017
	f.mouthY = 0.606
	f.mouthW = 0.020
	f.blush = colorOf("coral")
	f.draw(c)
}

// charIceCream: two scoops on a cone. The cone is a triangle with a lattice.
func charIceCream(c *sceneCtx) {
	cx := kidCX
	scoop := colorOf([]string{"pink", "nougat", "lyellow", "bgreen"}[c.seed%4])
	top := colorOf([]string{"white", "coral", "lblue", "lavender"}[c.seed%4])
	waffle := colorOf("tan")
	dark := mix3(waffle, colorOf("dtan"), 0.55)

	// cone: a wide V, wide enough that the scoop sits on it rather than beside
	// it. The lattice is gated on the cone test - painting it on the whole
	// lower half put a fishnet over the card.
	inCone := c.p.sv >= 0.520 && c.p.sv <= 0.800 &&
		c.p.su >= cx-0.155+(c.p.sv-0.520)*0.58 &&
		c.p.su <= cx+0.155-(c.p.sv-0.520)*0.58
	if inCone {
		c.set(waffle)
		if math.Mod(math.Abs(c.p.su-cx)*5+c.p.sv*6, 0.048) < 0.010 {
			c.set(dark)
		}
	}
	// scoops
	c.disc(cx, 0.480, 0.150, 0.090, scoop)
	c.disc(cx-0.045, 0.330, 0.125, 0.105, top)
	c.disc(cx+0.055, 0.375, 0.100, 0.085, top)
	// drip over the cone edge
	c.disc(cx-0.100, 0.520, 0.045, 0.035, scoop)
	c.disc(cx+0.075, 0.505, 0.040, 0.030, top)
	// cherry
	c.disc(cx-0.045, 0.228, 0.034, 0.030, colorOf("red"))
	c.dot(cx-0.055, 0.220, 0.010, colorOf("coral"))
	// face on the lower scoop
	f := newFace(cx, 0.470, 0.100, 0.070)
	f.eyeDX = 0.042
	f.eyeDY = -0.006
	f.eyeR = 0.020
	f.mouthY = 0.505
	f.mouthW = 0.018
	f.draw(c)
}

// charCupcake: a ridged case, a swirl of frosting, a cherry.
func charCupcake(c *sceneCtx) {
	cx := kidCX
	caseCol := colorOf([]string{"coral", "lblue", "lyellow", "bgreen"}[c.seed%4])
	frost := colorOf([]string{"white", "pink", "lavender"}[c.seed%3])
	dark := mix3(caseCol, colorOf("black"), 0.22)

	// case: trapezoid with vertical ridges
	if c.p.sv > 0.560 && c.p.sv < 0.790 {
		t := (c.p.sv - 0.560) / 0.230
		half := 0.165 - 0.045*t
		if c.p.box(cx-half, 0.560, cx+half, 0.790) {
			c.set(caseCol)
			if math.Mod(c.p.su-cx+0.400, 0.055) < 0.014 {
				c.set(dark)
			}
		}
	}
	c.rect(cx-0.170, 0.548, cx+0.170, 0.570, frost)
	// frosting: three stacked lobes
	c.disc(cx, 0.500, 0.175, 0.070, frost)
	c.disc(cx-0.020, 0.400, 0.130, 0.085, frost)
	c.disc(cx+0.010, 0.310, 0.085, 0.070, frost)
	c.disc(cx-0.005, 0.245, 0.042, 0.038, frost)
	// sprinkles
	for i := 0; i < 5; i++ {
		x := cx - 0.120 + float64(i)*0.058
		y := 0.470 - float64(i%2)*0.030
		c.rect(x, y, x+0.024, y+0.012, colorOf([]string{"red", "lblue", "lime", "orange", "purple"}[i]))
	}
	// face on the lower frosting
	f := newFace(cx, 0.492, 0.105, 0.055)
	f.eyeDX = 0.042
	f.eyeDY = 0.0
	f.eyeR = 0.019
	f.mouthY = 0.522
	f.mouthW = 0.018
	f.blush = colorOf("coral")
	f.draw(c)
}

// charDonut: a ring with icing and sprinkles. Nothing else in the set is a
// circle with a hole, so it reads at any size.
func charDonut(c *sceneCtx) {
	cx, cy := kidCX, 0.520
	dough := colorOf("tan")
	icing := colorOf([]string{"pink", "nougat", "white", "lavender"}[c.seed%4])

	// the ring, not the disc: the hole has to stay card colour, so every shape
	// here is drawn as "inside the ring".
	ring := func(rx, ry float64) bool {
		return c.p.at(cx, cy, rx, ry) && !c.p.at(cx, cy, 0.100, 0.096)
	}
	if ring(0.265, 0.255) {
		c.set(dough)
	}
	// icing sits on the upper two thirds; the dough shows at the bottom
	if ring(0.248, 0.235) && c.p.sv < cy+0.170 {
		c.set(icing)
	}
	// sprinkles
	for i := 0; i < 7; i++ {
		a := float64(i) * 2 * math.Pi / 7
		x := cx + math.Cos(a)*0.175
		y := cy - 0.020 + math.Sin(a)*0.165
		c.rect(x-0.018, y-0.008, x+0.018, y+0.008, colorOf([]string{"red", "lblue", "lime", "lyellow", "purple", "orange", "bgreen"}[i]))
	}
	// face
	f := newFace(cx, cy+0.030, 0.150, 0.120)
	f.eyeDX = 0.062
	f.eyeDY = -0.020
	f.eyeR = 0.030
	f.mouthY = cy + 0.060
	f.mouthW = 0.032
	f.draw(c)
}

// ----------------------------------------------------------------- plants

// charFlower: a pot, a stem, two leaves and a five-lobed flower with a face in
// the middle. The petals are a polar lobe function, not five ellipses: five
// ellipses on a ring get sampled into a row of dashes at 32 cells.
func charFlower(c *sceneCtx) {
	cx := kidCX
	petal := colorOf([]string{"red", "lorange", "pink", "lavender", "lyellow"}[c.seed%5])
	pot := colorOf([]string{"coral", "dazure", "orange", "purple"}[c.seed%4])
	stem := colorOf("bgreen")

	// pot
	if c.p.sv > 0.660 && c.p.sv < 0.790 {
		t := (c.p.sv - 0.660) / 0.130
		half := 0.135 - 0.030*t
		if c.p.box(cx-half, 0.660, cx+half, 0.790) {
			c.set(pot)
		}
	}
	c.rect(cx-0.150, 0.646, cx+0.150, 0.672, mix3(pot, colorOf("white"), 0.28))
	// stem
	c.rect(cx-0.014, 0.360, cx+0.014, 0.660, stem)
	// two mirrored leaves
	for _, s := range []float64{-1, 1} {
		if c.p.rot(cx+s*0.075, 0.560, 0.070, 0.030, s*0.55) {
			c.set(mix3(stem, colorOf("lime"), 0.35))
		}
	}
	// petals: r(theta) pinches to 0.58 between lobes
	fx, fy := cx, 0.290
	if d := math.Hypot(c.p.su-fx, c.p.sv-fy); d > 0.001 && d < 0.150 {
		th := math.Atan2(c.p.sv-fy, c.p.su-fx)
		if d < 0.148*(0.56+0.44*math.Abs(math.Cos(2.5*th))) {
			c.set(petal)
		}
	}
	// core
	c.disc(fx, fy, 0.072, 0.068, colorOf("lorange"))
	c.disc(fx, fy, 0.048, 0.045, colorOf("lyellow"))
	// face in the core
	f := newFace(fx, fy, 0.070, 0.066)
	f.eyeDX = 0.026
	f.eyeDY = -0.004
	f.eyeR = 0.016
	f.mouthY = fy + 0.024
	f.mouthW = 0.014
	f.blush = colorOf("coral")
	f.draw(c)
}

// charTree: a trunk, a round crown, three apples. A tree with a face in the
// crown is a character; a tree without one is scenery.
func charTree(c *sceneCtx) {
	cx := kidCX
	trunk := colorOf("dbrown")
	crown := colorOf([]string{"bgreen", "green", "lime"}[c.seed%3])
	dark := mix3(crown, colorOf("dgreen"), 0.45)

	c.rect(cx-0.055, 0.600, cx+0.055, 0.790, trunk)
	// two roots
	for _, s := range []float64{-1, 1} {
		c.tri3(cx+s*0.050, 0.740, cx+s*0.150, 0.790, cx+s*0.050, 0.790, trunk)
	}
	// crown: three overlapping discs
	c.disc(cx-0.110, 0.430, 0.145, 0.135, crown)
	c.disc(cx+0.110, 0.430, 0.145, 0.135, crown)
	c.disc(cx, 0.330, 0.175, 0.150, crown)
	c.disc(cx, 0.400, 0.130, 0.120, dark)
	// apples
	c.disc(cx-0.150, 0.400, 0.030, 0.030, colorOf("red"))
	c.disc(cx+0.160, 0.360, 0.030, 0.030, colorOf("red"))
	c.disc(cx+0.020, 0.230, 0.028, 0.028, colorOf("red"))
	// face
	f := newFace(cx, 0.395, 0.125, 0.110)
	f.eyeDX = 0.050
	f.eyeDY = -0.010
	f.eyeR = 0.026
	f.mouthY = 0.440
	f.mouthW = 0.026
	f.white = colorOf("lyellow")
	f.draw(c)
}

// charMushroom: a red cap with white spots and a fat stem.
func charMushroom(c *sceneCtx) {
	cx := kidCX
	cap := colorOf([]string{"red", "orange", "purple"}[c.seed%3])
	stem := colorOf("tan")
	spot := colorOf("white")

	// stem
	c.rrect(cx-0.090, 0.470, cx+0.090, 0.790, 0.045, stem)
	c.rect(cx-0.105, 0.700, cx+0.105, 0.790, mix3(stem, colorOf("white"), 0.35))
	// cap: a dome with a flat brim
	c.disc(cx, 0.400, 0.235, 0.190, cap)
	c.rect(cx-0.235, 0.390, cx+0.235, 0.470, cap)
	c.rect(cx-0.235, 0.390, cx+0.235, 0.410, mix3(cap, colorOf("black"), 0.22))
	// spots
	c.disc(cx-0.130, 0.330, 0.045, 0.040, spot)
	c.disc(cx+0.135, 0.350, 0.038, 0.034, spot)
	c.disc(cx-0.005, 0.270, 0.036, 0.030, spot)
	// face on the stem
	f := newFace(cx, 0.590, 0.090, 0.080)
	f.eyeDX = 0.036
	f.eyeDY = -0.006
	f.eyeR = 0.020
	f.mouthY = 0.628
	f.mouthW = 0.020
	f.draw(c)
}

// ----------------------------------------------------------------- nature

// charMountain is the one landscape left. It keeps a face on the sun, because
// a picture in this set without a face is a picture the child turns past.
func charMountain(c *sceneCtx) {
	cx := kidCX
	rock := colorOf([]string{"sandblue", "dtan", "lgray", "tan"}[c.seed%4])
	lit := mix3(rock, colorOf("white"), 0.22)
	shade := mix3(rock, colorOf("black"), 0.30)
	snow := colorOf("white")

	// sun with a face, high and right
	sx, sy := cx+0.230, 0.210
	c.disc(sx, sy, 0.090, 0.086, colorOf("lyellow"))
	f := newFace(sx, sy, 0.090, 0.086)
	f.eyeDX = 0.030
	f.eyeDY = -0.006
	f.eyeR = 0.016
	f.mouthY = sy + 0.030
	f.mouthW = 0.016
	f.blush = colorOf("lorange")
	f.draw(c)

	// clouds
	for _, s := range []float64{-1, 1} {
		ccx := cx + s*0.280
		c.disc(ccx, 0.245, 0.075, 0.036, colorOf("white"))
		c.disc(ccx+s*0.045, 0.235, 0.050, 0.030, colorOf("white"))
	}
	// three peaks, back to front
	type peak struct{ x, y, w float64 }
	peaks := []peak{
		{cx - 0.230, 0.360, 0.200},
		{cx + 0.215, 0.330, 0.215},
		{cx - 0.010, 0.250, 0.290},
	}
	for _, pk := range peaks {
		if c.p.sv < pk.y || c.p.sv > kidGround {
			continue
		}
		t := (c.p.sv - pk.y) / (kidGround - pk.y)
		if math.Abs(c.p.su-pk.x) > pk.w*t {
			continue
		}
		ridge := pk.x - pk.w*t*0.20
		if c.p.su < ridge {
			c.set(lit)
		} else {
			c.set(shade)
		}
		if c.p.sv < pk.y+0.075 {
			c.set(snow)
		}
	}
	// two fir trees at the foot
	for _, s := range []float64{-1, 1} {
		tx := cx + s*0.310
		c.tri3(tx-0.060, 0.790, tx+0.060, 0.790, tx, 0.640, colorOf("dgreen"))
		c.tri3(tx-0.045, 0.730, tx+0.045, 0.730, tx, 0.610, colorOf("green"))
		c.rect(tx-0.014, 0.760, tx+0.014, 0.790, colorOf("dbrown"))
	}
}

// charSun: one big sun with rays and a face, two clouds. The simplest card in
// the set, and the one every toddler recognises first.
func charSun(c *sceneCtx) {
	cx, cy := kidCX, 0.400
	face := colorOf("lyellow")
	ray := colorOf("lorange")

	for i := 0; i < 10; i++ {
		a := float64(i) * math.Pi / 5
		rx := cx + math.Cos(a)*0.290
		ry := cy + math.Sin(a)*0.280
		c.tri3(rx-0.040*math.Sin(a), ry+0.040*math.Cos(a), rx+0.040*math.Sin(a), ry-0.040*math.Cos(a),
			cx+math.Cos(a)*0.360, cy+math.Sin(a)*0.350, ray)
	}
	c.disc(cx, cy, 0.230, 0.225, face)
	f := newFace(cx, cy, 0.230, 0.225)
	f.eyeDX = 0.086
	f.eyeDY = -0.012
	f.eyeR = 0.048
	f.mouthY = cy + 0.080
	f.mouthW = 0.045
	f.blush = colorOf("lorange")
	f.draw(c)
	// two clouds at the foot
	for _, s := range []float64{-1, 1} {
		ccx := cx + s*0.290
		c.disc(ccx, 0.700, 0.110, 0.048, colorOf("white"))
		c.disc(ccx+s*0.060, 0.682, 0.065, 0.042, colorOf("white"))
	}
}

// charRainbow: five bands over two clouds, with a face on the near cloud.
func charRainbow(c *sceneCtx) {
	cx := kidCX
	bands := []string{"red", "orange", "lyellow", "bgreen", "dazure"}
	// the arc is a set of concentric rings, drawn outermost first
	for i, id := range bands {
		r := 0.330 - float64(i)*0.035
		if d := math.Hypot((c.p.su-cx)*1.0, (c.p.sv-0.700)*1.25); d < r && d > r-0.036 {
			c.set(colorOf(id))
		}
	}
	// clouds at both ends
	for _, s := range []float64{-1, 1} {
		ccx := cx + s*0.270
		c.disc(ccx, 0.690, 0.115, 0.052, colorOf("white"))
		c.disc(ccx+s*0.065, 0.672, 0.070, 0.045, colorOf("white"))
		c.disc(ccx-s*0.055, 0.678, 0.055, 0.038, colorOf("white"))
	}
	// face on the left cloud, which is the bigger one
	f := newFace(cx-0.270, 0.688, 0.085, 0.045)
	f.eyeDX = 0.034
	f.eyeDY = -0.002
	f.eyeR = 0.017
	f.mouthY = 0.706
	f.mouthW = 0.016
	f.draw(c)
}

// ------------------------------------------------------------- subject sets

// The order here is the order the gallery walks through a category, so the
// most-loved characters come first: a child who taps "next" three times should
// meet the cat, the dog and the rabbit before the elephant.
var (
	animalSubjects = []subject{
		{"小猫", charCat},
		{"小狗", charDog},
		{"小兔", charRabbit},
		{"熊猫", charPanda},
		{"小熊", charBear},
		{"小鸡", charChick},
		{"小鸭", charDuck},
		{"青蛙", charFrog},
		{"小猪", charPig},
		{"小狮子", charLion},
		{"企鹅", charPenguin},
	}
	peopleSubjects = []subject{
		{"小女孩", charGirl},
		{"小男孩", charBoy},
		{"小公主", charPrincess},
		{"小超人", charHero},
		{"机器人", charRobot},
		{"雪人", charSnowman},
		{"小丑", charClown},
	}
	vehicleSubjects = []subject{
		{"小汽车", charCar},
		{"小巴士", charBus},
		{"小火车", charTrain},
		{"小飞机", charPlane},
		{"小帆船", charBoat},
		{"挖掘机", charDigger},
		{"消防车", charFire},
	}
	sweetSubjects = []subject{
		{"生日蛋糕", charCake},
		{"冰淇淋", charIceCream},
		{"纸杯蛋糕", charCupcake},
		{"甜甜圈", charDonut},
	}
	plantSubjects = []subject{
		{"小花", charFlower},
		{"大树", charTree},
		{"小蘑菇", charMushroom},
	}
	natureSubjects = []subject{
		{"太阳", charSun},
		{"彩虹", charRainbow},
		{"小山", charMountain},
	}
)

// subject is one drawable character in a category.
type subject struct {
	name string
	draw drawFn
}
