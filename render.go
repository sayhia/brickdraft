package main

import (
	"fmt"
	"html"
	"image"
	"image/color"
	"math"
	"sort"
	"strings"
)

// ExportDesign is the payload used by every export format.
type ExportDesign struct {
	Name     string         `json:"name"`
	Width    int            `json:"width"`
	Height   int            `json:"height"`
	Cells    []int          `json:"cells"`
	Settings MosaicSettings `json:"settings"`
}

// ExportOptions controls raster/vector output.
type ExportOptions struct {
	Kind  string `json:"kind"`  // png | svg-sheet | svg-steps | csv
	Scale int    `json:"scale"` // pixels per stud for PNG
	Grid  bool   `json:"grid"`
	Studs bool   `json:"studs"`
}

func normalizeCells(d ExportDesign) []int {
	if len(d.Cells) != d.Width*d.Height {
		out := make([]int, d.Width*d.Height)
		for i := range out {
			out[i] = -1
		}
		return out
	}
	return d.Cells
}

// ------------------------------------------------------------------ drawing

func blend(img *image.RGBA, x, y int, c color.RGBA, cov float64) {
	if cov <= 0 {
		return
	}
	b := img.Bounds()
	if x < b.Min.X || x >= b.Max.X || y < b.Min.Y || y >= b.Max.Y {
		return
	}
	a := cov * float64(c.A) / 255.0
	if a <= 0 {
		return
	}
	if a > 1 {
		a = 1
	}
	i := img.PixOffset(x, y)
	img.Pix[i] = uint8(float64(img.Pix[i])*(1-a) + float64(c.R)*a)
	img.Pix[i+1] = uint8(float64(img.Pix[i+1])*(1-a) + float64(c.G)*a)
	img.Pix[i+2] = uint8(float64(img.Pix[i+2])*(1-a) + float64(c.B)*a)
	img.Pix[i+3] = 255
}

func fillRect(img *image.RGBA, x, y, w, h int, c color.RGBA) {
	for py := y; py < y+h; py++ {
		for px := x; px < x+w; px++ {
			blend(img, px, py, c, 1)
		}
	}
}

func fillCircle(img *image.RGBA, cx, cy, r float64, c color.RGBA) {
	x0 := int(math.Floor(cx - r - 1))
	x1 := int(math.Ceil(cx + r + 1))
	y0 := int(math.Floor(cy - r - 1))
	y1 := int(math.Ceil(cy + r + 1))
	for py := y0; py <= y1; py++ {
		for px := x0; px <= x1; px++ {
			d := math.Hypot(float64(px)+0.5-cx, float64(py)+0.5-cy)
			cov := clampF(r-d+0.5, 0, 1)
			if cov > 0 {
				blend(img, px, py, c, cov)
			}
		}
	}
}

func shift(c color.RGBA, amt float64) color.RGBA {
	if amt >= 0 {
		return color.RGBA{
			R: uint8(clampF(float64(c.R)+(255-float64(c.R))*amt, 0, 255)),
			G: uint8(clampF(float64(c.G)+(255-float64(c.G))*amt, 0, 255)),
			B: uint8(clampF(float64(c.B)+(255-float64(c.B))*amt, 0, 255)),
			A: c.A,
		}
	}
	k := 1 + amt
	return color.RGBA{
		R: uint8(clampF(float64(c.R)*k, 0, 255)),
		G: uint8(clampF(float64(c.G)*k, 0, 255)),
		B: uint8(clampF(float64(c.B)*k, 0, 255)),
		A: c.A,
	}
}

func rgbaOf(pc PaletteColor) color.RGBA {
	return color.RGBA{uint8(pc.R), uint8(pc.G), uint8(pc.B), 255}
}

