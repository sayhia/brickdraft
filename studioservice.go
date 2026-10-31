package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"image/gif"
	"image/jpeg"

	"github.com/wailsapp/wails/v3/pkg/application"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/webp"
)

// StudioService is the Go side of 像素积木.
type StudioService struct {
	mu      sync.Mutex
	img     image.Image
	imgName string
	imgPath string
	srcData []byte
	srcMime string
	// demo is the built-in scene currently loaded (ID 0 for real files).
	//
	// Scenes are redrawn on demand at the requested grid size instead of being
	// resampled, so they stay pixel-clean at every width: resampling a 64-cell
	// picture up to 200 would blend two brick colours into a third, which is
	// exactly the muddy edge a brick mosaic is judged on.
	demo scene
}

// ImageInfo describes the currently loaded source picture.
type ImageInfo struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Width   int    `json:"width"`
	Height  int    `json:"height"`
	Preview string `json:"preview"`
	Size    int64  `json:"size"`
}

// StartupData is everything the UI needs on boot.
type StartupData struct {
	Version  string          `json:"version"`
	Image    *ImageInfo      `json:"image"`
	Mosaic   *Mosaic         `json:"mosaic"`
	Palettes []PaletteInfo   `json:"palettes"`
	Modes    []BrickModeInfo `json:"modes"`
	Demos    []SceneInfo     `json:"demos"`
	StudMM   float64         `json:"studMm"`
}

// RelayoutRequest carries an edited cell grid.
type RelayoutRequest struct {
	Width    int            `json:"width"`
	Height   int            `json:"height"`
	Cells    []int          `json:"cells"`
	Settings MosaicSettings `json:"settings"`
}

// ExportRequest asks for a rendered file.
type ExportRequest struct {
	Design  ExportDesign  `json:"design"`
	Options ExportOptions `json:"options"`
}

// ExportResult reports where a file was written.
type ExportResult struct {
	Path     string `json:"path"`
	Canceled bool   `json:"canceled"`
	Bytes    int    `json:"bytes"`
}

// ProjectFile is the on-disk document format.
type ProjectFile struct {
	Version  int            `json:"version"`
	Name     string         `json:"name"`
	SavedAt  string         `json:"savedAt"`
	Settings MosaicSettings `json:"settings"`
	Width    int            `json:"width"`
	Height   int            `json:"height"`
	Cells    []int          `json:"cells"`
	Source   ProjectSource  `json:"source"`
}

// ProjectSource keeps the original picture so a project can be reopened alone.
type ProjectSource struct {
	Name    string `json:"name"`
	DataURL string `json:"dataUrl"`
}

// The built-in scenes are authored at brick resolution: 64 cells wide.
const (
	demoCols = 64
	demoRows = 48
)

func defaultSettings() MosaicSettings {
	return MosaicSettings{
		Width:      demoCols,
		Height:     0,
		PaletteID:  "classic",
		BrickMode:  "micro",
		Dither:     false,
		Brightness: 0,
		Contrast:   0,
		Saturation: 0,
		Background: "#FFFFFF",
	}
}

func (s *StudioService) setImage(img image.Image, path string, data []byte, mime string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.img = img
	s.imgPath = path
	s.imgName = filepath.Base(path)
	s.demo = scene{}
	s.srcData = data
	s.srcMime = mime
}

// setDemo stores a freshly drawn built-in scene.
func (s *StudioService) setDemo(sc scene, img image.Image, data []byte, mime string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.img = img
	s.imgPath = ""
	s.imgName = "示例 · " + sceneName(sc.ID)
	s.demo = sc
	s.srcData = data
	s.srcMime = mime
}

func (s *StudioService) currentImage() (image.Image, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.img == nil {
		return nil, fmt.Errorf("还没有载入图片")
	}
	return s.img, nil
}

// sourceImage returns the picture to quantise, drawing the demo scene afresh
// when that is what is loaded.
func (s *StudioService) sourceImage(settings MosaicSettings) (image.Image, error) {
	s.mu.Lock()
	demo := s.demo
	img := s.img
	s.mu.Unlock()

	if demo.ID != 0 {
		width := settings.Width
		if width <= 0 {
			width = demoCols
		}
		return demoFor(demo, width), nil
	}
	if img == nil {
		return nil, fmt.Errorf("还没有载入图片")
	}
	return img, nil
}

