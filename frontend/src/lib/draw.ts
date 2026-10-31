import type { Brick } from '../../bindings/xiangsu'

export interface Camera {
  cell: number // css px per stud
  ox: number
  oy: number
}

export interface Scene {
  width: number
  height: number
  cells: Int16Array
  colors: string[]
  bricks: Brick[]
  studs: boolean
  view: 'render' | 'flat'
  focus: number | null
}

const BACKDROP = '#f4f2ec'

function hexToRgb(hex: string): [number, number, number] {
  const h = hex.replace('#', '')
  const s = h.length === 3 ? h.split('').map((c) => c + c).join('') : h
  const n = parseInt(s, 16)
  return [(n >> 16) & 255, (n >> 8) & 255, n & 255]
}

function rgbToHex(r: number, g: number, b: number): string {
  const c = (v: number) => Math.max(0, Math.min(255, Math.round(v))).toString(16).padStart(2, '0')
  return '#' + c(r) + c(g) + c(b)
}

export function mix(hex: string, target: string, t: number): string {
  const a = hexToRgb(hex)
  const b = hexToRgb(target)
  return rgbToHex(a[0] + (b[0] - a[0]) * t, a[1] + (b[1] - a[1]) * t, a[2] + (b[2] - a[2]) * t)
}

export function shade(hex: string, amt: number): string {
  if (amt >= 0) return mix(hex, '#ffffff', amt)
  return mix(hex, '#101010', -amt)
}

/**
 * MosaicBaker pre-renders the whole mosaic into an offscreen canvas at the
 * current zoom level. Panning and hovering then only blit the cached bitmap,
 * which keeps 300x300 grids smooth.
 */
export class MosaicBaker {
  private cv = document.createElement('canvas')
  private key = ''
  cellPx = 12

  get canvas(): HTMLCanvasElement {
    return this.cv
  }

  ensure(scene: Scene, targetCell: number, rev: number) {
    const cellPx = Math.max(6, Math.min(48, Math.round(targetCell)))
    const key = [
      rev,
      scene.width,
      scene.height,
      scene.view,
      scene.studs ? 1 : 0,
      scene.focus ?? 'x',
      cellPx,
      scene.colors.join(','),
    ].join('|')

    if (key === this.key) return this.cv
    this.key = key
    this.cellPx = cellPx

    const w = Math.max(1, scene.width * cellPx)
    const h = Math.max(1, scene.height * cellPx)
    this.cv.width = w
    this.cv.height = h
    const ctx = this.cv.getContext('2d')
    if (!ctx) return this.cv
    ctx.clearRect(0, 0, w, h)
    ctx.imageSmoothingEnabled = false

    const colorOf = (idx: number): string => {
      const base = scene.colors[idx] ?? '#CCCCCC'
      if (scene.focus !== null && idx !== scene.focus) return mix(base, BACKDROP, 0.8)
      return base
    }

    if (scene.view === 'flat') {
      for (let y = 0; y < scene.height; y++) {
        let x = 0
        while (x < scene.width) {
          const idx = scene.cells[y * scene.width + x]
          let run = 1
          while (x + run < scene.width && scene.cells[y * scene.width + x + run] === idx) run++
          if (idx >= 0) {
            ctx.fillStyle = colorOf(idx)
            ctx.fillRect(x * cellPx, y * cellPx, run * cellPx, cellPx)
          }
          x += run
        }
      }
      return this.cv
    }

    const studR = cellPx * 0.275
    const drawStuds = scene.studs && cellPx >= 8

    for (const b of scene.bricks) {
      if (b.color < 0) continue
      const px = b.x * cellPx
      const py = b.y * cellPx
      const pw = b.w * cellPx
      const ph = b.h * cellPx
      const base = colorOf(b.color)

      if (b.kind === 'round') {
        const cx = px + pw / 2
        const cy = py + ph / 2
        const r = pw / 2 - cellPx * 0.04
        ctx.beginPath()
        ctx.arc(cx, cy, r, 0, Math.PI * 2)
        ctx.fillStyle = shade(base, -0.28)
        ctx.fill()
        ctx.beginPath()
        ctx.arc(cx, cy, r * 0.84, 0, Math.PI * 2)
        ctx.fillStyle = base
        ctx.fill()
        ctx.beginPath()
        ctx.arc(cx - r * 0.16, cy - r * 0.18, r * 0.3, 0, Math.PI * 2)
        ctx.fillStyle = shade(base, 0.18)
        ctx.fill()
        continue
      }

      // body
      ctx.fillStyle = base
      ctx.fillRect(px, py, pw, ph)

      // subtle bevel to read as a physical brick
      if (b.kind === 'brick') {
        ctx.fillStyle = 'rgba(0,0,0,0.14)'
        ctx.fillRect(px, py + ph - Math.max(2, cellPx * 0.16), pw, Math.max(2, cellPx * 0.16))
        ctx.fillStyle = 'rgba(255,255,255,0.16)'
        ctx.fillRect(px, py, pw, Math.max(1, cellPx * 0.07))
      } else if (b.kind === 'tile') {
        ctx.fillStyle = 'rgba(255,255,255,0.22)'
        ctx.fillRect(px, py, pw, Math.max(1, cellPx * 0.06))
      }

      if (drawStuds && b.kind !== 'tile') {
        for (let sy = 0; sy < b.h; sy++) {
          for (let sx = 0; sx < b.w; sx++) {
            const cx = px + sx * cellPx + cellPx / 2
            const cy = py + sy * cellPx + cellPx / 2 + studR * 0.14
            ctx.beginPath()
            ctx.arc(cx, cy, studR, 0, Math.PI * 2)
            ctx.fillStyle = shade(base, -0.13)
            ctx.fill()
            ctx.beginPath()
            ctx.arc(cx, cy - studR * 0.08, studR * 0.85, 0, Math.PI * 2)
            ctx.fillStyle = shade(base, 0.07)
            ctx.fill()
            ctx.beginPath()
            ctx.arc(cx - studR * 0.22, cy - studR * 0.3, studR * 0.32, 0, Math.PI * 2)
            ctx.fillStyle = shade(base, 0.22)
            ctx.fill()
          }
        }
      }

      // seam
      ctx.strokeStyle = 'rgba(18,16,12,0.15)'
      ctx.lineWidth = Math.max(0.5, cellPx * 0.03)
      ctx.strokeRect(px + 0.25, py + 0.25, pw - 0.5, ph - 0.5)
    }

    return this.cv
  }
}

