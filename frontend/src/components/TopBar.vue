<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { ui, openImage, openProject, saveProject, exportAs, type ExportKind } from '../lib/store'

const menuOpen = ref(false)
const root = ref<HTMLElement | null>(null)

const items: { kind: ExportKind; title: string; desc: string; badge: string }[] = [
  { kind: 'png', title: 'PNG 效果图', desc: '带凸点的高清渲染图', badge: 'PNG' },
  { kind: 'svg-sheet', title: '1:1 施工图', desc: '按真实尺寸打印，铺着拼', badge: 'SVG' },
  { kind: 'svg-steps', title: '分步拼装图', desc: '按颜色分页，逐步照拼', badge: 'SVG' },
  { kind: 'csv', title: '零件清单', desc: '颜色 / 料号 / 数量，方便采购', badge: 'CSV' },
]

function pick(kind: ExportKind) {
  menuOpen.value = false
  void exportAs(kind)
}

function onDocClick(e: MouseEvent) {
  if (!menuOpen.value) return
  if (root.value && !root.value.contains(e.target as Node)) menuOpen.value = false
}

onMounted(() => document.addEventListener('mousedown', onDocClick))
onBeforeUnmount(() => document.removeEventListener('mousedown', onDocClick))
</script>

<template>
  <header class="top-bar">
    <div class="brand">
      <div class="brand-mark"><i /><i /><i /><i /></div>
      <div class="brand-text">
        <h1>像素积木</h1>
        <p>Micro Brick Studio</p>
      </div>
    </div>

    <div class="project-chip">
      <span class="dot" :class="{ dirty: ui.dirty }" />
      <input v-model="ui.projectName" spellcheck="false" />
    </div>

    <div class="top-actions">
      <button class="btn ghost" @click="openProject()">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
          <path d="M4 6.5A1.5 1.5 0 015.5 5h3.6l1.6 2h7.8A1.5 1.5 0 0120 8.5v9A1.5 1.5 0 0118.5 19h-13A1.5 1.5 0 014 17.5v-11z" />
        </svg>
        打开
      </button>
      <button class="btn ghost" @click="saveProject()">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
          <path d="M5 4h11l3 3v13H5z" /><path d="M8 4v6h7V4M8 20v-6h8v6" />
        </svg>
        保存
      </button>
      <button class="btn" @click="openImage()">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.9">
          <rect x="3.5" y="4.5" width="17" height="15" rx="2" />
          <circle cx="9" cy="10" r="1.6" />
          <path d="M4 17l5-4.5 4 3.5 3-2.5 4 3.5" stroke-linecap="round" />
        </svg>
        导入图片
      </button>

      <div ref="root" class="menu-anchor">
        <button class="btn accent" @click="menuOpen = !menuOpen">
          导出
          <span class="caret">▼</span>
        </button>
        <div v-if="menuOpen" class="menu">
          <button v-for="it in items" :key="it.kind" class="menu-item" @click="pick(it.kind)">
            <span class="mi-icon">{{ it.badge }}</span>
            <span class="mi-text">
              <strong>{{ it.title }}</strong>
              <small>{{ it.desc }}</small>
            </span>
          </button>
        </div>
      </div>
    </div>
  </header>
</template>
