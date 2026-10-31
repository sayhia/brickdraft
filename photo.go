package main

import (
	"bytes"
	"embed"
	"image"
	"image/draw"
	"image/jpeg"
	"math"
	"sync"
)

// This file holds the photographic half of the built-in library.
//
// The 1000 generated scenes are drawings. However carefully they are painted,
// every shape in them is a sign for a thing rather than the thing itself: an
// ear is a triangle, a tower is a rectangle, and the palette is chosen for
// graphic punch rather than for light. That reads as an illustration, and no
// amount of geometry fixes it.
//
// A photograph does not have that problem, so the gallery also ships a small
// set of real pictures and pushes them through exactly the same quantiser the
// user's own files go through. That is the honest demo of what the app does,
// and it is the only way a thumbnail can look like a photograph.
//
//go:embed photos/*.jpg
var photoFS embed.FS

// photoDef is one shot in the built-in library. Category reuses the drawing
// categories, so a photograph of a cat appears under 动物 next to the painted
// ones; the Photo flag is what tells the gallery to badge it.
type photoDef struct {
	file string
	name string
	cat  int
	tags []string
}

var photoDefs = []photoDef{
	// 自然
	{"01-valley.jpg", "晨雾山谷", catMountain, []string{"实拍", "山", "湖", "晨雾"}},
	{"02-surf.jpg", "黎明礁石", catMountain, []string{"实拍", "海", "浪", "长曝"}},
	{"03-glacier.jpg", "冰川湖", catMountain, []string{"实拍", "雪山", "湖", "正午"}},
	{"04-creek.jpg", "秋日溪流", catMountain, []string{"实拍", "溪水", "落叶", "秋"}},
	// 动物
	{"05-tabby.jpg", "窗边的橘猫", catAnimal, []string{"实拍", "猫", "窗光"}},
	{"06-retriever.jpg", "草地上的金毛", catAnimal, []string{"实拍", "狗", "草地", "逆光"}},
	{"07-dove.jpg", "石阶上的白鸽", catAnimal, []string{"实拍", "鸽", "灰调"}},
	{"08-kitten.jpg", "毛毯上的小猫", catAnimal, []string{"实拍", "猫", "幼崽", "室内"}},
	// 城市
	{"09-skyline.jpg", "蓝调时刻", catCity, []string{"实拍", "天际线", "夜景", "蓝调"}},
	{"10-alley.jpg", "雨夜街巷", catCity, []string{"实拍", "街", "霓虹", "雨后"}},
	{"11-avenue.jpg", "黄昏大道", catCity, []string{"实拍", "街", "落日", "远景"}},
	{"12-oldtown.jpg", "老城午后", catCity, []string{"实拍", "老街", "晒衣绳", "午后"}},
	// 建筑
	{"13-lighthouse.jpg", "阴天的灯塔", catStructure, []string{"实拍", "灯塔", "海", "阴天"}},
	{"14-cabin.jpg", "草原木屋", catStructure, []string{"实拍", "小屋", "暮色", "草原"}},
	{"15-glasshouse.jpg", "玻璃温室", catStructure, []string{"实拍", "温室", "冬", "对称"}},
	{"16-museum.jpg", "混凝土美术馆", catStructure, []string{"实拍", "美术馆", "混凝土", "极简"}},
	// 美食
	{"17-cake.jpg", "草莓蛋糕", catFood, []string{"实拍", "甜点", "草莓", "木桌"}},
	{"18-ramen.jpg", "一碗拉面", catFood, []string{"实拍", "面", "蒸汽", "暖光"}},
	{"19-bread.jpg", "面包篮", catFood, []string{"实拍", "面包", "亚麻", "俯拍"}},
	{"20-cafe.jpg", "咖啡与可颂", catFood, []string{"实拍", "咖啡", "可颂", "晨光"}},
	// 交通
	{"21-harbour.jpg", "落日港口", catVehicle, []string{"实拍", "船", "落日", "港口"}},
	{"22-bicycle.jpg", "墙边的自行车", catVehicle, []string{"实拍", "自行车", "影子", "晨光"}},
	{"23-hatchback.jpg", "雨后的黄车", catVehicle, []string{"实拍", "汽车", "街", "雨后"}},
	{"24-train.jpg", "乡间列车", catVehicle, []string{"实拍", "火车", "田野", "阴天"}},
	// 植物
	{"25-flowers.jpg", "窗台野花", catBotanical, []string{"实拍", "花", "逆光", "窗"}},
	{"26-succulents.jpg", "多肉方阵", catBotanical, []string{"实拍", "多肉", "陶盆", "侧光"}},
	{"27-sunflowers.jpg", "向日葵田", catBotanical, []string{"实拍", "向日葵", "逆光", "金色"}},
	{"28-moss.jpg", "苔藓与蕨", catBotanical, []string{"实拍", "苔藓", "蕨", "微距"}},
	// 星球
	{"29-saturn.jpg", "带环行星", catSpace, []string{"实拍", "行星", "星环", "深空"}},
	{"30-moonrise.jpg", "月面与地升", catSpace, []string{"实拍", "月面", "地球", "环形山"}},
	{"31-nebula.jpg", "星云", catSpace, []string{"实拍", "星云", "深空", "窄带"}},
	{"32-milkyway.jpg", "银河下的沙漠", catSpace, []string{"实拍", "银河", "沙漠", "长曝"}},
	// 人物
	{"33-profile.jpg", "窗边侧脸", catPortrait, []string{"实拍", "人像", "逆光", "侧脸"}},
	{"34-elder.jpg", "老人", catPortrait, []string{"实拍", "人像", "皱纹", "侧光"}},
	{"35-child.jpg", "窗前的孩子", catPortrait, []string{"实拍", "人像", "儿童", "自然光"}},
	{"36-sunhat.jpg", "草帽", catPortrait, []string{"实拍", "人像", "草帽", "花园"}},
}

