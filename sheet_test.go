package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"
)

// TestContactSheet renders a contact sheet of the generated gallery so the
// scenes can be reviewed by eye. Skipped unless XIANGSU_SHEET is set:
//
//	XIANGSU_SHEET=1,240,20 go test -run ContactSheet -count=1
func TestContactSheet(t *testing.T) {
	spec := os.Getenv("XIANGSU_SHEET")
	if spec == "" {
		t.Skip("set XIANGSU_SHEET=first,count,cols to render a sheet")
	}
	var first, count, cols int
	fmt.Sscanf(spec, "%d,%d,%d", &first, &count, &cols)
	if count <= 0 {
		count = 60
	}
	if cols <= 0 {
		cols = 12
	}
	cell := 140
	if v := os.Getenv("XIANGSU_SHEET_CELL"); v != "" {
		fmt.Sscanf(v, "%d", &cell)
	}
	// XIANGSU_SHEET_GRID renders at a different mosaic width, which is how the
	// toy recipes get checked at 32 and 48 cells as well as the default 64.
	grid := 64
	if v := os.Getenv("XIANGSU_SHEET_GRID"); v != "" {
		fmt.Sscanf(v, "%d", &grid)
	}
	rows := (count + cols - 1) / cols
	sheet := image.NewRGBA(image.Rect(0, 0, cols*cell, rows*cell))
	// XIANGSU_SHEET_CAT=星球 renders only that category, which is how the
	// per-category composition gets reviewed.
	ids := make([]int, 0, count)
	if cat := os.Getenv("XIANGSU_SHEET_CAT"); cat != "" {
		for id := firstSceneID; id < firstSceneID+sceneCount && len(ids) < count; id++ {
			if categories[sceneCat(id)].name == cat {
				ids = append(ids, id)
			}
		}
		rows = (len(ids) + cols - 1) / cols
		sheet = image.NewRGBA(image.Rect(0, 0, cols*cell, rows*cell))
	} else {
		for i := 0; i < count; i++ {
			ids = append(ids, first+i)
		}
	}
	for i, id := range ids {
		sc := sceneByID(id)
		img := sceneImage(sc, grid, sceneRows(grid, sceneAspect(id)), 2)
		b := img.Bounds()
		scale := float64(cell-6) / float64(b.Dx())
		if v := float64(cell-6) / float64(b.Dy()); v < scale {
			scale = v
		}
		dw, dh := int(float64(b.Dx())*scale), int(float64(b.Dy())*scale)
		ox := (i%cols)*cell + (cell-dw)/2
		oy := (i/cols)*cell + (cell-dh)/2
		for y := 0; y < dh; y++ {
			for x := 0; x < dw; x++ {
				sheet.Set(ox+x, oy+y, img.At(b.Min.X+x*b.Dx()/dw, b.Min.Y+y*b.Dy()/dh))
			}
		}
		gray := color.RGBA{170, 168, 160, 255}
		for y := 0; y < cell; y++ {
			sheet.Set((i%cols)*cell, (i/cols)*cell+y, gray)
			sheet.Set((i%cols)*cell+cell-1, (i/cols)*cell+y, gray)
		}
		for x := 0; x < cell; x++ {
			sheet.Set((i%cols)*cell+x, (i/cols)*cell, gray)
			sheet.Set((i%cols)*cell+x, (i/cols)*cell+cell-1, gray)
		}
	}
	out := fmt.Sprintf(".workbuddy/sheet_%d.png", first)
	f, err := os.Create(out)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, sheet); err != nil {
		t.Fatal(err)
	}
	t.Log("wrote", out)
}