// ---------------------------------------------------------------- lifecycle

// Startup loads the default scene and returns the full initial UI payload.
func (s *StudioService) Startup() (*StartupData, error) {
	// Open on a photograph. The first thing the app shows should be what it is
	// good at - a real picture turned into bricks - not a drawing.
	sc := sceneByID(firstPhotoID)
	img, data, mime := scenePayload(sc, demoCols)
	s.setDemo(sc, img, data, mime)

	settings := defaultSettings()
	m := buildMosaic(img, settings)
	info := imageInfo(img, "示例 · "+sceneName(sc.ID), "", 0, nil, mime)

	return &StartupData{
		Version:  "1.0.0",
		Image:    info,
		Mosaic:   &m,
		Palettes: allPalettes(),
		Modes:    brickModes(),
		Demos:    sceneCatalogue(),
		StudMM:   studMM,
	}, nil
}

// Palettes lists every available LEGO colour set.
func (s *StudioService) Palettes() []PaletteInfo { return allPalettes() }

// Modes lists every brick set.
func (s *StudioService) Modes() []BrickModeInfo { return brickModes() }

// OpenImage prompts for a picture and loads it.
func (s *StudioService) OpenImage() (*ImageInfo, error) {
	app := application.Get()
	if app == nil {
		return nil, fmt.Errorf("应用尚未就绪")
	}
	path, err := app.Dialog.OpenFile().
		SetTitle("选择图片").
		AddFilter("图片文件", "*.png;*.jpg;*.jpeg;*.gif;*.bmp;*.webp").
		AddFilter("所有文件", "*.*").
		PromptForSingleSelection()
	if err != nil {
		return nil, err
	}
	if path == "" {
		return nil, nil // user cancelled
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}
	img, mime, err := decodeImage(data)
	if err != nil {
		return nil, err
	}
	s.setImage(img, path, data, mime)
	st, _ := os.Stat(path)
	var size int64
	if st != nil {
		size = st.Size()
	}
	return imageInfo(img, filepath.Base(path), path, size, data, mime), nil
}

// Demo reloads the default built-in scene.
func (s *StudioService) Demo(settings MosaicSettings) (*Mosaic, error) {
	return s.LoadDemo(firstSceneID, settings)
}

// LoadDemo draws one of the built-in scenes at the requested grid size.
func (s *StudioService) LoadDemo(id int, settings MosaicSettings) (*Mosaic, error) {
	sc := sceneByID(id)
	width := settings.Width
	if width <= 0 {
		width = demoCols
	}
	img, data, mime := scenePayload(sc, width)
	s.setDemo(sc, img, data, mime)
	m := buildMosaic(img, settings)
	return &m, nil
}

// SceneCount reports how many pictures the built-in library holds.
func (s *StudioService) SceneCount() int { return sceneCount + photoCount }

// SceneThumbs renders the previews the gallery is about to show. The UI asks
// for the visible window only, so a 1000-entry gallery still starts instantly
// and browsing never renders what nobody looked at.
func (s *StudioService) SceneThumbs(ids []int) []SceneInfo {
	out := make([]SceneInfo, 0, len(ids))
	for _, id := range ids {
		sc := sceneByID(id)
		b := sceneBlobFor(sc)
		info := b.info
		info.Thumb = b.thumb
		out = append(out, info)
	}
	return out
}

// CurrentImage returns metadata for the picture currently in memory.
func (s *StudioService) CurrentImage() (*ImageInfo, error) {
	s.mu.Lock()
	img := s.img
	name := s.imgName
	path := s.imgPath
	raw := s.srcData
	mime := s.srcMime
	s.mu.Unlock()
	if img == nil {
		return nil, fmt.Errorf("还没有载入图片")
	}
	return imageInfo(img, name, path, int64(len(raw)), raw, mime), nil
}

// Generate converts the loaded picture using the given settings.
func (s *StudioService) Generate(settings MosaicSettings) (*Mosaic, error) {
	img, err := s.sourceImage(settings)
	if err != nil {
		return nil, err
	}
	if settings.PaletteID == "" {
		settings.PaletteID = "classic"
	}
	if settings.BrickMode == "" {
		settings.BrickMode = "micro"
	}
	m := buildMosaic(img, settings)
	return &m, nil
}

