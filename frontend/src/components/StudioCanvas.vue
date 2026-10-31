<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch, nextTick } from 'vue'
import { MosaicBaker, drawScene, type Camera, type HoverInfo, type Scene } from '../lib/draw'
import {
  data,
  ui,
  cellAt,
  getCells,
  paintCell,
  beginStroke,
  endStroke,
  undo,
  redo,
  setBrush,
  openImage,
  loadDemo,
} from '../lib/store'

const wrap = ref<HTMLElement | null>(null)
const canvas = ref<HTMLCanvasElement | null>(null)
const baker = new MosaicBaker()

const cam = reactive<Camera>({ cell: 16, ox: 0, oy: 0 })
const hover = ref<HoverInfo | null>(null)
const isPanning = ref(false)
const focusName = computed(() => {
  const colors = data.palette?.colors ?? []
  return ui.focusColor !== null ? colors[ui.focusColor]?.name ?? '' : ''
})

let dpr = window.devicePixelRatio || 1
let panning: { x: number; y: number } | null = null
let painting = false
let lastCell: { x: number; y: number } | null = null
let spaceDown = false
let raf = 0

function scene(): Scene {
  return {
    width: data.width,
    height: data.height,
    cells: getCells(),
    colors: (data.palette?.colors ?? []).map((c) => c.hex),
    bricks: data.bricks,
    studs: ui.showStuds,
    view: ui.view,
    focus: ui.focusColor,
  }
}

function scheduleDraw() {
  if (raf) return
  raf = requestAnimationFrame(() => {
    raf = 0
    paint()
  })
}

function paint() {
  const el = canvas.value
  if (!el || !data.width) return
  const s = scene()
  baker.ensure(s, cam.cell, data.version)
  drawScene(el, s, cam, baker, { grid: ui.showGrid, hover: hover.value, dpr })
  ui.zoom = cam.cell
}

function resize() {
  const el = canvas.value
  const box = wrap.value
  if (!el || !box) return
  dpr = window.devicePixelRatio || 1
  const rect = box.getBoundingClientRect()
  el.width = Math.max(1, Math.floor(rect.width * dpr))
  el.height = Math.max(1, Math.floor(rect.height * dpr))
  scheduleDraw()
}

function fit() {
  const box = wrap.value
  if (!box || !data.width) return
  const rect = box.getBoundingClientRect()
  const availW = rect.width - 88
  const availH = rect.height - 88
  const cell = Math.min(availW / data.width, availH / data.height)
  cam.cell = Math.max(2, Math.min(40, cell))
  cam.ox = (rect.width - data.width * cam.cell) / 2
  cam.oy = (rect.height - data.height * cam.cell) / 2
  scheduleDraw()
}

function zoomAt(px: number, py: number, factor: number) {
  const next = Math.max(1.5, Math.min(64, cam.cell * factor))
  const k = next / cam.cell
  cam.ox = px - (px - cam.ox) * k
  cam.oy = py - (py - cam.oy) * k
  cam.cell = next
  scheduleDraw()
}

function zoomBy(factor: number) {
  const box = wrap.value
  if (!box) return
  const rect = box.getBoundingClientRect()
  zoomAt(rect.width / 2, rect.height / 2, factor)
}

function point(e: { clientX: number; clientY: number }) {
  const el = canvas.value!
  const r = el.getBoundingClientRect()
  return { x: e.clientX - r.left, y: e.clientY - r.top }
}

function cellAtPoint(p: { x: number; y: number }) {
  return {
    x: Math.floor((p.x - cam.ox) / cam.cell),
    y: Math.floor((p.y - cam.oy) / cam.cell),
  }
}

function inBounds(c: { x: number; y: number }) {
  return c.x >= 0 && c.y >= 0 && c.x < data.width && c.y < data.height
}

function paintStroke(from: { x: number; y: number }, to: { x: number; y: number }) {
  let { x: x0, y: y0 } = from
  const { x: x1, y: y1 } = to
  const dx = Math.abs(x1 - x0)
  const dy = Math.abs(y1 - y0)
  const sx = x0 < x1 ? 1 : -1
  const sy = y0 < y1 ? 1 : -1
  let err = dx - dy
  for (let guard = 0; guard < 4096; guard++) {
    if (inBounds({ x: x0, y: y0 })) paintCell(x0, y0, ui.brushColor)
    if (x0 === x1 && y0 === y1) break
    const e2 = 2 * err
    if (e2 > -dy) {
      err -= dy
      x0 += sx
    }
    if (e2 < dx) {
      err += dx
      y0 += sy
    }
  }
}

function pickAt(c: { x: number; y: number }) {
  if (!inBounds(c)) return
  const idx = cellAt(c.x, c.y)
  if (idx >= 0) setBrush(idx)
}

