import { reactive } from 'vue'
import { StudioService } from '../../bindings/xiangsu'
import type {
  Mosaic,
  MosaicSettings,
  PaletteInfo,
  BrickModeInfo,
  PartRow,
  ColorUsage,
  Brick,
  Stats,
  ImageInfo,
  SceneInfo,
} from '../../bindings/xiangsu'

export type Tool = 'brush' | 'picker' | 'pan'
export type ExportKind = 'png' | 'svg-sheet' | 'svg-steps' | 'csv'

/** Settings mirrored to the Go renderer. */
export const settings = reactive<MosaicSettings>({
  width: 64,
  height: 0,
  paletteId: 'classic',
  brickMode: 'micro',
  dither: false,
  brightness: 0,
  contrast: 0,
  saturation: 0,
  background: '#FFFFFF',
})

/** Transient UI state. */
export const ui = reactive({
  booting: true,
  rendering: false,
  exporting: false,
  tool: 'brush' as Tool,
  brushColor: 0,
  focusColor: null as number | null,
  showGrid: true,
  showStuds: true,
  view: 'render' as 'render' | 'flat',
  projectName: '落日山峦',
  dirty: false,
  zoom: 18,
  tab: 'colors' as 'colors' | 'parts',
  fits: 0,
  demoOpen: false,
  demoQuery: '',
  demoCat: -1,
  sceneId: 0,
})

/** The current design, as returned by Go. */
export const data = reactive({
  width: 0,
  height: 0,
  palette: null as PaletteInfo | null,
  bricks: [] as Brick[],
  parts: [] as PartRow[],
  colors: [] as ColorUsage[],
  stats: null as Stats | null,
  image: null as ImageInfo | null,
  palettes: [] as PaletteInfo[],
  modes: [] as BrickModeInfo[],
  demos: [] as SceneInfo[],
  version: 0,
})

let cells = new Int16Array(0)

export function cellAt(x: number, y: number): number {
  if (x < 0 || y < 0 || x >= data.width || y >= data.height) return -2
  return cells[y * data.width + x]
}

export function cellsArray(): number[] {
  return Array.from(cells)
}

/** Live cell buffer (mutable, not reactive — bump data.version to redraw). */
export function getCells(): Int16Array {
  return cells
}

export function colorHex(index: number): string {
  const c = data.palette?.colors?.[index]
  return c ? c.hex : '#CCCCCC'
}

export function colorName(index: number): string {
  const c = data.palette?.colors?.[index]
  return c ? c.name : '—'
}

/* ------------------------------------------------------------------ toast */

export interface ToastItem {
  id: number
  text: string
  kind: 'info' | 'error'
  action?: { label: string; run: () => void }
}

export const toasts = reactive<ToastItem[]>([])
let toastSeq = 0

export function notify(text: string, kind: 'info' | 'error' = 'info', action?: ToastItem['action']) {
  const id = ++toastSeq
  toasts.push({ id, text, kind, action })
  window.setTimeout(
    () => {
      const i = toasts.findIndex((t) => t.id === id)
      if (i >= 0) toasts.splice(i, 1)
    },
    kind === 'error' ? 6200 : 4200,
  )
}

export function dismissToast(id: number) {
  const i = toasts.findIndex((t) => t.id === id)
  if (i >= 0) toasts.splice(i, 1)
}

function errText(e: unknown): string {
  if (typeof e === 'string') return e
  if (e && typeof e === 'object' && 'message' in e) return String((e as { message: unknown }).message)
  return String(e)
}

/* ------------------------------------------------------------------ edits */

interface Edit {
  i: number
  before: number
  after: number
}

const undoStack: Edit[][] = []
const redoStack: Edit[][] = []
let stroke: Edit[] = []

export const history = reactive({ canUndo: false, canRedo: false })

function syncHistory() {
  history.canUndo = undoStack.length > 0
  history.canRedo = redoStack.length > 0
}

export function beginStroke() {
  stroke = []
}

export function paintCell(x: number, y: number, color: number) {
  if (x < 0 || y < 0 || x >= data.width || y >= data.height) return
  const i = y * data.width + x
  const before = cells[i]
  if (before === color) return
  cells[i] = color
  stroke.push({ i, before, after: color })
  ui.dirty = true
  data.version++
}

export function endStroke() {
  if (!stroke.length) return
  undoStack.push(stroke)
  if (undoStack.length > 80) undoStack.shift()
  redoStack.length = 0
  stroke = []
  syncHistory()
  scheduleRelayout()
}

export function undo() {
  const batch = undoStack.pop()
  if (!batch) return
  for (let k = batch.length - 1; k >= 0; k--) cells[batch[k].i] = batch[k].before
  redoStack.push(batch)
  syncHistory()
  ui.dirty = true
  data.version++
  scheduleRelayout()
}