// Relayout recomputes bricks, parts and stats after manual edits.
func (s *StudioService) Relayout(req RelayoutRequest) (*Layout, error) {
	layout := relayout(req.Width, req.Height, req.Cells, req.Settings)
	return &layout, nil
}

// ------------------------------------------------------------------- export

func (s *StudioService) saveDialog(title, defaultName, filterName, pattern string) (string, error) {
	app := application.Get()
	if app == nil {
		return "", fmt.Errorf("应用尚未就绪")
	}
	path, err := app.Dialog.SaveFile().
		SetMessage(title).
		SetButtonText("保存").
		SetFilename(defaultName).
		AddFilter(filterName, pattern).
		PromptForSingleSelection()
	if err != nil {
		return "", err
	}
	if path != "" && !strings.Contains(filepath.Base(path), ".") {
		path += strings.TrimPrefix(pattern, "*")
	}
	return path, nil
}

func safeName(name string) string {
	if name == "" {
		return "像素积木"
	}
	bad := []string{"\\", "/", ":", "*", "?", "\"", "<", ">", "|"}
	out := name
	for _, b := range bad {
		out = strings.ReplaceAll(out, b, "_")
	}
	return out
}

// Export renders the design to PNG / SVG / CSV and writes it where the user asks.
func (s *StudioService) Export(req ExportRequest) (*ExportResult, error) {
	d := req.Design
	if d.Width <= 0 || d.Height <= 0 {
		return nil, fmt.Errorf("设计为空")
	}
	if d.Settings.PaletteID == "" {
		d.Settings = defaultSettings()
	}
	name := safeName(d.Name)

	var (
		path     string
		payload  []byte
		err      error
		defName  string
		filterNm string
		pattern  string
	)

	switch req.Options.Kind {
	case "png":
		defName = name + "-效果图.png"
		filterNm, pattern = "PNG 图片", "*.png"
	case "svg-sheet":
		defName = name + "-施工图.svg"
		filterNm, pattern = "SVG 矢量图", "*.svg"
	case "svg-steps":
		defName = name + "-分步图.svg"
		filterNm, pattern = "SVG 矢量图", "*.svg"
	case "csv":
		defName = name + "-零件清单.csv"
		filterNm, pattern = "CSV 表格", "*.csv"
	default:
		return nil, fmt.Errorf("不支持的导出类型: %s", req.Options.Kind)
	}

	path, err = s.saveDialog("导出 "+defName, defName, filterNm, pattern)
	if err != nil {
		return nil, err
	}
	if path == "" {
		return &ExportResult{Canceled: true}, nil
	}

	switch req.Options.Kind {
	case "png":
		img := renderPNG(d, req.Options)
		var buf bytes.Buffer
		enc := png.Encoder{CompressionLevel: png.BestSpeed}
		if err := enc.Encode(&buf, img); err != nil {
			return nil, err
		}
		payload = buf.Bytes()
	case "svg-sheet":
		payload = []byte(renderSheetSVG(d))
	case "svg-steps":
		payload = []byte(renderStepsSVG(d))
	case "csv":
		payload = []byte(renderCSV(d))
	}

	if err := os.WriteFile(path, payload, 0o644); err != nil {
		return nil, fmt.Errorf("写入失败: %w", err)
	}
	return &ExportResult{Path: path, Bytes: len(payload)}, nil
}

// SaveProject writes the editable document (.xiangsu, JSON inside).
func (s *StudioService) SaveProject(name string, d ExportDesign) (string, error) {
	s.mu.Lock()
	src := s.srcData
	mime := s.srcMime
	srcName := s.imgName
	s.mu.Unlock()

	proj := ProjectFile{
		Version:  1,
		Name:     name,
		SavedAt:  time.Now().Format(time.RFC3339),
		Settings: d.Settings,
		Width:    d.Width,
		Height:   d.Height,
		Cells:    normalizeCells(d),
		Source: ProjectSource{
			Name: srcName,
		},
	}
	if len(src) > 0 {
		if mime == "" {
			mime = "image/png"
		}
		proj.Source.DataURL = "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(src)
	}

	payload, err := json.MarshalIndent(proj, "", "  ")
	if err != nil {
		return "", err
	}
	path, err := s.saveDialog("保存工程", safeName(name)+".xiangsu", "像素积木工程", "*.xiangsu")
	if err != nil {
		return "", err
	}
	if path == "" {
		return "", nil
	}
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		return "", fmt.Errorf("写入失败: %w", err)
	}
	return path, nil
}