// Photo ids live past the generated ones so that scene ids stay comparable and
// the generated range stays exactly sceneCount long.
const firstPhotoID = sceneCount + firstSceneID

var photoCount = len(photoDefs)

// photoEntry decodes its JPEG on first use and keeps it: opening the gallery
// must not decode nine photographs, and a picture that has been looked at is
// looked at again.
type photoEntry struct {
	def   photoDef
	mu    sync.Mutex
	tried bool
	img   *image.RGBA
}

var photoTable = func() []*photoEntry {
	t := make([]*photoEntry, len(photoDefs))
	for i, d := range photoDefs {
		t[i] = &photoEntry{def: d}
	}
	return t
}()

func isPhotoID(id int) bool { return id >= firstPhotoID && id < firstPhotoID+photoCount }

// photoAt returns the entry for a photo id, or nil for a generated scene.
func photoAt(id int) *photoEntry {
	if !isPhotoID(id) {
		return nil
	}
	return photoTable[id-firstPhotoID]
}

// picture is the decoded photograph, already in RGBA so that sampling does not
// go through the image.Image interface for every cell.
func (e *photoEntry) picture() *image.RGBA {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.tried {
		return e.img
	}
	e.tried = true

	raw, err := photoFS.ReadFile("photos/" + e.def.file)
	if err != nil {
		return nil
	}
	src, err := jpeg.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil
	}
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Src)
	e.img = dst
	return e.img
}

// photoShade samples the photograph once per cell.
//
// Three points per axis is the sweet spot: at the 64-cell grid a cell covers
// eight pixels of a 512px shot, so a 3x3 stencil already averages most of the
// area, and at 128 cells it degrades to a centre sample, which is all the
// quantiser can resolve anyway.
func photoShade(e *photoEntry) func(x, y, w, h float64) [3]float64 {
	img := e.picture()
	if img == nil {
		grey := [3]float64{128, 128, 128}
		return func(x, y, w, h float64) [3]float64 { return grey }
	}
	sw, sh := float64(img.Bounds().Dx()), float64(img.Bounds().Dy())
	pix := img.Pix
	stride := img.Stride

	return func(x, y, w, hh float64) [3]float64 {
		if w <= 0 || hh <= 0 {
			return [3]float64{128, 128, 128}
		}
		u0 := (x - 0.5) / w
		v0 := (y - 0.5) / hh
		du, dv := 1/w, 1/hh
		const n = 3
		var sr, sg, sb float64
		for j := 0; j < n; j++ {
			py := int((v0 + dv*(float64(j)+0.5)/n) * sh)
			if py < 0 {
				py = 0
			}
			if py >= int(sh) {
				py = int(sh) - 1
			}
			row := py * stride
			for i := 0; i < n; i++ {
				px := int((u0 + du*(float64(i)+0.5)/n) * sw)
				if px < 0 {
					px = 0
				}
				if px >= int(sw) {
					px = int(sw) - 1
				}
				o := row + px*4
				sr += float64(pix[o])
				sg += float64(pix[o+1])
				sb += float64(pix[o+2])
			}
		}
		k := 1.0 / (n * n)
		return [3]float64{sr * k, sg * k, sb * k}
	}
}

// photoScaled resamples the photograph with bilinear filtering. The quantiser
// averages whatever it is handed, so feeding it real pixels keeps detail that
// sampling straight to cell centres would throw away.
func photoScaled(src *image.RGBA, w, h int) *image.RGBA {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	if src == nil {
		return image.NewRGBA(image.Rect(0, 0, w, h))
	}
	b := src.Bounds()
	sw, sh := b.Dx(), b.Dy()
	if sw == 0 || sh == 0 {
		return image.NewRGBA(image.Rect(0, 0, w, h))
	}

	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	scaleX := float64(sw) / float64(w)
	scaleY := float64(sh) / float64(h)
	pix, stride := src.Pix, src.Stride
	out := dst.Pix
	outStride := dst.Stride

	for y := 0; y < h; y++ {
		fy := (float64(y)+0.5)*scaleY - 0.5
		y0 := int(math.Floor(fy))
		ty := fy - float64(y0)
		if y0 < 0 {
			y0, ty = 0, 0
		}
		y1 := y0 + 1
		if y1 >= sh {
			y1 = sh - 1
		}
		row0, row1 := y0*stride, y1*stride
		dstRow := y * outStride
		for x := 0; x < w; x++ {
			fx := (float64(x)+0.5)*scaleX - 0.5
			x0 := int(math.Floor(fx))
			tx := fx - float64(x0)
			if x0 < 0 {
				x0, tx = 0, 0
			}
			x1 := x0 + 1
			if x1 >= sw {
				x1 = sw - 1
			}
			o00, o10 := row0+x0*4, row0+x1*4
			o01, o11 := row1+x0*4, row1+x1*4
			o := dstRow + x*4
			for c := 0; c < 3; c++ {
				top := float64(pix[o00+c])*(1-tx) + float64(pix[o10+c])*tx
				bot := float64(pix[o01+c])*(1-tx) + float64(pix[o11+c])*tx
				out[o+c] = uint8(clampF(math.Round(top*(1-ty)+bot*ty), 0, 255))
			}
			out[o+3] = 255
		}
	}
	return dst
}

// encodeJPEG keeps photographic thumbnails small: a 256px PNG of a real scene
// runs to six figures of base64, a JPEG to four.
func encodeJPEG(img image.Image, quality int) ([]byte, error) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