export function redo() {
  const batch = redoStack.pop()
  if (!batch) return
  for (const e of batch) cells[e.i] = e.after
  undoStack.push(batch)
  syncHistory()
  ui.dirty = true
  data.version++
  scheduleRelayout()
}

function resetHistory() {
  undoStack.length = 0
  redoStack.length = 0
  stroke = []
  syncHistory()
}

/* ---------------------------------------------------------------- helpers */

function debounce<T extends (...args: any[]) => void>(fn: T, ms: number) {
  let timer: number | undefined
  return (...args: Parameters<T>) => {
    if (timer) window.clearTimeout(timer)
    timer = window.setTimeout(() => fn(...args), ms)
  }
}

/* ------------------------------------------------------------- operations */

function applyMosaic(m: Mosaic) {
  data.width = m.width
  data.height = m.height
  cells = Int16Array.from(m.cells ?? [])
  data.palette = m.palette
  data.bricks = m.bricks ?? []
  data.parts = m.parts ?? []
  data.colors = m.colors ?? []
  data.stats = m.stats
  data.version++
  if (m.settings) {
    settings.width = m.settings.width
    settings.height = m.settings.height
    settings.paletteId = m.settings.paletteId
    settings.brickMode = m.settings.brickMode
  }
  if (data.colors.length && !data.colors.some((c) => c.index === ui.brushColor)) {
    ui.brushColor = data.colors[0].index
  }
  if (ui.focusColor !== null && !data.colors.some((c) => c.index === ui.focusColor)) {
    ui.focusColor = null
  }
  resetHistory()
}

let genSeq = 0

export async function regenerate() {
  if (!data.image) return
  const seq = ++genSeq
  ui.rendering = true
  try {
    const m = await StudioService.Generate({ ...settings })
    if (seq !== genSeq) return
    if (m) applyMosaic(m)
  } catch (e) {
    notify('生成失败：' + errText(e), 'error')
  } finally {
    if (seq === genSeq) ui.rendering = false
  }
}

export const regenerateDebounced = debounce(() => {
  void regenerate()
}, 260)

const relayoutNow = debounce(async () => {
  if (!data.width) return
  try {
    const layout = await StudioService.Relayout({
      width: data.width,
      height: data.height,
      cells: cellsArray(),
      settings: { ...settings },
    })
    if (!layout) return
    data.bricks = layout.bricks ?? []
    data.parts = layout.parts ?? []
    data.colors = layout.colors ?? []
    data.stats = layout.stats
    ui.fits++
  } catch (e) {
    notify('重算零件失败：' + errText(e), 'error')
  }
}, 160)

export function scheduleRelayout() {
  void relayoutNow()
}

export async function boot() {
  try {
    const res = await StudioService.Startup()
    if (!res) return
    data.palettes = res.palettes ?? []
    data.modes = res.modes ?? []
    data.demos = res.demos ?? []
    data.image = res.image
    ui.sceneId = 1
    if (res.mosaic) {
      applyMosaic(res.mosaic)
      settings.width = res.mosaic.width
      // rows follow the picture's aspect until the user pins them
      settings.height = 0
    }
  } catch (e) {
    notify('启动失败：' + errText(e), 'error')
  } finally {
    ui.booting = false
  }
}

export async function openImage() {
  try {
    const info = await StudioService.OpenImage()
    if (!info) return
    data.image = info
    ui.projectName = info.name.replace(/\.[^.]+$/, '')
    ui.dirty = false
    // let the new picture decide its own row count
    settings.height = 0
    await regenerate()
    notify(`已载入 ${info.name} · ${info.width}×${info.height}`)
  } catch (e) {
    notify('载入图片失败：' + errText(e), 'error')
  }
}

export function openDemoGallery() {
  ui.demoOpen = true
}

export function closeDemoGallery() {
  ui.demoOpen = false
}

/* ------------------------------------------------------------- example art */

/** Gallery thumbnails, keyed by scene id. Filled in by the gallery on demand. */
export const thumbs = reactive<Record<number, string>>({})

/**
 * Renders previews for the ids that are not cached yet.
 *
 * The library has a thousand pictures and start-up has to stay instant, so
 * thumbnails are pulled in batches for what is actually on screen. Responses
 * are merged rather than replace anything, so a slow batch landing after a
 * fast one can only ever add pictures, never blank the grid.
 */
export async function ensureThumbs(ids: number[]) {
  const missing = ids.filter((id) => !thumbs[id])
  if (!missing.length) return
  try {
    const list = await StudioService.SceneThumbs(missing)
    for (const it of list ?? []) {
      if (it.thumb) thumbs[it.id] = it.thumb
    }
  } catch {
    // a thumbnail is decoration: never surface this as an error toast
  }
}