function onPointerDown(e: PointerEvent) {
  const el = canvas.value
  if (!el || !data.width) return
  el.setPointerCapture(e.pointerId)
  const p = point(e)
  const c = cellAtPoint(p)

  if (e.button === 1 || e.button === 2 || spaceDown || ui.tool === 'pan') {
    panning = { x: e.clientX, y: e.clientY }
    isPanning.value = true
    return
  }
  if (e.button !== 0) return

  if (e.altKey || ui.tool === 'picker') {
    pickAt(c)
    return
  }
  beginStroke()
  painting = true
  lastCell = c
  paintStroke(c, c)
  scheduleDraw()
}

function onPointerMove(e: PointerEvent) {
  const p = point(e)
  const c = cellAtPoint(p)
  hover.value = inBounds(c) ? c : null

  if (panning) {
    cam.ox += e.clientX - panning.x
    cam.oy += e.clientY - panning.y
    panning = { x: e.clientX, y: e.clientY }
    scheduleDraw()
    return
  }
  if (painting && lastCell) {
    paintStroke(lastCell, c)
    lastCell = c
    scheduleDraw()
    return
  }
  scheduleDraw()
}

function onPointerUp(e: PointerEvent) {
  if (painting) {
    painting = false
    lastCell = null
    endStroke()
  }
  panning = null
  isPanning.value = false
  const el = canvas.value
  if (el && el.hasPointerCapture(e.pointerId)) el.releasePointerCapture(e.pointerId)
}

function onLeave() {
  hover.value = null
  scheduleDraw()
}

function onWheel(e: WheelEvent) {
  e.preventDefault()
  if (!data.width) return
  const p = point(e)
  const factor = Math.exp(-e.deltaY * 0.0016)
  zoomAt(p.x, p.y, factor)
}

function onKeyDown(e: KeyboardEvent) {
  const t = e.target as HTMLElement | null
  if (t && (t.tagName === 'INPUT' || t.tagName === 'TEXTAREA' || t.isContentEditable)) return

  const mod = e.ctrlKey || e.metaKey
  if (mod && e.key.toLowerCase() === 'z') {
    e.preventDefault()
    if (e.shiftKey) redo()
    else undo()
    return
  }
  if (mod && e.key.toLowerCase() === 'y') {
    e.preventDefault()
    redo()
    return
  }
  if (mod) return

  switch (e.key.toLowerCase()) {
    case 'b':
      ui.tool = 'brush'
      break
    case 'i':
      ui.tool = 'picker'
      break
    case 'h':
      ui.tool = 'pan'
      break
    case 'g':
      ui.showGrid = !ui.showGrid
      break
    case ' ':
      spaceDown = true
      e.preventDefault()
      break
    case '+':
    case '=':
      zoomBy(1.2)
      break
    case '-':
      zoomBy(1 / 1.2)
      break
    case '0':
      fit()
      break
  }
}

function onKeyUp(e: KeyboardEvent) {
  if (e.key === ' ') spaceDown = false
}

let ro: ResizeObserver | null = null

onMounted(() => {
  ro = new ResizeObserver(() => resize())
  if (wrap.value) ro.observe(wrap.value)
  resize()
  window.addEventListener('keydown', onKeyDown)
  window.addEventListener('keyup', onKeyUp)
  setTimeout(fit, 60)
})

onBeforeUnmount(() => {
  ro?.disconnect()
  window.removeEventListener('keydown', onKeyDown)
  window.removeEventListener('keyup', onKeyUp)
})

watch(
  () => [data.version, ui.showGrid, ui.showStuds, ui.view, ui.focusColor, data.width, data.bricks] as const,
  () => scheduleDraw(),
)
watch(() => data.image, () => nextTick(fit))
watch(() => ui.fits, () => scheduleDraw())

// A new grid (width preset or slider) changes the artwork's extent, so refit
// instead of leaving it stranded at the previous zoom level.
watch(
  () => [data.width, data.height] as const,
  ([w, h], [pw, ph]) => {
    if (w === pw && h === ph) return
    nextTick(fit)
  },
)

defineExpose({ fit, zoomBy })
</script>

