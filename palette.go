package main

import (
	"math"
	"strings"
)

// PaletteColor is a single brick colour in a palette.
type PaletteColor struct {
	ID   string     `json:"id"`
	Name string     `json:"name"`
	Hex  string     `json:"hex"`
	R    int        `json:"r"`
	G    int        `json:"g"`
	B    int        `json:"b"`
	Lab  [3]float64 `json:"-"`
}

// Palette is a named collection of LEGO colours.
type Palette struct {
	ID     string         `json:"id"`
	Name   string         `json:"name"`
	Desc   string         `json:"desc"`
	Colors []PaletteColor `json:"colors"`
}

// PaletteInfo is the light-weight description sent to the frontend.
type PaletteInfo struct {
	ID     string         `json:"id"`
	Name   string         `json:"name"`
	Desc   string         `json:"desc"`
	Count  int            `json:"count"`
	Colors []PaletteColor `json:"colors"`
}

func newColor(id, name, hex string) PaletteColor {
	r, g, b := hexToRGB(hex)
	return PaletteColor{
		ID:   id,
		Name: name,
		Hex:  strings.ToUpper(hex),
		R:    r,
		G:    g,
		B:    b,
		Lab:  rgbToLab(float64(r), float64(g), float64(b)),
	}
}

func hexToRGB(hex string) (int, int, int) {
	h := strings.TrimPrefix(hex, "#")
	if len(h) == 3 {
		h = string([]byte{h[0], h[0], h[1], h[1], h[2], h[2]})
	}
	if len(h) != 6 {
		return 0, 0, 0
	}
	var v [3]int
	for i := 0; i < 3; i++ {
		n := hexNibble(h[i*2])*16 + hexNibble(h[i*2+1])
		v[i] = n
	}
	return v[0], v[1], v[2]
}

func hexNibble(b byte) int {
	switch {
	case b >= '0' && b <= '9':
		return int(b - '0')
	case b >= 'a' && b <= 'f':
		return int(b-'a') + 10
	case b >= 'A' && b <= 'F':
		return int(b-'A') + 10
	}
	return 0
}

func rgbToHex(r, g, b int) string {
	const digits = "0123456789ABCDEF"
	clamp := func(v int) int {
		if v < 0 {
			return 0
		}
		if v > 255 {
			return 255
		}
		return v
	}
	out := []byte{'#'}
	for _, v := range []int{r, g, b} {
		v = clamp(v)
		out = append(out, digits[v>>4], digits[v&0x0f])
	}
	return string(out)
}

// --- colour science -------------------------------------------------------

func srgbToLinear(c float64) float64 {
	c /= 255
	if c <= 0.04045 {
		return c / 12.92
	}
	return math.Pow((c+0.055)/1.055, 2.4)
}

func linearToSrgb(c float64) float64 {
	if c <= 0.0031308 {
		return 12.92 * c
	}
	return 1.055*math.Pow(c, 1/2.4) - 0.055
}

func rgbToLab(r, g, b float64) [3]float64 {
	lr, lg, lb := srgbToLinear(r), srgbToLinear(g), srgbToLinear(b)
	// sRGB D65 -> XYZ
	x := lr*0.4124564 + lg*0.3575761 + lb*0.1804375
	y := lr*0.2126729 + lg*0.7151522 + lb*0.0721750
	z := lr*0.0193339 + lg*0.1191920 + lb*0.9503041

	// D65 reference white
	const xn, yn, zn = 0.95047, 1.0, 1.08883
	f := func(t float64) float64 {
		if t > 0.008856 {
			return math.Cbrt(t)
		}
		return 7.787*t + 16.0/116.0
	}
	fx, fy, fz := f(x/xn), f(y/yn), f(z/zn)
	return [3]float64{116*fy - 16, 500 * (fx - fy), 200 * (fy - fz)}
}

func deg2rad(d float64) float64 { return d * math.Pi / 180 }

func atan2Deg(y, x float64) float64 {
	if x == 0 && y == 0 {
		return 0
	}
	d := math.Atan2(y, x) * 180 / math.Pi
	if d < 0 {
		d += 360
	}
	return d
}

func sq(v float64) float64 { return v * v }

