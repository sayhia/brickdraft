package main

// A mood is a small, curated set of real brick colours plus a sky band. Every
// scene pulls its palette from one mood, which is what keeps 1000 generated
// pictures looking like a series instead of a pile of noise: inside a scene the
// colours are guaranteed to sit well together.
type mood struct {
	name  string
	sky   []string // zenith to horizon, 1-4 entries
	earth []string // ground, foliage, water, stone
	warm  []string // accents, highlights, fire, flowers
	lit   []string // skyline lighting, lamps, screens
}

var moods = []mood{
	{
		name:  "晴日",
		sky:   []string{"azure", "lblue", "white"},
		earth: []string{"green", "bgreen", "lime", "dgreen"},
		warm:  []string{"lyellow", "lorange", "yellow", "white"},
		lit:   []string{"white", "lyellow", "sandblue"},
	},
	{
		name:  "日落",
		sky:   []string{"dpurple", "magenta", "coral", "orange"},
		earth: []string{"dblue", "dpurple", "purple", "black"},
		warm:  []string{"lyellow", "lorange", "yellow"},
		lit:   []string{"lorange", "lyellow", "tan"},
	},
	{
		name:  "暮色",
		sky:   []string{"dblue", "purple", "magenta", "magenta"},
		earth: []string{"black", "dblue", "dbrown", "dgray"},
		warm:  []string{"lorange", "lyellow", "tan"},
		lit:   []string{"lorange", "lyellow", "lyellow", "tan"},
	},
	{
		name:  "深夜",
		sky:   []string{"black", "dblue", "dblue", "dpurple"},
		earth: []string{"black", "dblue", "dgray", "dbrown"},
		warm:  []string{"lyellow", "white", "lorange"},
		lit:   []string{"lyellow", "lyellow", "lorange", "white"},
	},
	{
		name:  "雪原",
		sky:   []string{"white", "vlgray", "sandblue"},
		earth: []string{"white", "vlgray", "lgray", "sandblue"},
		warm:  []string{"dgray", "black", "tan"},
		lit:   []string{"lorange", "lyellow", "white"},
	},
	{
		name: "樱花",
		// a blossom sky is pale, not a magenta wall: the pink belongs in the
		// band just above the horizon, under a washed lavender zenith
		sky:   []string{"lavender", "pink", "white"},
		earth: []string{"dgreen", "bgreen", "green"},
		warm:  []string{"white", "pink", "yellow", "dpink"},
		lit:   []string{"white", "lyellow", "pink"},
	},
	{
		name:  "海洋",
		sky:   []string{"sandblue", "lblue", "white"},
		earth: []string{"dteal", "azure", "blue", "dazure"},
		warm:  []string{"white", "tan", "lyellow"},
		lit:   []string{"white", "lyellow", "azure"},
	},
	{
		name:  "秋林",
		sky:   []string{"tan", "lorange", "lyellow"},
		earth: []string{"olive", "dtan", "dorange", "redbrown"},
		warm:  []string{"red", "dorange", "yellow"},
		lit:   []string{"lyellow", "lorange", "white"},
	},
	{
		name:  "霓虹",
		sky:   []string{"black", "dpurple", "purple", "magenta"},
		earth: []string{"black", "dgray", "dpurple"},
		warm:  []string{"dazure", "lime", "coral", "lyellow"},
		lit:   []string{"dazure", "lime", "coral", "lyellow", "magenta"},
	},
	{
		name:  "沙漠",
		sky:   []string{"lorange", "tan", "yellow"},
		earth: []string{"tan", "dtan", "dorange", "nougat"},
		warm:  []string{"lyellow", "red", "white"},
		lit:   []string{"lyellow", "white", "lorange"},
	},
	{
		name:  "极地",
		sky:   []string{"dblue", "sandblue", "lblue"},
		earth: []string{"white", "lblue", "sandblue", "black"},
		warm:  []string{"lyellow", "white", "azure"},
		lit:   []string{"lyellow", "white", "azure"},
	},
	{
		name:  "苔原",
		sky:   []string{"vlgray", "sandgreen", "tan"},
		earth: []string{"olive", "dgreen", "sandgreen", "black"},
		warm:  []string{"tan", "lyellow", "dtan"},
		lit:   []string{"lyellow", "lorange", "white"},
	},
}