<template>
  <section
    ref="wrap"
    class="canvas-area"
    :class="{ brush: ui.tool === 'brush', picker: ui.tool === 'picker', panning: isPanning }"
  >
    <canvas
      ref="canvas"
      @pointerdown="onPointerDown"
      @pointermove="onPointerMove"
      @pointerup="onPointerUp"
      @pointercancel="onPointerUp"
      @pointerleave="onLeave"
      @wheel="onWheel"
      @dblclick="fit"
      @contextmenu.prevent
    />

    <div class="canvas-toolbar">
      <div class="tool-group">
        <button class="tool-btn" :class="{ active: ui.tool === 'brush' }" title="画笔 (B)" @click="ui.tool = 'brush'">
          <svg viewBox="0 0 24 24" fill="currentColor">
            <path d="M3 17.25V21h3.75L17.8 9.94l-3.74-3.75L3 17.25zM20.7 7.04a1 1 0 000-1.41l-2.33-2.34a1 1 0 00-1.42 0l-1.83 1.83 3.75 3.75 1.83-1.83z" />
          </svg>
        </button>
        <button class="tool-btn" :class="{ active: ui.tool === 'picker' }" title="吸管 (I)" @click="ui.tool = 'picker'">
          <svg viewBox="0 0 24 24" fill="currentColor">
            <path d="M20.7 3.3a2.4 2.4 0 00-3.4 0l-2.2 2.2-1.1-1.1-1.4 1.4 1.1 1.1L4 16.8V20h3.2l9.9-9.9 1.1 1.1 1.4-1.4-1.1-1.1 2.2-2.2a2.4 2.4 0 000-3.2zM6.4 18H6v-.4l7.9-7.9.4.4L6.4 18z" />
          </svg>
        </button>
        <button class="tool-btn" :class="{ active: ui.tool === 'pan' }" title="平移 (H / 空格)" @click="ui.tool = 'pan'">
          <svg viewBox="0 0 24 24" fill="currentColor">
            <path d="M12 2l3 3h-2v5h5V8l3 3-3 3v-2h-5v5h2l-3 3-3-3h2v-5H6v2l-3-3 3-3v2h5V5H9l3-3z" />
          </svg>
        </button>
      </div>
      <div class="tool-divider" />
      <button class="tool-btn" :class="{ active: ui.showGrid }" title="网格 (G)" @click="ui.showGrid = !ui.showGrid">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
          <rect x="3.5" y="3.5" width="7" height="7" rx="1.4" />
          <rect x="13.5" y="3.5" width="7" height="7" rx="1.4" />
          <rect x="3.5" y="13.5" width="7" height="7" rx="1.4" />
          <rect x="13.5" y="13.5" width="7" height="7" rx="1.4" />
        </svg>
      </button>
      <button class="tool-btn" :class="{ active: ui.showStuds }" title="凸点预览" @click="ui.showStuds = !ui.showStuds">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
          <circle cx="12" cy="12" r="4" />
          <circle cx="12" cy="12" r="8.2" stroke-dasharray="2 3" />
        </svg>
      </button>
      <div class="tool-divider" />
      <button class="tool-btn" title="缩小 (-)" @click="zoomBy(1 / 1.2)">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M5 12h14" /></svg>
      </button>
      <span class="zoom-chip">{{ Math.round(ui.zoom) }}px</span>
      <button class="tool-btn" title="放大 (+)" @click="zoomBy(1.2)">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 5v14M5 12h14" /></svg>
      </button>
      <button class="tool-btn" title="适应窗口 (0)" @click="fit">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round">
          <path d="M4 9V4h5M20 9V4h-5M4 15v5h5M20 15v5h-5" />
        </svg>
      </button>
      <div class="tool-divider" />
      <div class="tool-group">
        <button class="tool-btn" :class="{ active: ui.view === 'render' }" title="立体预览" @click="ui.view = 'render'">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7">
            <path d="M12 3l8 4.5v9L12 21l-8-4.5v-9L12 3z" />
            <path d="M12 12l8-4.5M12 12v9M12 12L4 7.5" />
          </svg>
        </button>
        <button class="tool-btn" :class="{ active: ui.view === 'flat' }" title="平面图" @click="ui.view = 'flat'">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7">
            <rect x="4" y="4" width="7" height="7" rx="1" />
            <rect x="13" y="4" width="7" height="7" rx="1" />
            <rect x="4" y="13" width="7" height="7" rx="1" />
            <rect x="13" y="13" width="7" height="7" rx="1" />
          </svg>
        </button>
      </div>
    </div>

    <div v-if="ui.rendering" class="canvas-badge"><span class="spinner" /> 正在重新排布颗粒…</div>
    <div v-else-if="data.width" class="canvas-badge">
      {{ data.width }} × {{ data.height }} 颗粒
      <span v-if="ui.focusColor !== null" style="color: var(--ink-4)">· 已高亮 {{ focusName }}</span>
    </div>

    <div v-if="data.width" class="canvas-hint">
      <span><kbd>B</kbd>画笔</span>
      <span><kbd>I</kbd>吸管</span>
      <span><kbd>空格</kbd>拖动画布</span>
      <span><kbd>Ctrl</kbd><kbd>Z</kbd>撤销</span>
    </div>

    <div v-if="!ui.booting && !data.width" class="empty-state">
      <div class="card">
        <div class="brand-mark" style="margin: 0 auto; width: 44px; height: 44px; padding: 9px">
          <i /><i /><i /><i />
        </div>
        <h3>还没有作品</h3>
        <p>导入一张照片，或从内置示例开始。<br />软件会把它拆成乐高颗粒，并算出全部零件。</p>
        <div class="row-btns" style="justify-content: center">
          <button class="btn primary" @click="openImage()">导入图片</button>
          <button class="btn" @click="loadDemo()">用示例开始</button>
        </div>
      </div>
    </div>
  </section>
</template>
