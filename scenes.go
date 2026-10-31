package main

import (
	"encoding/base64"
	"image"
	"sync"
)

// SceneInfo describes one picture of the built-in library.
//
// The gallery has 1009 entries, so thumbnails are not part of the startup
// payload: the UI asks for the visible ones in batches through SceneThumbs.
type SceneInfo struct {
	ID       int      `json:"id"`
	Name     string   `json:"name"`
	Cat      int      `json:"cat"`
	Mood     int      `json:"mood"`
	MoodName string   `json:"moodName"`
	Aspect   float64  `json:"aspect"`
	Tags     []string `json:"tags"`
	Thumb    string   `json:"thumb,omitempty"`
	Photo    bool     `json:"photo,omitempty"`
}

// ThumbRequest asks for the previews of a slice of the gallery.
type ThumbRequest struct {
	IDs []int `json:"ids"`
}

// sceneBlob is an immutable picture plus everything derived from it.
type sceneBlob struct {
	img   image.Image
	thumb string
	name  string
	info  SceneInfo
	key   scene
}

var (
	sceneBlobMu   sync.Mutex
	sceneBlobMemo = map[scene]int{}
	sceneBlobSeq  []*sceneBlob
)

// Blob memory is capped: the gallery can be browsed end to end, and nothing
// here is worth more than a few megabytes of pictures.
const (
	blobCacheMax   = 420
	blobCachePrune = 320
)

// sceneBlobFor renders (or recalls) a scene at the reference size. Drawing is
// deterministic, so the cache is pure memoisation.
func sceneBlobFor(sc scene) *sceneBlob {
	sceneBlobMu.Lock()
	if i, ok := sceneBlobMemo[sc]; ok {
		b := sceneBlobSeq[i]
		sceneBlobMu.Unlock()
		return b
	}
	sceneBlobMu.Unlock()

	cols := demoCols
	rows := sceneRows(cols, sceneAspect(sc.ID))
	img := sceneImage(sc, cols, rows, 4)

	b := &sceneBlob{
		img:  img,
		key:  sc,
		name: sceneName(sc.ID),
		info: sceneInfo(sc),
	}
	// Photographic thumbnails go out as JPEG. A 256px PNG of a real scene is
	// 100KB of base64 per card, which is fine for one picture and not fine for
	// the 48 the gallery asks for at once.
	if e := photoAt(sc.ID); e != nil {
		if jpg, err := encodeJPEG(img, 84); err == nil {
			b.thumb = "data:image/jpeg;base64," + b64(jpg)
		}
	} else if png, err := encodePNG(img); err == nil {
		b.thumb = "data:image/png;base64," + b64(png)
	}

	sceneBlobMu.Lock()
	if i, ok := sceneBlobMemo[sc]; ok { // lost a race, keep the first copy
		sceneBlobMu.Unlock()
		return sceneBlobSeq[i]
	}
	sceneBlobSeq = append(sceneBlobSeq, b)
	sceneBlobMemo[sc] = len(sceneBlobSeq) - 1
	if len(sceneBlobSeq) > blobCacheMax {
		// drop the oldest entries, then rebuild the index on what survived
		kept := append([]*sceneBlob(nil), sceneBlobSeq[len(sceneBlobSeq)-blobCachePrune:]...)
		sceneBlobSeq = kept
		for k := range sceneBlobMemo {
			delete(sceneBlobMemo, k)
		}
		for i, cached := range sceneBlobSeq {
			sceneBlobMemo[cached.key] = i
		}
	}
	sceneBlobMu.Unlock()
	return b
}

// sceneInfo builds the gallery record (without the thumbnail).
func sceneInfo(sc scene) SceneInfo {
	cat := sceneCat(sc.ID)
	if e := photoAt(sc.ID); e != nil {
		// tags[0] is what the gallery builds its category chips from, so the
		// category name has to stay in front; 实拍 follows it so a search for
		// the word narrows the list down to the photographs.
		tags := append([]string{categories[e.def.cat].name}, e.def.tags...)
		return SceneInfo{
			ID:       sc.ID,
			Name:     e.def.name,
			Cat:      cat,
			Mood:     -1,
			MoodName: "实拍",
			Aspect:   sceneAspect(sc.ID),
			Tags:     tags,
			Photo:    true,
		}
	}
	m := sc.paletteMood()
	word := categories[cat].word
	return SceneInfo{
		ID:       sc.ID,
		Name:     sceneName(sc.ID),
		Cat:      cat,
		Mood:     m,
		MoodName: moods[m].name,
		Aspect:   sceneAspect(sc.ID),
		Tags:     []string{categories[cat].name, moods[m].name, word.adj},
	}
}

// sceneImage renders a scene at an explicit cell size.
//
// A photograph is resampled rather than painted: it has no cells of its own,
// so the honest preview at N cells is the picture scaled to fit them.
func sceneImage(sc scene, cols, rows, scale int) image.Image {
	if e := photoAt(sc.ID); e != nil {
		return photoScaled(e.picture(), cols*scale, rows*scale)
	}
	return paintScene(cols, rows, scale, sceneShade(sc))
}

// demoImage is the boot picture, used by tests and the splash payload.
func demoImage(gw, gh int) image.Image {
	return sceneImage(scene{ID: firstSceneID, Mood: moodAuto}, gw, gh, 1)
}

// demoFor renders the loaded scene at the requested grid width.
func demoFor(sc scene, width int) image.Image {
	if width < 8 {
		width = 8
	}
	if width > 320 {
		width = 320
	}
	rows := sceneRows(width, sceneAspect(sc.ID))
	// Draw at the largest integer upscale that still keeps the bitmap around
	// 2K: the artwork is flat blocks, so any integer factor downsamples back to
	// exactly the sampled colour, and a 320-cell scene stays cheap to paint.
	scale := demoScale
	if maxW := 2048 / width; maxW < scale {
		scale = maxW
	}
	if maxH := 2048 / rows; maxH < scale {
		scale = maxH
	}
	if scale < 1 {
		scale = 1
	}
	// The photograph is handed to the quantiser at its own resolution: the
	// average step then does the downsample, which is both more accurate than
	// sampling to cell centres and gives the exported source image something
	// worth looking at.
	if e := photoAt(sc.ID); e != nil {
		return photoScaled(e.picture(), width*scale, rows*scale)
	}
	return sceneImage(sc, width, rows, scale)
}

// b64 is base64 without pulling the import into every file that needs it.
func b64(b []byte) string { return base64.StdEncoding.EncodeToString(b) }

// scenePayload renders a scene for the canvas and encodes it in whatever format
// suits it: photographs as JPEG, because a 1024px PNG of a real scene is over a
// megabyte of base64 for no benefit; drawings as PNG, which is both exact and
// tiny when there are only twenty distinct colours in the picture.
func scenePayload(sc scene, width int) (image.Image, []byte, string) {
	img := demoFor(sc, width)
	if photoAt(sc.ID) != nil {
		if jpg, err := encodeJPEG(img, 90); err == nil {
			return img, jpg, "image/jpeg"
		}
	}
	png, _ := encodePNG(img)
	return img, png, "image/png"
}