const moodNight = 3

// Subject category ids. The order must match the categories slice assembled in
// painter.go's init, which is what sceneCat indexes into.
const (
	catMountain = iota
	catAnimal
	catCity
	catStructure
	catFood
	catVehicle
	catBotanical
	catSpace
	catPortrait
)

// moodColour resolves a mood palette entry to RGB; the index wraps, so callers
// can ask for "the nth accent" without bounds checks.
func moodColour(list []string, i int) [3]float64 {
	if len(list) == 0 {
		return [3]float64{128, 128, 128}
	}
	if i < 0 {
		i = -i
	}
	return colorOf(list[i%len(list)])
}

func moodHas(m mood, id string) bool {
	for _, list := range [][]string{m.sky, m.earth, m.warm, m.lit} {
		for _, c := range list {
			if c == id {
				return true
			}
		}
	}
	return false
}

// ---------------------------------------------------------------- categories

type catWord struct{ adj, noun string }

// moodFloorFor keeps a night-only subject away from daylight palettes: a planet
// in a "clear morning" mood reads as a balloon. Portraits used to be here too;
// they are now sticker characters, and a child's face is not improved by being
// lit only by stars.
func moodFloorFor(cat int) int {
	switch cat {
	case catSpace:
		return moodNight
	}
	return moodAuto
}

// category is one kind of subject, and most kinds hold several characters: the
// animal category draws a cat on one id and a rabbit on the next.
type category struct {
	name string
	word catWord
	draw drawFn
	subs []subject
}

// pick chooses which character of the category a scene shows.
//
// The stride is large and coprime with the usual sub-counts (3, 4, 7, 11), so
// consecutive ids walk the list instead of repeating one animal for seven
// frames in a row - which is what a plain id%n does when the category wheel and
// the subject wheel share a period.
func (c category) pick(id int) drawFn {
	if len(c.subs) == 0 {
		return c.draw
	}
	n := (id - firstSceneID + sceneCount) % sceneCount
	return c.subs[(n*7+n/len(c.subs))%len(c.subs)].draw
}

// subName is the name of the character a scene shows, for the gallery label.
func (c category) subName(id int) string {
	if len(c.subs) == 0 {
		return c.word.noun
	}
	n := (id - firstSceneID + sceneCount) % sceneCount
	return c.subs[(n*7+n/len(c.subs))%len(c.subs)].name
}

// ------------------------------------------------------------------- names

// kidAdjectives are the names a picture card gets in this library: an adjective
// and the character, so the gallery reads as a sticker book ("笑眯眯的小猫")
// rather than as an art print ("落日 · 群山之一").
var kidAdjectives = []string{
	"笑眯眯的", "圆滚滚的", "好奇的", "开心的", "勇敢的",
	"安静的", "小小的", "神气的", "胖乎乎的", "甜甜的",
	"戴帽子的", "睡午觉的",
}

// composeName makes a stable, human-readable name for a scene id. The noun is
// the actual character drawn, so the label never disagrees with the picture.
func composeName(id int) string {
	n := (id - firstSceneID + sceneCount) % sceneCount
	cat := categories[sceneCat(id)]
	adj := kidAdjectives[(n*13+id)%len(kidAdjectives)]
	return adj + cat.subName(id)
}

// sceneByID maps the id the UI sends to a scene, clamping out-of-range input.
// Photo ids are valid even though they are outside the generated range.
func sceneByID(id int) scene {
	if !isPhotoID(id) && (id < firstSceneID || id >= firstSceneID+sceneCount) {
		id = firstSceneID
	}
	return scene{ID: id, Mood: moodAuto}
}

// sceneCatalogue is the whole gallery, without thumbnails (built on demand).
//
// The photographs come first. They are the strongest argument for the app -
// they show what a real picture looks like once it is a mosaic - so they should
// be what the gallery opens on, not page 21 of 21.
func sceneCatalogue() []SceneInfo {
	out := make([]SceneInfo, 0, sceneCount+photoCount)
	for i := 0; i < photoCount; i++ {
		out = append(out, sceneInfo(sceneByID(firstPhotoID+i)))
	}
	for i := 0; i < sceneCount; i++ {
		out = append(out, sceneInfo(sceneByID(firstSceneID+i)))
	}
	return out
}