// drawBrick renders one part with a subtle 3D treatment and studs.
func drawBrick(img *image.RGBA, x, y, w, h, cell int, c color.RGBA, kind string, studs bool) {
	if kind == "round" {
		cx := float64(x) + float64(w*cell)/2
		cy := float64(y) + float64(h*cell)/2
		r := float64(cell)*0.47 - 1
		fillCircle(img, cx, cy, r, shift(c, -0.25))
		fillCircle(img, cx, cy, r*0.86, c)
		fillCircle(img, cx-r*0.18, cy-r*0.18, r*0.34, shift(c, 0.16))
		return
	}

	// soft drop shadow
	fillRect(img, x+1, y+2, w*cell, h*cell, color.RGBA{0, 0, 0, 22})
	fillRect(img, x, y, w*cell, h*cell, shift(c, -0.06))

	if kind == "brick" {
		// taller body: darker bottom edge implies height
		fillRect(img, x, y+h*cell-int(math.Max(2, float64(cell)*0.16)), w*cell, int(math.Max(2, float64(cell)*0.16)), shift(c, -0.28))
		fillRect(img, x, y, w*cell, int(math.Max(1, float64(cell)*0.06)), shift(c, 0.10))
	} else if kind == "tile" {
		fillRect(img, x, y, w*cell, int(math.Max(1, float64(cell)*0.05)), shift(c, 0.14))
		fillRect(img, x, y+h*cell-1, w*cell, 1, shift(c, -0.12))
	}

	if studs && kind != "tile" {
		r := float64(cell) * 0.30
		for sy := 0; sy < h; sy++ {
			for sx := 0; sx < w; sx++ {
				cx := float64(x) + float64(sx*cell) + float64(cell)/2
				cy := float64(y) + float64(sy*cell) + float64(cell)/2
				fillCircle(img, cx, cy+r*0.18, r, shift(c, -0.30))
				fillCircle(img, cx, cy, r, shift(c, 0.10))
				fillCircle(img, cx, cy-r*0.10, r*0.66, shift(c, 0.20))
				fillCircle(img, cx-r*0.22, cy-r*0.30, r*0.26, shift(c, 0.42))
			}
		}
	}
}

// renderPNG renders the mosaic at the requested stud size.
func renderPNG(d ExportDesign, opts ExportOptions) *image.RGBA {
	cell := opts.Scale
	if cell < 8 {
		cell = 26
	}
	if cell > 64 {
		cell = 64
	}
	pad := int(math.Max(12, float64(cell)*0.8))
	W := d.Width*cell + pad*2
	H := d.Height*cell + pad*2

	img := image.NewRGBA(image.Rect(0, 0, W, H))
	fillRect(img, 0, 0, W, H, color.RGBA{0xFA, 0xF9, 0xF5, 0xFF})

	pal := paletteByID(d.Settings.PaletteID)
	cells := normalizeCells(d)

	// empty cells get a faint board texture
	for y := 0; y < d.Height; y++ {
		for x := 0; x < d.Width; x++ {
			if cells[y*d.Width+x] < 0 {
				fillRect(img, pad+x*cell, pad+y*cell, cell, cell, color.RGBA{0xF1, 0xEF, 0xEA, 0xFF})
			}
		}
	}

	bricks := layoutBricks(d.Width, d.Height, cells, d.Settings.BrickMode)
	for _, b := range bricks {
		if b.Color < 0 || b.Color >= len(pal.Colors) {
			continue
		}
		drawBrick(img, pad+b.X*cell, pad+b.Y*cell, b.W, b.H, cell, rgbaOf(pal.Colors[b.Color]), b.Kind, opts.Studs)
	}

	if opts.Grid {
		for i := 0; i <= d.Width; i++ {
			x := pad + i*cell
			c := color.RGBA{0, 0, 0, 18}
			if i%8 == 0 {
				c = color.RGBA{0, 0, 0, 55}
			}
			fillRect(img, x, pad, 1, d.Height*cell, c)
		}
		for i := 0; i <= d.Height; i++ {
			y := pad + i*cell
			c := color.RGBA{0, 0, 0, 18}
			if i%8 == 0 {
				c = color.RGBA{0, 0, 0, 55}
			}
			fillRect(img, pad, y, d.Width*cell, 1, c)
		}
	}

	// frame
	fillRect(img, pad-1, pad-1, d.Width*cell+2, 1, color.RGBA{0, 0, 0, 40})
	fillRect(img, pad-1, pad+d.Height*cell, d.Width*cell+2, 1, color.RGBA{0, 0, 0, 40})
	fillRect(img, pad-1, pad-1, 1, d.Height*cell+1, color.RGBA{0, 0, 0, 40})
	fillRect(img, pad+d.Width*cell, pad-1, 1, d.Height*cell+1, color.RGBA{0, 0, 0, 40})
	return img
}