export interface HoverInfo {
  x: number
  y: number
}

export function drawScene(
  canvas: HTMLCanvasElement,
  scene: Scene,
  cam: Camera,
  baker: MosaicBaker,
  opts: { grid: boolean; hover: HoverInfo | null; dpr: number },
) {
  const ctx = canvas.getContext('2d')
  if (!ctx) return
  const w = canvas.width / opts.dpr
  const h = canvas.height / opts.dpr
  ctx.setTransform(opts.dpr, 0, 0, opts.dpr, 0, 0)
  ctx.clearRect(0, 0, w, h)

  const gw = scene.width * cam.cell
  const gh = scene.height * cam.cell

  // board. The drop shadow is drawn as a few concentric strokes instead of
  // ctx.shadowBlur: a real blur is re-rasterised on every frame and is heavy
  // enough on software rendering to take the renderer down with it.
  ctx.fillStyle = '#ffffff'
  ctx.beginPath()
  ctx.roundRect(cam.ox - 1, cam.oy - 1, gw + 2, gh + 2, Math.min(8, cam.cell * 0.3))
  ctx.fill()

  ctx.save()
  for (let i = 3; i >= 1; i--) {
    ctx.strokeStyle = `rgba(30,27,20,${0.035 * (4 - i)})`
    ctx.lineWidth = i * 2
    ctx.beginPath()
    ctx.roundRect(cam.ox - 1 - i, cam.oy - 1 - i + 3, gw + 2 + i * 2, gh + 2 + i * 2, 10)
    ctx.stroke()
  }
  ctx.restore()

  ctx.imageSmoothingEnabled = true
  ctx.imageSmoothingQuality = 'high'
  ctx.drawImage(baker.canvas, cam.ox, cam.oy, gw, gh)

  // grid overlay
  if (opts.grid && cam.cell >= 3) {
    const minor = cam.cell >= 7
    ctx.save()
    ctx.beginPath()
    if (minor) {
      for (let x = 1; x < scene.width; x++) {
        if (x % 8 === 0) continue
        const px = Math.round(cam.ox + x * cam.cell) + 0.5
        ctx.moveTo(px, cam.oy)
        ctx.lineTo(px, cam.oy + gh)
      }
      for (let y = 1; y < scene.height; y++) {
        if (y % 8 === 0) continue
        const py = Math.round(cam.oy + y * cam.cell) + 0.5
        ctx.moveTo(cam.ox, py)
        ctx.lineTo(cam.ox + gw, py)
      }
      ctx.strokeStyle = 'rgba(20,18,14,0.07)'
      ctx.lineWidth = 1
      ctx.stroke()
      ctx.beginPath()
    }
    for (let x = 0; x <= scene.width; x += 8) {
      const px = Math.round(cam.ox + x * cam.cell) + 0.5
      ctx.moveTo(px, cam.oy)
      ctx.lineTo(px, cam.oy + gh)
    }
    for (let y = 0; y <= scene.height; y += 8) {
      const py = Math.round(cam.oy + y * cam.cell) + 0.5
      ctx.moveTo(cam.ox, py)
      ctx.lineTo(cam.ox + gw, py)
    }
    ctx.strokeStyle = 'rgba(20,18,14,0.22)'
    ctx.lineWidth = 1
    ctx.stroke()
    ctx.restore()
  }

  // hovered cell
  if (opts.hover && cam.cell >= 3) {
    const { x, y } = opts.hover
    const px = cam.ox + x * cam.cell
    const py = cam.oy + y * cam.cell
    ctx.save()
    ctx.strokeStyle = 'rgba(255,255,255,0.95)'
    ctx.lineWidth = 2.5
    ctx.strokeRect(px + 0.5, py + 0.5, cam.cell - 1, cam.cell - 1)
    ctx.strokeStyle = 'rgba(20,18,14,0.85)'
    ctx.lineWidth = 1.2
    ctx.strokeRect(px + 0.5, py + 0.5, cam.cell - 1, cam.cell - 1)
    ctx.restore()
  }
}