// deltaE2000 implements the CIEDE2000 colour difference formula.
// Reference: Sharma, Wu & Dalal (2005).
func deltaE2000(l1, l2 [3]float64) float64 {
	L1, a1, b1 := l1[0], l1[1], l1[2]
	L2, a2, b2 := l2[0], l2[1], l2[2]

	c1 := math.Hypot(a1, b1)
	c2 := math.Hypot(a2, b2)
	cBar := (c1 + c2) / 2
	cBar7 := math.Pow(cBar, 7)
	g := 0.5 * (1 - math.Sqrt(cBar7/(cBar7+math.Pow(25, 7))))

	a1p := (1 + g) * a1
	a2p := (1 + g) * a2
	c1p := math.Hypot(a1p, b1)
	c2p := math.Hypot(a2p, b2)
	h1p := atan2Deg(b1, a1p)
	h2p := atan2Deg(b2, a2p)

	dLp := L2 - L1
	dCp := c2p - c1p

	var dhp float64
	switch {
	case c1p*c2p == 0:
		dhp = 0
	case math.Abs(h2p-h1p) <= 180:
		dhp = h2p - h1p
	case h2p-h1p > 180:
		dhp = h2p - h1p - 360
	default:
		dhp = h2p - h1p + 360
	}
	dHp := 2 * math.Sqrt(c1p*c2p) * math.Sin(deg2rad(dhp/2))

	lBarP := (L1 + L2) / 2
	cBarP := (c1p + c2p) / 2

	var hBarP float64
	switch {
	case c1p*c2p == 0:
		hBarP = h1p + h2p
	case math.Abs(h1p-h2p) <= 180:
		hBarP = (h1p + h2p) / 2
	case h1p+h2p < 360:
		hBarP = (h1p + h2p + 360) / 2
	default:
		hBarP = (h1p + h2p - 360) / 2
	}

	t := 1 - 0.17*math.Cos(deg2rad(hBarP-30)) +
		0.24*math.Cos(deg2rad(2*hBarP)) +
		0.32*math.Cos(deg2rad(3*hBarP+6)) -
		0.20*math.Cos(deg2rad(4*hBarP-63))

	dTheta := 30 * math.Exp(-sq((hBarP-275)/25))
	cBarP7 := math.Pow(cBarP, 7)
	rc := 2 * math.Sqrt(cBarP7/(cBarP7+math.Pow(25, 7)))
	sl := 1 + (0.015*sq(lBarP-50))/math.Sqrt(20+sq(lBarP-50))
	sc := 1 + 0.045*cBarP
	sh := 1 + 0.015*cBarP*t
	rt := -math.Sin(deg2rad(2*dTheta)) * rc

	return math.Sqrt(sq(dLp/sl) + sq(dCp/sc) + sq(dHp/sh) + rt*(dCp/sc)*(dHp/sh))
}

// --- palettes -------------------------------------------------------------