// --------------------------------------------------------------------- SVG

type svgWriter struct {
	strings.Builder
}

func (s *svgWriter) p(format string, args ...any) {
	fmt.Fprintf(&s.Builder, format+"\n", args...)
}

const svgCellMM = 8.0

// renderSheetSVG produces a 1:1 printable build sheet with legend.
func renderSheetSVG(d ExportDesign) string {
	pal := paletteByID(d.Settings.PaletteID)
	cells := normalizeCells(d)
	bricks := layoutBricks(d.Width, d.Height, cells, d.Settings.BrickMode)
	parts, colors, stats := summarize(d.Width, d.Height, cells, bricks, PaletteInfo{Colors: pal.Colors})

	const margin = 14.0
	const headerH = 26.0
	const gap = 12.0
	legendCols := 4
	entryW := 46.0
	entryH := 8.0
	legendRows := (len(colors) + legendCols - 1) / legendCols
	if legendRows < 1 {
		legendRows = 1
	}
	gridW := float64(d.Width) * svgCellMM
	gridH := float64(d.Height) * svgCellMM
	contentW := math.Max(gridW, float64(legendCols)*entryW)
	totalW := contentW + margin*2
	legendH := float64(legendRows)*entryH + 10
	totalH := margin + headerH + gridH + gap + legendH + margin

	gx := margin + (contentW-gridW)/2
	gy := margin + headerH
	name := d.Name
	if name == "" {
		name = "微颗粒图"
	}

	s := &svgWriter{}
	s.p(`<svg xmlns="http://www.w3.org/2000/svg" width="%.2fmm" height="%.2fmm" viewBox="0 0 %.2f %.2f">`, totalW, totalH, totalW, totalH)
	s.p(`<rect width="100%%" height="100%%" fill="#FFFFFF"/>`)
	s.p(`<g font-family="'Segoe UI','PingFang SC','Microsoft YaHei',sans-serif">`)

	// header
	s.p(`<text x="%.2f" y="%.2f" font-size="6.5" font-weight="700" fill="#1A1A18">%s</text>`, margin, margin+7, html.EscapeString(name))
	s.p(`<text x="%.2f" y="%.2f" font-size="3.6" fill="#8A857C">%dx%d 颗粒 · %.0f×%.0f mm · %d 块零件 · %d 种颜色 · 1:1 施工图</text>`,
		margin, margin+14, d.Width, d.Height, stats.WidthMM, stats.HeightMM, stats.Bricks, stats.Colors)

	// grid background
	s.p(`<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" fill="#FBFAF7" stroke="#D6D2C8" stroke-width="0.3"/>`, gx, gy, gridW, gridH)

	// cells
	for y := 0; y < d.Height; y++ {
		for x := 0; x < d.Width; x++ {
			ci := cells[y*d.Width+x]
			if ci < 0 || ci >= len(pal.Colors) {
				continue
			}
			s.p(`<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" fill="%s"/>`,
				gx+float64(x)*svgCellMM, gy+float64(y)*svgCellMM, svgCellMM, svgCellMM, pal.Colors[ci].Hex)
		}
	}

	// major grid every 8 studs
	var major strings.Builder
	for i := 0; i <= d.Width; i += 8 {
		x := gx + float64(i)*svgCellMM
		fmt.Fprintf(&major, "M%.2f %.2fV%.2f", x, gy, gy+gridH)
	}
	for i := 0; i <= d.Height; i += 8 {
		y := gy + float64(i)*svgCellMM
		fmt.Fprintf(&major, "M%.2f %.2fH%.2f", gx, y, gx+gridW)
	}
	s.p(`<path d="%s" stroke="#000000" stroke-opacity="0.22" stroke-width="0.25" fill="none"/>`, major.String())

	// brick outlines
	var outlines strings.Builder
	for _, b := range bricks {
		fmt.Fprintf(&outlines, `<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f"/>`,
			gx+float64(b.X)*svgCellMM, gy+float64(b.Y)*svgCellMM,
			float64(b.W)*svgCellMM, float64(b.H)*svgCellMM)
	}
	s.p(`<g fill="none" stroke="#000000" stroke-opacity="0.30" stroke-width="0.28">%s</g>`, outlines.String())

	// coordinates
	for i := 0; i < d.Width; i += 8 {
		s.p(`<text x="%.2f" y="%.2f" font-size="2.8" fill="#9A958B" text-anchor="middle">%d</text>`, gx+(float64(i)+4)*svgCellMM, gy-2.4, i+1)
	}
	for i := 0; i < d.Height; i += 8 {
		s.p(`<text x="%.2f" y="%.2f" font-size="2.8" fill="#9A958B" text-anchor="end">%d</text>`, gx-2, gy+(float64(i)+4)*svgCellMM+1, i+1)
	}

	// legend
	ly := gy + gridH + gap
	s.p(`<text x="%.2f" y="%.2f" font-size="3.6" font-weight="700" fill="#3A3833">零件与配色</text>`, margin, ly-2)
	for i, cu := range colors {
		col := i % legendCols
		row := i / legendCols
		ex := margin + float64(col)*entryW
		ey := ly + float64(row)*entryH
		s.p(`<rect x="%.2f" y="%.2f" width="5.2" height="5.2" rx="0.8" fill="%s" stroke="#000000" stroke-opacity="0.18" stroke-width="0.2"/>`, ex, ey, cu.Hex)
		s.p(`<text x="%.2f" y="%.2f" font-size="3.0" fill="#3A3833">%s</text>`, ex+6.6, ey+2.6, html.EscapeString(cu.Name))
		s.p(`<text x="%.2f" y="%.2f" font-size="2.7" fill="#9A958B">%s · %d 块</text>`, ex+6.6, ey+6.0, cu.Hex, cu.Bricks)
	}

	s.p(`</g>`)
	s.p(`</svg>`)
	_ = parts
	return s.String()
}

