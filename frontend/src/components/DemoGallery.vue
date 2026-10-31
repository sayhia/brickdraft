<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { data, ui, thumbs, loadDemo, closeDemoGallery, ensureThumbs, randomDemo } from '../lib/store'

/**
 * The built-in library holds 1000 pictures, so the gallery pages instead of
 * rendering everything, and asks Go for thumbnails only for the page that is
 * open. Everything else (search, category, mood) filters the lightweight
 * catalogue that arrives with start-up.
 */

const PAGE = 48

const cats = computed(() => {
  const seen = new Map<number, string>()
  for (const d of data.demos) {
    if (!seen.has(d.cat) && d.tags?.[0]) seen.set(d.cat, d.tags[0])
  }
  return [...seen.entries()].map(([id, name]) => ({ id, name }))
})

const moods = computed(() => {
  const seen = new Set<string>()
  for (const d of data.demos) seen.add(d.moodName)
  return [...seen]
})

// the photographs are the honest half of the library, so the header counts them
const photoN = computed(() => data.demos.filter((d) => d.photo).length)

const filtered = computed(() => {
  const q = ui.demoQuery.trim().toLowerCase()
  const cat = ui.demoCat
  if (!q && cat < 0) return data.demos
  return data.demos.filter((d) => {
    if (cat >= 0 && d.cat !== cat) return false
    if (!q) return true
    return (
      d.name.toLowerCase().includes(q) ||
      d.moodName.toLowerCase().includes(q) ||
      (d.tags ?? []).some((t) => t.toLowerCase().includes(q)) ||
      String(d.id) === q
    )
  })
})

const pages = computed(() => Math.max(1, Math.ceil(filtered.value.length / PAGE)))
const page = ref(0)

const visible = computed(() => {
  const start = page.value * PAGE
  return filtered.value.slice(start, start + PAGE)
})

const scroller = ref<HTMLElement | null>(null)

function jumpTo(p: number) {
  page.value = Math.min(pages.value - 1, Math.max(0, p))
  void nextTick(() => scroller.value?.scrollTo({ top: 0 }))
}

// filters change the result set, so the old page number is meaningless
watch([() => ui.demoQuery, () => ui.demoCat], () => {
  page.value = 0
})

watch(
  visible,
  (list) => {
    const ids = list.map((d) => d.id)
    // two batches: the first rows paint immediately, the rest follows
    void ensureThumbs(ids.slice(0, 24))
    if (ids.length > 24) window.setTimeout(() => void ensureThumbs(ids.slice(24)), 60)
  },
  { immediate: true },
)

// the modal mounts with the app, so a first open must paint too
watch(
  () => ui.demoOpen,
  (open) => {
    if (open) void nextTick(() => scroller.value?.scrollTo({ top: 0 }))
  },
)

function pick(id: number) {
  void loadDemo(id)
}

function onKey(e: KeyboardEvent) {
  if (!ui.demoOpen) return
  if (e.key === 'Escape') closeDemoGallery()
  if (e.key === 'ArrowRight') jumpTo(page.value + 1)
  if (e.key === 'ArrowLeft') jumpTo(page.value - 1)
}

onMounted(() => window.addEventListener('keydown', onKey))
onUnmounted(() => window.removeEventListener('keydown', onKey))

const pageWindow = computed(() => {
  // five page buttons around the current one, never more than exists
  const last = pages.value - 1
  const from = Math.max(0, Math.min(page.value - 2, last - 4))
  const to = Math.min(last, from + 4)
  const out: number[] = []
  for (let i = from; i <= to; i++) out.push(i)
  return out
})
</script>

<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="ui.demoOpen" class="modal-backdrop" @click.self="closeDemoGallery()">
        <div class="modal demo-modal">
          <header class="modal-head">
            <div>
              <h3>示例库</h3>
              <p>
                {{ data.demos.length }} 个内置示例 · {{ photoN }} 张实拍照片 ·
                {{ filtered.length }} 个匹配
              </p>
            </div>
            <div class="head-tools">
              <button class="btn ghost sm" @click="randomDemo()">随机一个</button>
              <button class="icon-btn" title="关闭 (Esc)" @click="closeDemoGallery()">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M6 6l12 12M18 6L6 18" stroke-linecap="round" />
                </svg>
              </button>
            </div>
          </header>

          <div class="demo-tools">
            <label class="search">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="11" cy="11" r="6.5" />
                <path d="M16 16l4 4" stroke-linecap="round" />
              </svg>
              <input
                v-model="ui.demoQuery"
                placeholder="搜索名称、类别，或输入「实拍」/编号"
                spellcheck="false"
              />
            </label>
            <div class="chips">
              <button
                class="chip"
                :class="{ on: ui.demoCat < 0 }"
                @click="ui.demoCat = -1"
              >
                全部
              </button>
              <button
                v-for="c in cats"
                :key="c.id"
                class="chip"
                :class="{ on: ui.demoCat === c.id }"
                @click="ui.demoCat = c.id"
              >
                {{ c.name }}
              </button>
            </div>
          </div>

          <div ref="scroller" class="demo-grid">
            <button
              v-for="d in visible"
              :key="d.id"
              class="demo-card"
              :class="{ active: ui.sceneId === d.id }"
              @click="pick(d.id)"
            >
              <span class="demo-thumb">
                <img v-if="thumbs[d.id]" :src="thumbs[d.id]" :alt="d.name" />
                <i v-else class="ph" />
                <em v-if="d.photo" class="demo-badge">实拍</em>
              </span>
              <span class="demo-text">
                <strong>{{ d.name }}</strong>
                <em>{{ d.moodName }} · {{ String(d.id).padStart(3, '0') }}</em>
              </span>
            </button>
          </div>

          <div v-if="!filtered.length" class="demo-empty">
            没有匹配的示例，换个关键词或点「全部」。
          </div>

          <footer class="modal-foot">
            <div class="pager">
              <button class="pg" :disabled="page === 0" @click="jumpTo(page - 1)">上一页</button>
              <button
                v-for="p in pageWindow"
                :key="p"
                class="pg"
                :class="{ on: p === page }"
                @click="jumpTo(p)"
              >
                {{ p + 1 }}
              </button>
              <button class="pg" :disabled="page >= pages - 1" @click="jumpTo(page + 1)">
                下一页
              </button>
              <span class="pg-note">{{ page + 1 }} / {{ pages }}</span>
            </div>
            <span class="foot-note">
              {{ moods.join(' · ') }} · 场景随颗粒宽度重绘，任何尺寸都不会混色
            </span>
          </footer>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>
