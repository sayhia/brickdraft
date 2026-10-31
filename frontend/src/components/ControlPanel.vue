<script setup lang="ts">
import { computed } from 'vue'
import {
  data,
  settings,
  ui,
  setWidth,
  setTone,
  setPalette,
  setBrickMode,
  toggleDither,
  openImage,
  openDemoGallery,
  randomDemo,
} from '../lib/store'

const presets = [32, 48, 64, 96, 128]

const widthPct = computed(() => ((settings.width - 16) / (192 - 16)) * 100)
const tonePct = (v: number) => ((v + 1) / 2) * 100

const sizeLabel = computed(() => {
  const w = (data.width * 0.8).toFixed(1)
  const h = (data.height * 0.8).toFixed(1)
  return `${w} × ${h} cm`
})

function paletteDots(id: string): string[] {
  const pal = data.palettes.find((p) => p.id === id)
  const colors = pal?.colors ?? []
  if (!colors.length) return []
  const step = Math.max(1, Math.floor(colors.length / 8))
  const out: string[] = []
  for (let i = 0; i < colors.length && out.length < 8; i += step) out.push(colors[i].hex)
  return out
}

function onWidth(e: Event) {
  const v = Number((e.target as HTMLInputElement).value)
  setWidth(v)
}
</script>

<template>
  <aside class="panel">
    <div class="panel-scroll">
      <!-- source -->
      <div class="section" style="padding-top: 16px">
        <div class="section-head">
          <span class="section-title">图源</span>
          <span class="section-note" v-if="data.image">{{ data.image.width }} × {{ data.image.height }} px</span>
        </div>
        <div class="source-card">
          <div class="source-thumb" :style="{ backgroundImage: data.image?.preview ? `url(${data.image.preview})` : 'none' }" />
          <div class="source-meta">
            <strong>{{ data.image?.name || '尚未导入' }}</strong>
            <span v-if="data.image">{{ data.image.path || '内置示例 · ' + data.demos.length + ' 个场景' }}</span>
            <span v-else>支持 PNG / JPG / GIF / BMP / WebP</span>
          </div>
        </div>
        <div class="row-btns">
          <button class="btn" @click="openImage()">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.9">
              <path d="M12 16V4m0 0L7.5 8.5M12 4l4.5 4.5" stroke-linecap="round" />
              <path d="M4 16v2.5A1.5 1.5 0 005.5 20h13a1.5 1.5 0 001.5-1.5V16" stroke-linecap="round" />
            </svg>
            导入图片
          </button>
          <button class="btn ghost" @click="openDemoGallery()">
            示例<span class="count">{{ data.demos.length }}</span>
          </button>
          <button class="btn ghost" title="随机加载一个示例" @click="randomDemo()">随机</button>
        </div>
      </div>

      <!-- grid size -->
      <div class="section">
        <div class="section-head">
          <span class="section-title">成品尺寸</span>
          <span class="section-note">{{ data.width }} × {{ data.height }} 颗粒</span>
        </div>
        <div class="field">
          <label>宽度</label>
          <span class="value">{{ settings.width }} 颗粒</span>
        </div>
        <input
          type="range"
          min="16"
          max="192"
          step="2"
          :value="settings.width"
          :style="{ '--pct': widthPct + '%' }"
          @input="onWidth"
        />
        <div style="display: flex; gap: 5px; margin-top: 8px">
          <button
            v-for="p in presets"
            :key="p"
            class="btn ghost"
            style="flex: 1; height: 26px; padding: 0; font-size: 11px; border-radius: 7px"
            :style="settings.width === p ? 'background:#1d1c18;color:#fff' : ''"
            @click="setWidth(p)"
          >
            {{ p }}
          </button>
        </div>
        <div style="display: flex; justify-content: space-between; margin-top: 10px; font-size: 11px; color: var(--ink-3)">
          <span>实物尺寸</span>
          <span style="font-family: var(--mono); color: var(--ink-2)">{{ sizeLabel }}</span>
        </div>
      </div>

      <!-- brick mode -->
      <div class="section">
        <div class="section-head">
          <span class="section-title">颗粒类型</span>
        </div>
        <div class="mode-list">
          <button
            v-for="m in data.modes"
            :key="m.id"
            class="mode-card"
            :class="{ active: settings.brickMode === m.id }"
            @click="setBrickMode(m.id)"
          >
            <span class="mode-glyph">
              <span v-for="(s, i) in m.sample" :key="i" :style="{ flex: i === 0 ? 2 : 1 }" />
            </span>
            <span class="mode-info">
              <strong>{{ m.name }}</strong>
              <small>{{ m.desc }}</small>
            </span>
          </button>
        </div>
      </div>

      <!-- palette -->
      <div class="section">
        <div class="section-head">
          <span class="section-title">调色板</span>
          <span class="section-note">{{ data.colors.length }} 色在用</span>
        </div>
        <div class="palette-select">
          <button
            v-for="p in data.palettes"
            :key="p.id"
            class="palette-card"
            :class="{ active: settings.paletteId === p.id }"
            @click="setPalette(p.id)"
          >
            <strong>{{ p.name }}</strong>
            <span class="palette-dots">
              <i v-for="(c, i) in paletteDots(p.id)" :key="i" :style="{ background: c }" />
            </span>
          </button>
        </div>
      </div>

      <!-- tone -->
      <div class="section">
        <div class="section-head">
          <span class="section-title">影调</span>
          <span class="section-note">实时预览</span>
        </div>
        <div class="tone-grid">
          <div class="tone-cell">
            <div class="field">
              <label>亮度</label>
              <span class="value">{{ Math.round(settings.brightness * 100) }}</span>
            </div>
            <input
              type="range"
              min="-100"
              max="100"
              step="2"
              :value="settings.brightness * 100"
              :style="{ '--pct': tonePct(settings.brightness) + '%' }"
              @input="setTone('brightness', Number(($event.target as HTMLInputElement).value) / 100)"
            />
          </div>
          <div class="tone-cell">
            <div class="field">
              <label>对比度</label>
              <span class="value">{{ Math.round(settings.contrast * 100) }}</span>
            </div>
            <input
              type="range"
              min="-100"
              max="100"
              step="2"
              :value="settings.contrast * 100"
              :style="{ '--pct': tonePct(settings.contrast) + '%' }"
              @input="setTone('contrast', Number(($event.target as HTMLInputElement).value) / 100)"
            />
          </div>
          <div class="tone-cell">
            <div class="field">
              <label>饱和度</label>
              <span class="value">{{ Math.round(settings.saturation * 100) }}</span>
            </div>
            <input
              type="range"
              min="-100"
              max="100"
              step="2"
              :value="settings.saturation * 100"
              :style="{ '--pct': tonePct(settings.saturation) + '%' }"
              @input="setTone('saturation', Number(($event.target as HTMLInputElement).value) / 100)"
            />
          </div>
        </div>
        <div class="toggle-row" style="margin-top: 6px">
          <span class="label">
            抖动
            <span class="hint">用相邻色点混出更多层次，颗粒感更强</span>
          </span>
          <button class="switch" :class="{ on: settings.dither }" @click="toggleDither()" />
        </div>
      </div>
    </div>
  </aside>
</template>