// renderStepsSVG produces one page per colour so the build is easy to follow.
func renderStepsSVG(d ExportDesign) string {
	pal := paletteByID(d.Settings.PaletteID)
	cells := normalizeCells(d)
	bricks := layoutBricks(d.Width, d.Height, cells, d.Settings.BrickMode)
	_, colors, _ := summarize(d.Width, d.Height, cells, bricks, PaletteInfo{Colors: pal.Colors})

	const margin = 14.0
	const headerH = 14.0
	const pageGap = 16.0
	gridW := float64(d.Width) * svgCellMM
	gridH := float64(d.Height) * svgCellMM
	totalW := gridW + margin*2
	pages := len(colors)
	if pages == 0 {
		pages = 1
	}
	totalH := margin*2 + float64(pages)*(headerH+gridH) + float64(pages-1)*pageGap

	// group bricks per colour for ghost-free outlines
	byColor := make([][]Brick, len(pal.Colors))
	for _, b := range bricks {
		if b.Color >= 0 && b.Color < len(byColor) {
			byColor[b.Color] = append(byColor[b.Color], b)
		}
	}

	name := d.Name
	if name == "" {
		name = "微颗粒图"
	}

	s := &svgWriter{}
	s.p(`<svg xmlns="http://www.w3.org/2000/svg" width="%.2fmm" height="%.2fmm" viewBox="0 0 %.2f %.2f">`, totalW, totalH, totalW, totalH)
	s.p(`<rect width="100%%" height="100%%" fill="#FFFFFF"/>`)
	s.p(`<g font-family="'Segoe UI','PingFang SC','Microsoft YaHei',sans-serif">`)

	for pi, cu := range colors {
		gy := margin + float64(pi)*(headerH+gridH+pageGap) + headerH
		gx := margin
		s.p(`<text x="%.2f" y="%.2f" font-size="5" font-weight="700" fill="#1A1A18">第 %d 步 · %s</text>`, gx, gy-5, pi+1, html.EscapeString(cu.Name))
		s.p(`<text x="%.2f" y="%.2f" font-size="3.2" fill="#8A857C">%s · %d 块 · %s</text>`, gx+38, gy-5, cu.Hex, cu.Bricks, html.EscapeString(name))

		s.p(`<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" fill="#F4F2EE"/>`, gx, gy, gridW, gridH)

		for y := 0; y < d.Height; y++ {
			for x := 0; x < d.Width; x++ {
				if cells[y*d.Width+x] != cu.Index {
					continue
				}
				s.p(`<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" fill="%s"/>`,
					gx+float64(x)*svgCellMM, gy+float64(y)*svgCellMM, svgCellMM, svgCellMM, cu.Hex)
			}
		}

		var major strings.Builder
		for i := 0; i <= d.Width; i += 8 {
			fmt.Fprintf(&major, "M%.2f %.2fV%.2f", gx+float64(i)*svgCellMM, gy, gy+gridH)
		}
		for i := 0; i <= d.Height; i += 8 {
			fmt.Fprintf(&major, "M%.2f %.2fH%.2f", gx, gy+float64(i)*svgCellMM, gx+gridW)
		}
		s.p(`<path d="%s" stroke="#000000" stroke-opacity="0.18" stroke-width="0.2" fill="none"/>`, major.String())

		var outlines strings.Builder
		for _, b := range byColor[cu.Index] {
			fmt.Fprintf(&outlines, `<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f"/>`,
				gx+float64(b.X)*svgCellMM, gy+float64(b.Y)*svgCellMM, float64(b.W)*svgCellMM, float64(b.H)*svgCellMM)
		}
		s.p(`<g fill="none" stroke="#000000" stroke-opacity="0.34" stroke-width="0.25">%s</g>`, outlines.String())
		s.p(`<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" fill="none" stroke="#D6D2C8" stroke-width="0.3"/>`, gx, gy, gridW, gridH)
	}

	s.p(`</g>`)
	s.p(`</svg>`)
	return s.String()
}