var (
	// BrickLink style reference values for the classic LEGO palette.
	classicColors = []PaletteColor{
		newColor("white", "白色", "#F4F4F4"),
		newColor("vlgray", "极浅灰", "#E6E3DA"),
		newColor("lgray", "浅灰", "#A0A5A9"),
		newColor("dgray", "深灰", "#6C6E68"),
		newColor("black", "黑色", "#05131D"),
		newColor("red", "红色", "#C91A09"),
		newColor("darkred", "深红", "#720012"),
		newColor("coral", "珊瑚红", "#FF698F"),
		newColor("orange", "橙色", "#FE8A18"),
		newColor("lorange", "亮橙", "#F8BB3D"),
		newColor("dorange", "深橙", "#A95500"),
		newColor("yellow", "黄色", "#F2CD37"),
		newColor("lyellow", "亮黄", "#FFF03A"),
		newColor("tan", "沙色", "#E4CD9E"),
		newColor("dtan", "深沙", "#958A73"),
		newColor("mnougat", "中肉色", "#AA7D55"),
		newColor("nougat", "肤色", "#D09168"),
		newColor("redbrown", "红棕", "#582A12"),
		newColor("dbrown", "深棕", "#352100"),
		newColor("green", "绿色", "#237841"),
		newColor("bgreen", "亮绿", "#4B9F4A"),
		newColor("lime", "青柠", "#BBE90B"),
		newColor("dgreen", "深绿", "#184632"),
		newColor("sandgreen", "沙绿", "#A0BCAC"),
		newColor("olive", "橄榄绿", "#9B9A5A"),
		newColor("dteal", "深青", "#008F9B"),
		newColor("azure", "水蓝", "#36AEBF"),
		newColor("dazure", "靛蓝", "#078BC9"),
		newColor("blue", "蓝色", "#0055BF"),
		newColor("mblue", "中蓝", "#5A93DB"),
		newColor("lblue", "浅蓝", "#9FC3E9"),
		newColor("dblue", "深蓝", "#0A3463"),
		newColor("sandblue", "沙蓝", "#6074A1"),
		newColor("dpurple", "深紫", "#3F3691"),
		newColor("purple", "紫色", "#81007B"),
		newColor("lavender", "淡紫", "#B18AE4"),
		newColor("mlavender", "中紫", "#AC78BA"),
		newColor("dpink", "暗粉", "#C870A0"),
		newColor("pink", "粉色", "#E4ADC8"),
		newColor("magenta", "洋红", "#923978"),
		newColor("gold", "珍珠金", "#AA7F2E"),
	}

	portraitColors = []PaletteColor{
		newColor("white", "白色", "#F4F4F4"),
		newColor("black", "黑色", "#05131D"),
		newColor("dgray", "深灰", "#6C6E68"),
		newColor("lgray", "浅灰", "#A0A5A9"),
		newColor("tan", "沙色", "#E4CD9E"),
		newColor("dtan", "深沙", "#958A73"),
		newColor("nougat", "肤色", "#D09168"),
		newColor("dorange", "深橙", "#A95500"),
		newColor("redbrown", "红棕", "#582A12"),
		newColor("dbrown", "深棕", "#352100"),
		newColor("pink", "粉色", "#E4ADC8"),
		newColor("red", "红色", "#C91A09"),
		newColor("darkred", "深红", "#720012"),
		newColor("sandgreen", "沙绿", "#A0BCAC"),
		newColor("dblue", "深蓝", "#0A3463"),
		newColor("yellow", "黄色", "#F2CD37"),
	}

	vividColors = []PaletteColor{
		newColor("white", "白色", "#F4F4F4"),
		newColor("black", "黑色", "#05131D"),
		newColor("red", "红色", "#C91A09"),
		newColor("darkred", "深红", "#720012"),
		newColor("orange", "橙色", "#FE8A18"),
		newColor("lorange", "亮橙", "#F8BB3D"),
		newColor("yellow", "黄色", "#F2CD37"),
		newColor("lyellow", "亮黄", "#FFF03A"),
		newColor("bgreen", "亮绿", "#4B9F4A"),
		newColor("green", "绿色", "#237841"),
		newColor("azure", "水蓝", "#36AEBF"),
		newColor("blue", "蓝色", "#0055BF"),
		newColor("dblue", "深蓝", "#0A3463"),
		newColor("purple", "紫色", "#81007B"),
		newColor("magenta", "洋红", "#923978"),
		newColor("pink", "粉色", "#E4ADC8"),
	}

	grayColors = []PaletteColor{
		newColor("g0", "雪白", "#F7F7F7"),
		newColor("g1", "浅灰 1", "#D6D6D6"),
		newColor("g2", "浅灰 2", "#B4B7B8"),
		newColor("g3", "中灰", "#8E9294"),
		newColor("g4", "深灰 1", "#6C6E68"),
		newColor("g5", "深灰 2", "#4A4F52"),
		newColor("g6", "炭灰", "#2B3236"),
		newColor("g7", "墨黑", "#05131D"),
	}

	palettes = map[string]Palette{
		"classic":  {ID: "classic", Name: "经典色卡", Desc: "官方常用色，覆盖大多数照片与插画", Colors: classicColors},
		"portrait": {ID: "portrait", Name: "人像 16 色", Desc: "肤色与中性色为主，适合头像与人物", Colors: portraitColors},
		"vivid":    {ID: "vivid", Name: "高饱和 16 色", Desc: "海报、logo 与卡通风格", Colors: vividColors},
		"gray":     {ID: "gray", Name: "灰阶 8 色", Desc: "黑白摄影与线稿", Colors: grayColors},
	}
)

func paletteByID(id string) Palette {
	if p, ok := palettes[id]; ok {
		return p
	}
	return palettes["classic"]
}

func allPalettes() []PaletteInfo {
	order := []string{"classic", "portrait", "vivid", "gray"}
	out := make([]PaletteInfo, 0, len(order))
	for _, id := range order {
		p := palettes[id]
		out = append(out, PaletteInfo{
			ID:     p.ID,
			Name:   p.Name,
			Desc:   p.Desc,
			Count:  len(p.Colors),
			Colors: p.Colors,
		})
	}
	return out
}

// nearestColor returns the index of the closest palette entry using CIEDE2000.
func nearestColor(lab [3]float64, colors []PaletteColor) int {
	best := 0
	bestDist := math.Inf(1)
	for i, c := range colors {
		d := deltaE2000(lab, c.Lab)
		if d < bestDist {
			bestDist = d
			best = i
		}
	}
	return best
}