// OpenProject loads a .xiangsu document and restores its source picture.
func (s *StudioService) OpenProject() (*ProjectFile, error) {
	app := application.Get()
	if app == nil {
		return nil, fmt.Errorf("应用尚未就绪")
	}
	path, err := app.Dialog.OpenFile().
		SetTitle("打开工程").
		AddFilter("像素积木工程", "*.xiangsu;*.json").
		PromptForSingleSelection()
	if err != nil {
		return nil, err
	}
	if path == "" {
		return nil, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var proj ProjectFile
	if err := json.Unmarshal(raw, &proj); err != nil {
		return nil, fmt.Errorf("工程文件已损坏: %w", err)
	}
	if proj.Source.DataURL != "" {
		if data, mime, err := decodeDataURL(proj.Source.DataURL); err == nil {
			if img, _, err := decodeImage(data); err == nil {
				s.setImage(img, path, data, mime)
				s.mu.Lock()
				s.imgName = proj.Source.Name
				s.mu.Unlock()
			}
		}
	}
	return &proj, nil
}

// RevealFile opens the file manager at the given path.
func (s *StudioService) RevealFile(path string) error {
	if path == "" {
		return fmt.Errorf("路径为空")
	}
	switch runtime.GOOS {
	case "windows":
		return exec.Command("explorer", "/select,", path).Start()
	case "darwin":
		return exec.Command("open", "-R", path).Start()
	default:
		return exec.Command("xdg-open", filepath.Dir(path)).Start()
	}
}

// ------------------------------------------------------------------ helpers

func decodeImage(data []byte) (image.Image, string, error) {
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, "", fmt.Errorf("无法识别的图片格式: %w", err)
	}
	mime := "image/" + format
	switch format {
	case "jpeg":
		mime = "image/jpeg"
	}
	return img, mime, nil
}

func decodeDataURL(s string) ([]byte, string, error) {
	i := strings.Index(s, "base64,")
	if i < 0 {
		return nil, "", fmt.Errorf("无效的 data URL")
	}
	mime := strings.TrimPrefix(s[:i], "data:")
	mime = strings.TrimSuffix(mime, ";")
	raw, err := base64.StdEncoding.DecodeString(s[i+len("base64,"):])
	if err != nil {
		return nil, "", err
	}
	return raw, mime, nil
}

func encodePNG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func imageInfo(img image.Image, name, path string, size int64, raw []byte, mime string) *ImageInfo {
	thumb := thumbnail(img, 260)
	var dataURL string
	if tb, err := encodePNG(thumb); err == nil {
		dataURL = "data:image/png;base64," + base64.StdEncoding.EncodeToString(tb)
	} else if len(raw) > 0 {
		dataURL = "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(raw)
	}
	return &ImageInfo{
		Name:    name,
		Path:    path,
		Width:   img.Bounds().Dx(),
		Height:  img.Bounds().Dy(),
		Preview: dataURL,
		Size:    size,
	}
}

// thumbnail box-samples the image down to at most maxSide pixels.
func thumbnail(src image.Image, maxSide int) image.Image {
	b := src.Bounds()
	sw, sh := b.Dx(), b.Dy()
	scale := float64(maxSide) / float64(maxInt(sw, sh))
	if scale >= 1 {
		return src
	}
	dw := maxInt(1, int(float64(sw)*scale))
	dh := maxInt(1, int(float64(sh)*scale))
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	for y := 0; y < dh; y++ {
		for x := 0; x < dw; x++ {
			sx := b.Min.X + x*sw/dw
			sy := b.Min.Y + y*sh/dh
			dst.Set(x, y, src.At(sx, sy))
		}
	}
	return dst
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// keep gif/jpeg decoders linked even though we only reference them via image.Decode
var (
	_ = gif.Decode
	_ = jpeg.Decode
)
