package main

import (
	"math"
	"sync"
)

// This file holds the scene model used by the 1000-picture built-in library.
//
// A scene is an integer id plus an optional mood override. Everything else
// (subject, background, palette, aspect) is derived from that id, so the whole
// gallery costs nothing to store and any picture can be redrawn exactly on
// demand at whatever grid size the user picked.
//
// Generating instead of shipping 1000 files is also the faster option: the
// artwork has to be sampled at cell resolution anyway (see paintScene), and
// drawing it costs less than keeping a megabyte of PNGs in the binary.

const (
	sceneCount = 1000
	moodAuto   = -1
)

// Scene ids start at 1 so 0 can mean "nothing selected".
const firstSceneID = 1

// scene is one generated picture.
type scene struct {
	ID   int // 1..sceneCount
	Mood int // -1 = chooser picks from the mood wheel; otherwise a fixed mood
}

// coord spreads the scene number over four strides roughly coprime with 1000,
// so neighbours in the gallery never share a look. Multiplying by an odd number
// modulo 1000 wraps around exactly once, which is what keeps the gallery
// balanced rather than merely varied: every value of an axis appears exactly
// the same number of times across the whole library.
func (sc scene) coord() [4]int {
	n := (sc.ID - firstSceneID + sceneCount) % sceneCount
	return [4]int{
		(n * 13) % sceneCount,
		(n * 41) % sceneCount,
		(n * 83) % sceneCount,
		(n * 97) % sceneCount,
	}
}

// kidMoods is the palette wheel for the sticker library. It is weighted towards
// daylight: a picture book may end at dusk, but a third of the gallery in
// night-club neon is a gallery for somebody else. The indexes point into moods.
var kidMoods = []int{0, 0, 0, 5, 6, 9, 7, 10, 1, 11}

// paletteMood picks the scene's colour mood.
func (sc scene) paletteMood() int {
	if sc.Mood >= 0 {
		return sc.Mood % len(moods)
	}
	n := (sc.ID - firstSceneID + sceneCount) % sceneCount
	return kidMoods[(n*7)%len(kidMoods)]
}

// kidCatWheel is how the children's library divides itself up. Animals take the
// largest share because that is what the age group asks for by name; cities,
// structures and planets keep their category ids only so the photographs have
// somewhere to sit.
var kidCatWheel = []int{
	catAnimal, catAnimal, catAnimal, catAnimal, catAnimal, catAnimal,
	catPortrait, catPortrait, catPortrait,
	catVehicle, catVehicle, catVehicle,
	catFood, catFood,
	catBotanical,
	catMountain,
}

// sceneCat returns the subject category of a scene id.
func sceneCat(id int) int {
	if e := photoAt(id); e != nil {
		return e.def.cat
	}
	n := (id - firstSceneID + sceneCount) % sceneCount
	return kidCatWheel[(n*7)%len(kidCatWheel)]
}

// sceneAspect returns width/height for a scene id.
func sceneAspect(id int) float64 {
	if isPhotoID(id) {
		return 1 // the shots are cropped square before they are embedded
	}
	n := (id - firstSceneID + sceneCount) % sceneCount
	switch {
	case n%7 == 0:
		return 1
	case n%5 == 0:
		return 3.0 / 4.0 // portrait crop
	case n%3 == 0:
		return 4.0 / 3.0
	}
	return 0.8 // 5:4, the default photo crop
}

// sceneRows converts a width to a row count for the given aspect.
func sceneRows(width int, aspect float64) int {
	if aspect <= 0 {
		aspect = 1
	}
	rows := int(math.Round(float64(width) / aspect))
	if rows < 6 {
		rows = 6
	}
	if rows > 400 {
		rows = 400
	}
	return rows
}

// sceneName is built once per id and cached, since the gallery asks for names
// while rendering cards.
func sceneName(id int) string {
	if e := photoAt(id); e != nil {
		return e.def.name
	}
	sceneNameMu.Lock()
	defer sceneNameMu.Unlock()
	if n, ok := sceneNameCache[id]; ok {
		return n
	}
	n := composeName(id)
	sceneNameCache[id] = n
	return n
}

var (
	sceneNameMu    sync.Mutex
	sceneNameCache = make(map[int]string, sceneCount)
)
