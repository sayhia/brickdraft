package main

import (
	"image"
	"image/draw"
	"image/png"
	"os"
	"strconv"
	"testing"
)

// TestPhotoMosaicSheet is a review aid, not a test: it renders the photographic
// scenes through the real quantiser so the result can be looked at. The point of
// checking is that a photograph survives being cut down to 64 bricks - if the
// mosaic looks like mud, shipping the photo is worse than shipping nothing.
//
//	XIANGSU_PHOTO_SHEET=64 go test -run PhotoMosaicSheet -v .
func TestPhotoMosaicSheet(t *testing.T) {
	cols, _ := strconv.Atoi(os.Getenv("XIANGSU_PHOTO_SHEET"))
	if cols <= 0 {
		t.Skip("set XIANGSU_PHOTO_SHEET=<grid width> to render")
	}
	pal := os.Getenv("XIANGSU_PHOTO_PALETTE")
	if pal == "" {
		pal = "classic"
	}

	const grid = 3
	tiles := make([]image.Image, photoCount)
	cell := 8
	w, h := 0, 0
	for i := 0; i < photoCount; i++ {
		sc := sceneByID(firstPhotoID + i)
		s := defaultSettings()
		s.Width = cols
		s.PaletteID = pal
		img, _, _ := scenePayload(sc, cols)
		m := buildMosaic(img, s)
		d := ExportDesign{
			Width:    m.Width,
			Height:   m.Height,
			Cells:    m.Cells,
			Settings: m.Settings,
		}
		tiles[i] = renderPNG(d, ExportOptions{Scale: cell})
		w, h = tiles[i].Bounds().Dx(), tiles[i].Bounds().Dy()
	}

	sheet := image.NewRGBA(image.Rect(0, 0, grid*w, grid*h))
	for i, tl := range tiles {
		x, y := (i%grid)*w, (i/grid)*h
		draw.Draw(sheet, image.Rect(x, y, x+w, y+h), tl, tl.Bounds().Min, draw.Src)
	}

	out := ".workbuddy/photo_mosaic_" + strconv.Itoa(cols) + "_" + pal + ".png"
	f, err := os.Create(out)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, sheet); err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote %s", out)
}
