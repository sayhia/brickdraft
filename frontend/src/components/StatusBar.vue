<script setup lang="ts">
import { computed } from 'vue'
import { data, ui, settings } from '../lib/store'

const modeName = computed(() => data.modes.find((m) => m.id === settings.brickMode)?.name ?? '')

const dims = computed(() => {
  if (!data.width) return '—'
  return `${data.width} × ${data.height} 颗粒`
})

const real = computed(() => {
  if (!data.width) return '—'
  return `${(data.width * 0.8).toFixed(1)} × ${(data.height * 0.8).toFixed(1)} cm`
})

const bricks = computed(() => (data.stats ? data.stats.bricks.toLocaleString('zh-CN') : '—'))
const colors = computed(() => (data.stats ? data.stats.colors : '—'))
const weight = computed(() => (data.stats ? `${data.stats.weightG.toFixed(0)} g` : '—'))

const statusText = computed(() => {
  if (ui.booting) return '初始化中'
  if (ui.rendering) return '正在计算配色'
  if (!data.width) return '等待导入图片'
  return '就绪'
})
</script>

<template>
  <footer class="status-bar">
    <span class="status-item">
      <span class="status-dot" :style="ui.rendering ? 'background:#e8b22b' : ''" />
      {{ statusText }}
    </span>
    <span class="status-item">画布 <b>{{ dims }}</b></span>
    <span class="status-item">实物 <b>{{ real }}</b></span>
    <span class="status-item">零件 <b>{{ bricks }}</b></span>
    <span class="status-item">颜色 <b>{{ colors }}</b></span>
    <span class="status-item">估重 <b>{{ weight }}</b></span>
    <span class="status-spacer" />
    <span class="status-item">{{ modeName }}</span>
    <span class="status-item">缩放 <b>{{ Math.round(ui.zoom) }}px</b></span>
    <span class="status-item">网格 <b>{{ ui.showGrid ? '开' : '关' }}</b></span>
  </footer>
</template>
