package main

import (
	"fmt"
	"image"
	"math"
	"sort"
)

const studMM = 8.0 // one stud pitch = 8mm

// ---------------------------------------------------------------- brick data

// BrickType describes a physical LEGO part footprint.
type BrickType struct {
	ID     string  `json:"id"`
	Label  string  `json:"label"`
	PartNo string  `json:"partNo"`
	Kind   string  `json:"kind"` // plate | brick | tile | round
	W      int     `json:"w"`
	H      int     `json:"h"`
	Grams  float64 `json:"grams"`
}

// Brick is one placed part on the grid.
type Brick struct {
	X      int    `json:"x"`
	Y      int    `json:"y"`
	W      int    `json:"w"`
	H      int    `json:"h"`
	TypeID string `json:"typeId"`
	PartNo string `json:"partNo"`
	Kind   string `json:"kind"`
	Label  string `json:"label"`
	Color  int    `json:"color"`
	Rot    bool   `json:"rot"`
}

// BrickModeInfo describes a selectable brick set.
type BrickModeInfo struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Desc   string   `json:"desc"`
	Sample []string `json:"sample"`
}

// PartRow is one line of the bill of materials.
type PartRow struct {
	Color     int     `json:"color"`
	ColorName string  `json:"colorName"`
	ColorHex  string  `json:"colorHex"`
	PartNo    string  `json:"partNo"`
	Label     string  `json:"label"`
	Kind      string  `json:"kind"`
	Count     int     `json:"count"`
	Grams     float64 `json:"grams"`
}

// ColorUsage aggregates how much of a colour is needed.
type ColorUsage struct {
	Index  int    `json:"index"`
	ID     string `json:"id"`
	Name   string `json:"name"`
	Hex    string `json:"hex"`
	Cells  int    `json:"cells"`
	Bricks int    `json:"bricks"`
}

// Stats summarises the finished model.
type Stats struct {
	Width     int     `json:"width"`
	Height    int     `json:"height"`
	Bricks    int     `json:"bricks"`
	Colors    int     `json:"colors"`
	WidthMM   float64 `json:"widthMm"`
	HeightMM  float64 `json:"heightMm"`
	WeightG   float64 `json:"weightG"`
	EmptyCell int     `json:"emptyCells"`
}

// MosaicSettings holds every knob the frontend exposes.
type MosaicSettings struct {
	Width      int     `json:"width"`
	Height     int     `json:"height"`
	PaletteID  string  `json:"paletteId"`
	BrickMode  string  `json:"brickMode"`
	Dither     bool    `json:"dither"`
	Brightness float64 `json:"brightness"`
	Contrast   float64 `json:"contrast"`
	Saturation float64 `json:"saturation"`
	Background string  `json:"background"`
}

// Mosaic is the complete design returned to the frontend.
type Mosaic struct {
	Width    int            `json:"width"`
	Height   int            `json:"height"`
	Cells    []int          `json:"cells"`
	Palette  PaletteInfo    `json:"palette"`
	Bricks   []Brick        `json:"bricks"`
	Parts    []PartRow      `json:"parts"`
	Colors   []ColorUsage   `json:"colors"`
	Stats    Stats          `json:"stats"`
	Settings MosaicSettings `json:"settings"`
}

// Layout is everything that can be derived from a cell grid.
type Layout struct {
	Bricks []Brick      `json:"bricks"`
	Parts  []PartRow    `json:"parts"`
	Colors []ColorUsage `json:"colors"`
	Stats  Stats        `json:"stats"`
}