/** Loads one of the built-in scenes; a bare call restores the default one. */
export async function loadDemo(id = 1) {
  try {
    ui.rendering = true
    settings.height = 0
    const m = await StudioService.LoadDemo(id, { ...settings })
    if (m) applyMosaic(m)
    const info = await StudioService.CurrentImage()
    if (info) {
      data.image = info
      ui.projectName = info.name.replace(/^示例 · /, '')
      ui.dirty = false
    }
    ui.sceneId = id
    ui.demoOpen = false
    ui.fits++
  } catch (e) {
    notify('示例生成失败：' + errText(e), 'error')
  } finally {
    ui.rendering = false
  }
}

/** Jumps to a random picture: the fastest way to see what the library holds. */
export function randomDemo() {
  const n = data.demos.length || 1
  let id = 1 + Math.floor(Math.random() * n)
  if (id === ui.sceneId && n > 1) id = (id % n) + 1
  void loadDemo(id)
}

export function setWidth(w: number) {
  settings.width = Math.min(320, Math.max(8, Math.round(w)))
  settings.height = 0
  void regenerateDebounced()
}

export function setTone(key: 'brightness' | 'contrast' | 'saturation', v: number) {
  settings[key] = v
  void regenerateDebounced()
}

export function setPalette(id: string) {
  if (settings.paletteId === id) return
  settings.paletteId = id
  ui.focusColor = null
  void regenerateDebounced()
}

export function setBrickMode(id: string) {
  if (settings.brickMode === id) return
  settings.brickMode = id
  void regenerateDebounced()
}

export function toggleDither() {
  settings.dither = !settings.dither
  void regenerateDebounced()
}

export function setBrush(index: number) {
  ui.brushColor = index
  ui.tool = 'brush'
}

export function toggleFocus(index: number) {
  ui.focusColor = ui.focusColor === index ? null : index
}

export function basename(p: string): string {
  const parts = p.split(/[\\/]/)
  return parts[parts.length - 1] || p
}

export async function exportAs(kind: ExportKind) {
  if (!data.width) return
  ui.exporting = true
  try {
    const res = await StudioService.Export({
      design: {
        name: ui.projectName || '像素积木',
        width: data.width,
        height: data.height,
        cells: cellsArray(),
        settings: { ...settings },
      },
      options: {
        kind,
        scale: 26,
        grid: false,
        studs: ui.showStuds,
      },
    })
    if (res && !res.canceled && res.path) {
      const path = res.path
      notify(`已导出 ${basename(path)}`, 'info', {
        label: '打开位置',
        run: () => void StudioService.RevealFile(path),
      })
    }
  } catch (e) {
    notify('导出失败：' + errText(e), 'error')
  } finally {
    ui.exporting = false
  }
}

export async function saveProject() {
  if (!data.width) return
  try {
    const path = await StudioService.SaveProject(ui.projectName || '像素积木', {
      name: ui.projectName,
      width: data.width,
      height: data.height,
      cells: cellsArray(),
      settings: { ...settings },
    })
    if (path) {
      ui.dirty = false
      notify('工程已保存 · ' + basename(path), 'info', {
        label: '打开位置',
        run: () => void StudioService.RevealFile(path),
      })
    }
  } catch (e) {
    notify('保存失败：' + errText(e), 'error')
  }
}

export async function openProject() {
  try {
    const proj = await StudioService.OpenProject()
    if (!proj) return
    settings.width = proj.width
    settings.height = proj.height
    settings.paletteId = proj.settings?.paletteId ?? 'classic'
    settings.brickMode = proj.settings?.brickMode ?? 'micro'
    settings.dither = proj.settings?.dither ?? false
    settings.brightness = proj.settings?.brightness ?? 0
    settings.contrast = proj.settings?.contrast ?? 0
    settings.saturation = proj.settings?.saturation ?? 0
    settings.background = proj.settings?.background ?? '#FFFFFF'
    ui.projectName = proj.name || '未命名作品'

    data.width = proj.width
    data.height = proj.height
    cells = Int16Array.from(proj.cells ?? [])
    data.version++
    resetHistory()

    const layout = await StudioService.Relayout({
      width: proj.width,
      height: proj.height,
      cells: cellsArray(),
      settings: { ...settings },
    })
    if (layout) {
      data.bricks = layout.bricks ?? []
      data.parts = layout.parts ?? []
      data.colors = layout.colors ?? []
      data.stats = layout.stats
    }
    const info = await StudioService.CurrentImage()
    if (info) data.image = info
    ui.dirty = false
    notify(`已打开「${ui.projectName}」`)
  } catch (e) {
    notify('打开工程失败：' + errText(e), 'error')
  }
}
