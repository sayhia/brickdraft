package main

import (
	"math"
	"testing"
)

// Reference pairs from Sharma, Wu & Dalal (2005), "The CIEDE2000
// Color-Difference Formula".
func TestDeltaE2000ReferenceValues(t *testing.T) {
	cases := []struct {
		l1, l2       [3]float64
		expected     float64
		descriptioin string
	}{
		{[3]float64{50.0000, 2.6772, -79.7751}, [3]float64{50.0000, 0.0000, -82.7485}, 2.0425, "pair 1"},
		{[3]float64{50.0000, 3.1571, -77.2803}, [3]float64{50.0000, 0.0000, -82.7485}, 2.8615, "pair 2"},
		{[3]float64{50.0000, 2.8361, -74.0200}, [3]float64{50.0000, 0.0000, -82.7485}, 3.4412, "pair 3"},
		{[3]float64{50.0000, -1.3802, -84.2814}, [3]float64{50.0000, 0.0000, -82.7485}, 1.0000, "pair 4"},
		{[3]float64{50.0000, 0.0000, 0.0000}, [3]float64{50.0000, -1.0000, 2.0000}, 2.3669, "pair 9"},
		{[3]float64{50.0000, 2.4900, -0.0010}, [3]float64{50.0000, -2.4900, 0.0009}, 7.1792, "pair 14"},
		{[3]float64{60.2574, -34.0099, 36.2677}, [3]float64{60.4626, -34.1751, 39.4387}, 1.2644, "pair 31"},
		{[3]float64{22.7233, 20.0904, -46.6940}, [3]float64{23.0331, 14.9730, -42.5619}, 2.0373, "pair 33"},
	}
	for _, c := range cases {
		got := deltaE2000(c.l1, c.l2)
		if math.Abs(got-c.expected) > 0.0001 {
			t.Errorf("%s: got %.4f want %.4f", c.descriptioin, got, c.expected)
		}
	}
}

func TestDeltaESymmetry(t *testing.T) {
	a := [3]float64{40, 10, -20}
	b := [3]float64{45, -5, 12}
	if math.Abs(deltaE2000(a, b)-deltaE2000(b, a)) > 1e-9 {
		t.Fatal("deltaE2000 should be symmetric")
	}
}

func TestHexParsing(t *testing.T) {
	r, g, b := hexToRGB("#C91A09")
	if r != 0xC9 || g != 0x1A || b != 0x09 {
		t.Fatalf("hexToRGB = %d,%d,%d", r, g, b)
	}
	if got := rgbToHex(201, 26, 9); got != "#C91A09" {
		t.Fatalf("rgbToHex = %s", got)
	}
	if r2, g2, b2 := hexToRGB("#fff"); r2 != 255 || g2 != 255 || b2 != 255 {
		t.Fatalf("shorthand hex failed: %d,%d,%d", r2, g2, b2)
	}
}

func TestNearestColorExactMatch(t *testing.T) {
	pal := paletteByID("classic")
	for i, c := range pal.Colors {
		lab := rgbToLab(float64(c.R), float64(c.G), float64(c.B))
		if got := nearestColor(lab, pal.Colors); got != i {
			t.Errorf("nearestColor(%s) = %d want %d", c.Name, got, i)
		}
	}
}

func TestLayoutCoversEveryCellOnce(t *testing.T) {
	const w, h = 37, 23
	cells := make([]int, w*h)
	for i := range cells {
		cells[i] = (i / 3) % 5
	}
	for _, mode := range []string{"micro", "round", "tile", "plate", "brick"} {
		bricks := layoutBricks(w, h, cells, mode)
		cover := make([]int, w*h)
		for _, b := range bricks {
			for dy := 0; dy < b.H; dy++ {
				for dx := 0; dx < b.W; dx++ {
					cover[(b.Y+dy)*w+b.X+dx]++
					if cells[(b.Y+dy)*w+b.X+dx] != b.Color {
						t.Fatalf("%s: brick covers a cell of another colour", mode)
					}
				}
			}
		}
		for i, c := range cover {
			if c != 1 {
				t.Fatalf("%s: cell %d covered %d times", mode, i, c)
			}
		}
		if mode == "plate" && len(bricks) >= w*h {
			t.Errorf("plate mode should merge cells, got %d bricks for %d cells", len(bricks), w*h)
		}
	}
}