var brickTypes = map[string]BrickType{
	"1x1-plate": {ID: "1x1-plate", Label: "1×1 板", PartNo: "3024", Kind: "plate", W: 1, H: 1, Grams: 0.18},
	"1x2-plate": {ID: "1x2-plate", Label: "1×2 板", PartNo: "3023", Kind: "plate", W: 2, H: 1, Grams: 0.33},
	"1x3-plate": {ID: "1x3-plate", Label: "1×3 板", PartNo: "3623", Kind: "plate", W: 3, H: 1, Grams: 0.48},
	"1x4-plate": {ID: "1x4-plate", Label: "1×4 板", PartNo: "3710", Kind: "plate", W: 4, H: 1, Grams: 0.64},
	"1x6-plate": {ID: "1x6-plate", Label: "1×6 板", PartNo: "3666", Kind: "plate", W: 6, H: 1, Grams: 0.95},
	"1x8-plate": {ID: "1x8-plate", Label: "1×8 板", PartNo: "3460", Kind: "plate", W: 8, H: 1, Grams: 1.25},
	"2x2-plate": {ID: "2x2-plate", Label: "2×2 板", PartNo: "3022", Kind: "plate", W: 2, H: 2, Grams: 0.62},
	"2x3-plate": {ID: "2x3-plate", Label: "2×3 板", PartNo: "3021", Kind: "plate", W: 3, H: 2, Grams: 0.90},
	"2x4-plate": {ID: "2x4-plate", Label: "2×4 板", PartNo: "3020", Kind: "plate", W: 4, H: 2, Grams: 1.20},

	"1x1-brick": {ID: "1x1-brick", Label: "1×1 砖", PartNo: "3005", Kind: "brick", W: 1, H: 1, Grams: 0.50},
	"1x2-brick": {ID: "1x2-brick", Label: "1×2 砖", PartNo: "3004", Kind: "brick", W: 2, H: 1, Grams: 0.88},
	"1x3-brick": {ID: "1x3-brick", Label: "1×3 砖", PartNo: "3622", Kind: "brick", W: 3, H: 1, Grams: 1.26},
	"1x4-brick": {ID: "1x4-brick", Label: "1×4 砖", PartNo: "3010", Kind: "brick", W: 4, H: 1, Grams: 1.65},
	"2x2-brick": {ID: "2x2-brick", Label: "2×2 砖", PartNo: "3003", Kind: "brick", W: 2, H: 2, Grams: 1.72},
	"2x4-brick": {ID: "2x4-brick", Label: "2×4 砖", PartNo: "3001", Kind: "brick", W: 4, H: 2, Grams: 3.20},

	"1x1-round": {ID: "1x1-round", Label: "1×1 圆板", PartNo: "4073", Kind: "round", W: 1, H: 1, Grams: 0.14},
	"1x1-tile":  {ID: "1x1-tile", Label: "1×1 光面砖", PartNo: "3070b", Kind: "tile", W: 1, H: 1, Grams: 0.16},
}

// candidateOrder lists brick types in placement priority order per mode.
var candidateOrder = map[string][]string{
	"micro": {"1x1-plate"},
	"round": {"1x1-round"},
	"tile":  {"1x1-tile"},
	"plate": {"1x8-plate", "1x6-plate", "1x4-plate", "2x4-plate", "2x3-plate", "1x3-plate", "2x2-plate", "1x2-plate", "1x1-plate"},
	"brick": {"1x4-brick", "2x4-brick", "1x3-brick", "2x2-brick", "1x2-brick", "1x1-brick"},
}

func brickModes() []BrickModeInfo {
	return []BrickModeInfo{
		{ID: "micro", Name: "微颗粒", Desc: "全部 1×1 板，经典像素画质感，颗粒感最强", Sample: []string{"1×1 板"}},
		{ID: "round", Name: "圆颗粒", Desc: "1×1 圆板点阵，像珠绣一样柔和", Sample: []string{"1×1 圆板"}},
		{ID: "tile", Name: "光面", Desc: "1×1 光面砖，无凸点，成品像一幅平整的色块画", Sample: []string{"1×1 光面砖"}},
		{ID: "plate", Name: "薄板拼接", Desc: "1×1–1×8 薄板混合，零件数更少、结构更稳", Sample: []string{"1×4 板", "2×4 板", "1×2 板"}},
		{ID: "brick", Name: "标准砖", Desc: "真砖厚度，适合做立体摆件与底座", Sample: []string{"1×4 砖", "2×4 砖", "1×2 砖"}},
	}
}