// renderCSV produces the bill of materials.
func renderCSV(d ExportDesign) string {
	pal := paletteByID(d.Settings.PaletteID)
	cells := normalizeCells(d)
	bricks := layoutBricks(d.Width, d.Height, cells, d.Settings.BrickMode)
	parts, _, stats := summarize(d.Width, d.Height, cells, bricks, PaletteInfo{Colors: pal.Colors})

	sort.Slice(parts, func(i, j int) bool {
		if parts[i].Count != parts[j].Count {
			return parts[i].Count > parts[j].Count
		}
		return parts[i].Label < parts[j].Label
	})

	var sb strings.Builder
	sb.WriteString("\uFEFF") // BOM so Excel opens UTF-8 correctly
	sb.WriteString("序号,颜色,色号,零件号,零件,数量,小计克重\n")
	for i, p := range parts {
		fmt.Fprintf(&sb, "%d,%s,%s,%s,%s,%d,%.1f\n", i+1, p.ColorName, p.ColorHex, p.PartNo, p.Label, p.Count, p.Grams)
	}
	fmt.Fprintf(&sb, "\n合计,,,,,%d,%.1f\n", stats.Bricks, stats.WeightG)
	fmt.Fprintf(&sb, "成品尺寸,,%dx%d 颗粒,%.0f×%.0f mm,,,\n", stats.Width, stats.Height, stats.WidthMM, stats.HeightMM)
	return sb.String()
}