func TestLayoutHandlesEmptyCells(t *testing.T) {
	cells := []int{0, 0, -1, 1, 1, -1, 0, 0, 0}
	bricks := layoutBricks(3, 3, cells, "plate")
	for _, b := range bricks {
		if b.Color < 0 {
			t.Fatal("brick placed on an empty cell")
		}
	}
	if len(bricks) == 0 {
		t.Fatal("expected bricks")
	}
}

func TestPipelineOnDemoImage(t *testing.T) {
	img := demoImage(600, 400)
	m := buildMosaic(img, defaultSettings())
	if m.Width != 64 {
		t.Fatalf("width = %d", m.Width)
	}
	if m.Height == 0 {
		t.Fatal("height should be derived from the aspect ratio")
	}
	if len(m.Cells) != m.Width*m.Height {
		t.Fatal("cell grid size mismatch")
	}
	total := 0
	for _, p := range m.Parts {
		total += p.Count
	}
	if total != len(m.Bricks) {
		t.Fatalf("parts total %d != bricks %d", total, len(m.Bricks))
	}
	if m.Stats.Colors == 0 {
		t.Fatal("expected at least one colour")
	}
	if m.Stats.WeightG <= 0 {
		t.Fatal("expected a positive weight estimate")
	}
}

func TestRenderersProduceOutput(t *testing.T) {
	img := demoImage(320, 240)
	m := buildMosaic(img, defaultSettings())
	d := ExportDesign{Name: "test", Width: m.Width, Height: m.Height, Cells: m.Cells, Settings: m.Settings}

	pngImg := renderPNG(d, ExportOptions{Scale: 12, Studs: true, Grid: true})
	if pngImg.Bounds().Dx() < m.Width*12 {
		t.Fatal("png too small")
	}
	sheet := renderSheetSVG(d)
	if len(sheet) < 2000 || !contains(sheet, "<svg") {
		t.Fatal("sheet svg looks wrong")
	}
	steps := renderStepsSVG(d)
	if !contains(steps, "第 1 步") {
		t.Fatal("steps svg missing titles")
	}
	csv := renderCSV(d)
	if !contains(csv, "零件号") {
		t.Fatal("csv missing header")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}

// ------------------------------------------------------- generated library

// The gallery is generated from a scene id, so the risk is not a bad pixel but
// a whole picture that looks like a bug: an empty sky, a flat slab of colour,
// or two neighbours that are identical. These tests pin that down.

func TestSceneCatalogueIsComplete(t *testing.T) {
	list := sceneCatalogue()
	want := sceneCount + photoCount
	if len(list) != want {
		t.Fatalf("catalogue has %d entries, want %d", len(list), want)
	}
	seen := map[int]bool{}
	photos := 0
	for i, s := range list {
		if s.ID < firstSceneID || s.ID >= firstPhotoID+photoCount {
			t.Fatalf("id %d out of range", s.ID)
		}
		if seen[s.ID] {
			t.Fatalf("duplicate id %d", s.ID)
		}
		seen[s.ID] = true
		if s.Name == "" || s.MoodName == "" {
			t.Fatalf("id %d has an empty name or mood", s.ID)
		}
		if s.Aspect <= 0 {
			t.Fatalf("id %d has aspect %v", s.ID, s.Aspect)
		}
		if s.Photo {
			photos++
			// the photographs open the gallery: they are what the app is
			// trying to sell, and a user should not have to page to the end
			// of 1000 drawings to find one
			if i >= photoCount {
				t.Errorf("photo %d sits at position %d, want it in the lead block", s.ID, i)
			}
		}
	}
	if photos != photoCount {
		t.Errorf("catalogue marks %d photos, want %d", photos, photoCount)
	}
}

func TestSceneCategoriesAreBalanced(t *testing.T) {
	counts := make([]int, len(categories))
	for i := firstSceneID; i < firstSceneID+sceneCount; i++ {
		counts[sceneCat(i)]++
	}
	want := sceneCount / len(categories)
	for i, n := range counts {
		// every subject should appear to within a few percent
		if n < want-60 || n > want+60 {
			t.Errorf("category %s has %d scenes, want about %d", categories[i].name, n, want)
		}
	}
}

func TestScenesAreDrawnNotBlank(t *testing.T) {
	for _, id := range []int{1, 2, 3, 4, 5, 17, 42, 99, 250, 501, 777, 1000} {
		sc := sceneByID(id)
		m := buildMosaic(sceneImage(sc, 48, sceneRows(48, sceneAspect(id)), 2), defaultSettings())
		if m.Stats.Colors < 3 {
			t.Errorf("id %d uses only %d colours: the scene is probably blank", id, m.Stats.Colors)
		}
		// a scene where one brick covers everything has lost its subject
		gross := 0
		for _, c := range m.Colors {
			if c.Cells > gross {
				gross = c.Cells
			}
		}
		if gross > m.Width*m.Height*95/100 {
			t.Errorf("id %d is %.0f%% one colour", id, 100*float64(gross)/float64(m.Width*m.Height))
		}
	}
}

func TestNeighbouringScenesDiffer(t *testing.T) {
	for id := firstSceneID; id < firstSceneID+sceneCount-1; id++ {
		a := sceneImage(sceneByID(id), 32, 32, 1)
		b := sceneImage(sceneByID(id+1), 32, 32, 1)
		same := 0
		for y := 0; y < 32; y++ {
			for x := 0; x < 32; x++ {
				ar, ag, ab, _ := a.At(x, y).RGBA()
				br, bg, bb, _ := b.At(x, y).RGBA()
				if ar == br && ag == bg && ab == bb {
					same++
				}
			}
		}
		if same > 32*32*9/10 {
			t.Fatalf("scenes %d and %d are more than 90%% identical", id, id+1)
		}
	}
}

func TestScenesAreDeterministic(t *testing.T) {
	sc := sceneByID(313)
	a := sceneImage(sc, 40, 32, 2)
	b := sceneImage(sc, 40, 32, 2)
	for y := 0; y < 32; y++ {
		for x := 0; x < 40; x++ {
			ar, ag, ab, _ := a.At(x*2, y*2).RGBA()
			br, bg, bb, _ := b.At(x*2, y*2).RGBA()
			if ar != br || ag != bg || ab != bb {
				t.Fatalf("scene is not reproducible at (%d,%d)", x, y)
			}
		}
	}
}

// A tall grid used to squash the artwork into a letterbox. The frame always
// spans the full height, so the visible window measures the grid's own aspect
// in screen cells while covering an equal number of design units in both
// directions: that equality is exactly "no distortion".
func TestFrameKeepsAspect(t *testing.T) {
	for _, tc := range []struct{ w, h float64 }{{80, 60}, {60, 60}, {60, 80}, {64, 80}, {64, 48}} {
		asp := tc.w / tc.h
		f := frameFor(asp)

		// A cell always spans one unit vertically (the frame is the full
		// height). Horizontally it must span the same one unit, or the picture
		// has been stretched to fill the grid.
		perCellX := (f.x1 - f.x0) / tc.w // frame width in design units / cells
		perCellY := (f.y1 - f.y0) / tc.h
		if math.Abs(perCellX-perCellY) > 1e-9 {
			t.Errorf("grid %vx%v: %.5f units per cell across vs %.5f down", tc.w, tc.h, perCellX, perCellY)
		}
		// The window itself must have the grid's aspect in design units
		// (width measured against the full-height window), so what the
		// painter sees is a faithful crop, never a squeezed one.
		visible := (f.x1 - f.x0) / (f.y1 - f.y0)
		if math.Abs(visible-asp) > 1e-9 {
			t.Errorf("grid %vx%v: window covers %.3f aspect, want %.3f", tc.w, tc.h, visible, asp)
		}
		// and it must be centred on the design box: a wide window overhangs
		// the box on both sides, a tall one sits inside it, but neither may be
		// off to one side
		if math.Abs(f.x0-(1-f.x1)) > 1e-9 {
			t.Errorf("grid %vx%v: frame %+v is not centred", tc.w, tc.h, f)
		}
	}
}

func TestThumbsAreLazyAndComplete(t *testing.T) {
	got := (&StudioService{}).SceneThumbs([]int{1, 2, 3})
	if len(got) != 3 {
		t.Fatalf("got %d thumbnails", len(got))
	}
	for _, it := range got {
		if len(it.Thumb) < 200 || !contains(it.Thumb, "data:image/png;base64,") {
			t.Fatalf("id %d has no usable thumbnail", it.ID)
		}
	}
}

// A recipe's thin features - a whisker, a mouth, a rim light - are expressed as
// a multiple of one cell. That unit has to be the cell's size in *design* space,
// because the frame crops the design box: on a 3:4 grid a cell covers 0.75/cols
// units, not 1/cols. Getting this wrong made every hairline 30% too thick on
// tall grids and too thin on wide ones, so features disappeared at some sizes.
func TestHairIsOneDesignCell(t *testing.T) {
	p := painter{cols: 60, rows: 45, frame: frame{x0: 0.125, y0: 0, x1: 0.875, y1: 1}}
	if got, want := p.hair(), 0.75/60; math.Abs(got-want) > 1e-12 {
		t.Errorf("hair() = %v, want %v", got, want)
	}
	if got, want := p.hairV(), 1.0/45; math.Abs(got-want) > 1e-12 {
		t.Errorf("hairV() = %v, want %v", got, want)
	}
	// a grid whose frame is narrower than the box still has square cells
	q := painter{cols: 40, rows: 40, frame: frame{x0: -0.2, y0: 0, x1: 1.2, y1: 1}}
	if math.Abs(q.hair()-1.4/40) > 1e-12 {
		t.Errorf("hair() on a wide frame = %v, want %v", q.hair(), 1.4/40)
	}
}

// Subject categories that paint an outlined figure rather than a landscape have
// to keep their detail at small grids: thin features that vanish at 48 cells
// make a portrait read as a blob in the gallery thumbnail while looking fine in
// the editor.
func TestFiguresSurviveSmallGrids(t *testing.T) {
	for _, name := range []string{"动物", "人物"} {
		var id int
		for i := firstSceneID; i < firstSceneID+sceneCount && id == 0; i++ {
			if categories[sceneCat(i)].name == name {
				id = i
			}
		}
		if id == 0 {
			t.Fatalf("no scene in category %s", name)
		}
		counts := []int{}
		for _, cols := range []int{48, 96} {
			img := sceneImage(sceneByID(id), cols, sceneRows(cols, sceneAspect(id)), 1)
			seen := map[uint32]bool{}
			b := img.Bounds()
			for y := b.Min.Y; y < b.Max.Y; y++ {
				for x := b.Min.X; x < b.Max.X; x++ {
					r, g, bl, _ := img.At(x, y).RGBA()
					seen[uint32(r>>8)<<16|uint32(g>>8)<<8|uint32(bl>>8)] = true
				}
			}
			counts = append(counts, len(seen))
		}
		if counts[0] < 6 {
			t.Errorf("%s at 48 cells uses only %d colours: detail is gone", name, counts[0])
		}
		// A higher grid may not lose a feature either. The toy style is drawn in
		// flat plates on purpose, so the count is allowed to stay level instead
		// of growing - what it may never do is drop.
		if counts[1] < counts[0] {
			t.Errorf("%s at 96 cells uses %d colours, fewer than %d at 48", name, counts[1], counts[0])
		}
		// and it has to stay buildable: a child's set is a dozen plates, not a
		// photographic gradient
		if counts[1] > 18 {
			t.Errorf("%s at 96 cells uses %d colours: too many for a child's set", name, counts[1])
		}
	}
}