func candidatesFor(mode string) []BrickType {
	ids, ok := candidateOrder[mode]
	if !ok {
		ids = candidateOrder["micro"]
	}
	out := make([]BrickType, 0, len(ids))
	for _, id := range ids {
		if bt, ok := brickTypes[id]; ok {
			out = append(out, bt)
		}
	}
	return out
}

// --------------------------------------------------------------- grid sizing

func gridSize(sw, sh int, s MosaicSettings) (int, int) {
	w, h := s.Width, s.Height
	if sw <= 0 || sh <= 0 {
		sw, sh = 1, 1
	}
	if w <= 0 && h <= 0 {
		w = 64
	}
	if h <= 0 {
		h = int(math.Round(float64(w) * float64(sh) / float64(sw)))
		if h < 1 {
			h = 1
		}
	}
	if w <= 0 {
		w = int(math.Round(float64(h) * float64(sw) / float64(sh)))
		if w < 1 {
			w = 1
		}
	}
	w = clampInt(w, 8, 320)
	h = clampInt(h, 8, 320)
	return w, h
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func clampF(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// ------------------------------------------------------------------ sampling

var linLUT [256]float64

func init() {
	for i := 0; i < 256; i++ {
		linLUT[i] = srgbToLinear(float64(i))
	}
}

// averageGrid downsamples the image into a w*h grid of gamma-correct averages.
// The result is a flat RGB slice in 0..255 sRGB, plus an alpha slice.
func averageGrid(img image.Image, w, h int, bg PaletteColor) ([]float64, []float64) {
	b := img.Bounds()
	sw, sh := b.Dx(), b.Dy()
	rgb := make([]float64, w*h*3)
	alpha := make([]float64, w*h)

	for cy := 0; cy < h; cy++ {
		y0 := float64(cy) * float64(sh) / float64(h)
		y1 := float64(cy+1) * float64(sh) / float64(h)
		iy0 := int(math.Floor(y0))
		iy1 := int(math.Ceil(y1))
		for cx := 0; cx < w; cx++ {
			x0 := float64(cx) * float64(sw) / float64(w)
			x1 := float64(cx+1) * float64(sw) / float64(w)
			ix0 := int(math.Floor(x0))
			ix1 := int(math.Ceil(x1))

			var lr, lg, lb, la, wsum float64
			for py := iy0; py < iy1; py++ {
				if py < 0 || py >= sh {
					continue
				}
				wy := math.Min(y1, float64(py+1)) - math.Max(y0, float64(py))
				if wy <= 0 {
					continue
				}
				for px := ix0; px < ix1; px++ {
					if px < 0 || px >= sw {
						continue
					}
					wx := math.Min(x1, float64(px+1)) - math.Max(x0, float64(px))
					if wx <= 0 {
						continue
					}
					ww := wx * wy
					r, g, bl, a := img.At(b.Min.X+px, b.Min.Y+py).RGBA()
					af := float64(a) / 65535.0
					// composite over background in linear light
					cr := linLUT[r>>8]*af + linLUT[bg.R]*(1-af)
					cg := linLUT[g>>8]*af + linLUT[bg.G]*(1-af)
					cb := linLUT[bl>>8]*af + linLUT[bg.B]*(1-af)
					lr += cr * ww
					lg += cg * ww
					lb += cb * ww
					la += af * ww
					wsum += ww
				}
			}
			idx := cy*w + cx
			if wsum <= 0 {
				rgb[idx*3] = float64(bg.R)
				rgb[idx*3+1] = float64(bg.G)
				rgb[idx*3+2] = float64(bg.B)
				alpha[idx] = 0
				continue
			}
			rgb[idx*3] = clampF(linearToSrgb(lr/wsum)*255, 0, 255)
			rgb[idx*3+1] = clampF(linearToSrgb(lg/wsum)*255, 0, 255)
			rgb[idx*3+2] = clampF(linearToSrgb(lb/wsum)*255, 0, 255)
			alpha[idx] = la / wsum
		}
	}
	return rgb, alpha
}

// adjust applies brightness / contrast / saturation in sRGB space.
func adjust(rgb []float64, brightness, contrast, saturation float64) {
	bf := brightness * 80
	cf := 1 + contrast*0.85
	sf := 1 + saturation
	for i := 0; i < len(rgb); i += 3 {
		r, g, b := rgb[i]+bf, rgb[i+1]+bf, rgb[i+2]+bf
		r = (r-128)*cf + 128
		g = (g-128)*cf + 128
		b = (b-128)*cf + 128
		lum := 0.2126*r + 0.7152*g + 0.0722*b
		r = lum + (r-lum)*sf
		g = lum + (g-lum)*sf
		b = lum + (b-lum)*sf
		rgb[i] = clampF(r, 0, 255)
		rgb[i+1] = clampF(g, 0, 255)
		rgb[i+2] = clampF(b, 0, 255)
	}
}

// quantize maps an RGB grid onto the palette, optionally with Floyd–Steinberg
// error diffusion. Returns per-cell palette indices (-1 for empty cells).
func quantize(rgb, alpha []float64, w, h int, colors []PaletteColor, dither bool) []int {
	// The diffused error lives in its own buffer and is clamped per channel.
	// Letting it accumulate without a bound lets a flat area drift far enough
	// from its true colour that a couple of cells land on an unrelated part of
	// the palette — that is what turned smooth gradients into confetti.
	err := make([]float64, len(rgb))
	const strength = 0.55
	const maxErr = 14.0

	cells := make([]int, w*h)
	cache := make(map[uint32]int, 4096)

	nearest := func(r, g, b float64) int {
		ri := int(clampF(math.Round(r), 0, 255))
		gi := int(clampF(math.Round(g), 0, 255))
		bi := int(clampF(math.Round(b), 0, 255))
		key := uint32(ri)<<16 | uint32(gi)<<8 | uint32(bi)
		if v, ok := cache[key]; ok {
			return v
		}
		idx := nearestColor(rgbToLab(float64(ri), float64(gi), float64(bi)), colors)
		cache[key] = idx
		return idx
	}

	for y := 0; y < h; y++ {
		dir := 1
		if y%2 == 1 {
			dir = -1
		}
		for k := 0; k < w; k++ {
			x := k
			if dir < 0 {
				x = w - 1 - k
			}
			i := y*w + x
			if alpha[i] < 0.35 {
				cells[i] = -1
				continue
			}
			r, g, b := rgb[i*3], rgb[i*3+1], rgb[i*3+2]
			j3 := i * 3
			cr := r + clampF(err[j3], -maxErr, maxErr)
			cg := g + clampF(err[j3+1], -maxErr, maxErr)
			cb := b + clampF(err[j3+2], -maxErr, maxErr)
			ci := nearest(cr, cg, cb)
			cells[i] = ci
			if !dither {
				continue
			}
			pc := colors[ci]
			er := (cr - float64(pc.R)) * strength
			eg := (cg - float64(pc.G)) * strength
			eb := (cb - float64(pc.B)) * strength
			add := func(ax, ay int, f float64) {
				if ax < 0 || ax >= w || ay < 0 || ay >= h {
					return
				}
				j := (ay*w + ax) * 3
				err[j] = clampF(err[j]+er*f, -maxErr, maxErr)
				err[j+1] = clampF(err[j+1]+eg*f, -maxErr, maxErr)
				err[j+2] = clampF(err[j+2]+eb*f, -maxErr, maxErr)
			}
			add(x+dir, y, 7.0/16)
			add(x-dir, y+1, 3.0/16)
			add(x, y+1, 5.0/16)
			add(x+dir, y+1, 1.0/16)
		}
	}
	return cells
}

// ------------------------------------------------------------------- layout

func sameColor(cells []int, w, x, y, bw, bh, c int) bool {
	for dy := 0; dy < bh; dy++ {
		for dx := 0; dx < bw; dx++ {
			if cells[(y+dy)*w+x+dx] != c {
				return false
			}
		}
	}
	return true
}

// layoutBricks greedily tiles the grid with the largest parts that fit.
// Rows alternate scan direction so the seams stagger like a real build.
func layoutBricks(w, h int, cells []int, mode string) []Brick {
	types := candidatesFor(mode)
	used := make([]bool, w*h)
	bricks := make([]Brick, 0, w*h/2+8)

	for y := 0; y < h; y++ {
		reverse := y%2 == 1
		for k := 0; k < w; k++ {
			x := k
			if reverse {
				x = w - 1 - k
			}
			idx := y*w + x
			if used[idx] {
				continue
			}
			c := cells[idx]
			if c < 0 {
				used[idx] = true
				continue
			}

			placed := false
			for _, bt := range types {
				orients := [][2]int{{bt.W, bt.H}}
				if bt.W != bt.H {
					orients = append(orients, [2]int{bt.H, bt.W})
				}
				for _, o := range orients {
					bw, bh := o[0], o[1]
					if x+bw > w || y+bh > h {
						continue
					}
					ok := true
					for dy := 0; dy < bh && ok; dy++ {
						for dx := 0; dx < bw; dx++ {
							if used[(y+dy)*w+x+dx] {
								ok = false
								break
							}
						}
					}
					if !ok || !sameColor(cells, w, x, y, bw, bh, c) {
						continue
					}
					bricks = append(bricks, Brick{
						X:      x,
						Y:      y,
						W:      bw,
						H:      bh,
						TypeID: bt.ID,
						PartNo: bt.PartNo,
						Kind:   bt.Kind,
						Label:  fmt.Sprintf("%d×%d %s", minInt(bw, bh), maxInt(bw, bh), kindWord(bt.Kind)),
						Color:  c,
						Rot:    bw != bt.W,
					})
					for dy := 0; dy < bh; dy++ {
						for dx := 0; dx < bw; dx++ {
							used[(y+dy)*w+x+dx] = true
						}
					}
					placed = true
					break
				}
				if placed {
					break
				}
			}
			if !placed {
				used[idx] = true
			}
		}
	}
	return bricks
}

func kindWord(kind string) string {
	switch kind {
	case "plate":
		return "板"
	case "brick":
		return "砖"
	case "tile":
		return "光面砖"
	case "round":
		return "圆板"
	}
	return "颗粒"
}

// summarize builds parts list, colour usage and stats from a layout.
func summarize(w, h int, cells []int, bricks []Brick, pal PaletteInfo) ([]PartRow, []ColorUsage, Stats) {
	cellCount := make([]int, len(pal.Colors))
	brickCount := make([]int, len(pal.Colors))
	empty := 0
	for _, c := range cells {
		if c < 0 {
			empty++
			continue
		}
		if c < len(cellCount) {
			cellCount[c]++
		}
	}

	type pkey struct {
		color  int
		partNo string
	}
	partMap := map[pkey]*PartRow{}
	weight := 0.0
	for _, b := range bricks {
		if b.Color >= 0 && b.Color < len(brickCount) {
			brickCount[b.Color]++
		}
		k := pkey{b.Color, b.PartNo}
		row, ok := partMap[k]
		if !ok {
			row = &PartRow{
				Color:  b.Color,
				PartNo: b.PartNo,
				Label:  b.Label,
				Kind:   b.Kind,
			}
			if b.Color >= 0 && b.Color < len(pal.Colors) {
				row.ColorName = pal.Colors[b.Color].Name
				row.ColorHex = pal.Colors[b.Color].Hex
			}
			partMap[k] = row
		}
		row.Count++
		if bt, ok := brickTypeByPartNo(b.PartNo); ok {
			row.Grams += bt.Grams
			weight += bt.Grams
		}
	}

	parts := make([]PartRow, 0, len(partMap))
	for _, row := range partMap {
		parts = append(parts, *row)
	}
	// Group by part first (so the builder picks a size and then counts colours
	// out of it), busiest colour on top.
	sort.Slice(parts, func(i, j int) bool {
		if parts[i].Kind != parts[j].Kind {
			return parts[i].Kind < parts[j].Kind
		}
		if parts[i].Label != parts[j].Label {
			return parts[i].Label < parts[j].Label
		}
		if parts[i].Count != parts[j].Count {
			return parts[i].Count > parts[j].Count
		}
		return parts[i].ColorName < parts[j].ColorName
	})

	colors := make([]ColorUsage, 0, len(pal.Colors))
	for i, c := range pal.Colors {
		if cellCount[i] == 0 && brickCount[i] == 0 {
			continue
		}
		colors = append(colors, ColorUsage{
			Index:  i,
			ID:     c.ID,
			Name:   c.Name,
			Hex:    c.Hex,
			Cells:  cellCount[i],
			Bricks: brickCount[i],
		})
	}
	sort.Slice(colors, func(i, j int) bool { return colors[i].Cells > colors[j].Cells })

	stats := Stats{
		Width:     w,
		Height:    h,
		Bricks:    len(bricks),
		Colors:    len(colors),
		WidthMM:   float64(w) * studMM,
		HeightMM:  float64(h) * studMM,
		WeightG:   math.Round(weight*10) / 10,
		EmptyCell: empty,
	}
	return parts, colors, stats
}

func brickTypeByPartNo(partNo string) (BrickType, bool) {
	for _, bt := range brickTypes {
		if bt.PartNo == partNo {
			return bt, true
		}
	}
	return BrickType{}, false
}

// buildMosaic runs the whole pipeline from a source image.
func buildMosaic(img image.Image, s MosaicSettings) Mosaic {
	pal := paletteByID(s.PaletteID)
	bgHex := s.Background
	if bgHex == "" {
		bgHex = "#FFFFFF"
	}
	br, bgc, bb := hexToRGB(bgHex)
	bg := PaletteColor{R: br, G: bgc, B: bb}

	w, h := gridSize(img.Bounds().Dx(), img.Bounds().Dy(), s)
	rgb, alpha := averageGrid(img, w, h, bg)
	adjust(rgb, s.Brightness, s.Contrast, s.Saturation)
	cells := quantize(rgb, alpha, w, h, pal.Colors, s.Dither)

	bricks := layoutBricks(w, h, cells, s.BrickMode)
	info := PaletteInfo{ID: pal.ID, Name: pal.Name, Desc: pal.Desc, Count: len(pal.Colors), Colors: pal.Colors}
	parts, colors, stats := summarize(w, h, cells, bricks, info)

	return Mosaic{
		Width:    w,
		Height:   h,
		Cells:    cells,
		Palette:  info,
		Bricks:   bricks,
		Parts:    parts,
		Colors:   colors,
		Stats:    stats,
		Settings: s,
	}
}

// relayout recomputes bricks/parts for an edited cell grid.
func relayout(w, h int, cells []int, s MosaicSettings) Layout {
	pal := paletteByID(s.PaletteID)
	info := PaletteInfo{ID: pal.ID, Name: pal.Name, Desc: pal.Desc, Count: len(pal.Colors), Colors: pal.Colors}
	if len(cells) != w*h {
		cells = make([]int, w*h)
		for i := range cells {
			cells[i] = -1
		}
	}
	bricks := layoutBricks(w, h, cells, s.BrickMode)
	parts, colors, stats := summarize(w, h, cells, bricks, info)
	return Layout{Bricks: bricks, Parts: parts, Colors: colors, Stats: stats}
}
