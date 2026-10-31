<script setup lang="ts">
import { computed } from 'vue'
import { data, ui, setBrush, toggleFocus } from '../lib/store'

const maxCells = computed(() => Math.max(1, ...data.colors.map((c) => c.cells)))

const totalGrams = computed(() => data.stats?.weightG ?? 0)

const summary = computed(() =>
  data.colors.length ? `${data.stats?.bricks ?? 0} 块 / ${data.colors.length} 色` : '—',
)

function pct(cells: number) {
  return Math.max(6, (cells / maxCells.value) * 100) + '%'
}
</script>

<template>
  <aside class="panel">
    <div class="tabs">
      <button class="tab" :class="{ active: ui.tab === 'colors' }" @click="ui.tab = 'colors'">配色</button>
      <button class="tab" :class="{ active: ui.tab === 'parts' }" @click="ui.tab = 'parts'">零件清单</button>
      <span style="margin-left: auto; align-self: center; font-size: 11px; color: var(--ink-4)">{{ summary }}</span>
    </div>

    <template v-if="ui.tab === 'colors'">
      <div v-if="!data.colors.length" class="panel-empty">还没有配色信息</div>
      <div v-else class="color-list">
        <button
          v-for="c in data.colors"
          :key="c.index"
          class="color-row"
          :class="{ active: ui.focusColor === c.index }"
          :title="ui.focusColor === c.index ? '取消高亮' : '高亮该颜色'"
          @click="toggleFocus(c.index)"
          @dblclick="setBrush(c.index)"
        >
          <span class="color-swatch" :style="{ background: c.hex }" />
          <span class="color-name">
            {{ c.name }}
            <span class="usage-bar"><i :style="{ width: pct(c.cells), background: c.hex }" /></span>
          </span>
          <span class="color-hex">{{ c.hex }}</span>
          <span class="color-count">{{ c.cells }}</span>
        </button>
      </div>
      <div v-if="data.colors.length" class="panel-footer">
        <span>单击高亮 · 双击取为画笔色</span>
        <button class="btn ghost" style="height: 24px; padding: 0 8px; font-size: 11px" @click="ui.focusColor = null">
          清除高亮
        </button>
      </div>
    </template>

    <template v-else>
      <div v-if="!data.parts.length" class="panel-empty">还没有零件</div>
      <div v-else class="bom-wrap">
        <table class="bom-table">
          <thead>
            <tr>
              <th>颜色</th>
              <th>零件</th>
              <th style="text-align: right">数量</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="p in data.parts" :key="p.color + '-' + p.partNo">
              <td>
                <span class="bom-color">
                  <i :style="{ background: p.colorHex }" />
                  {{ p.colorName }}
                </span>
              </td>
              <td>
                <span class="bom-part">
                  {{ p.label }}
                  <small>{{ p.partNo }}</small>
                </span>
              </td>
              <td class="bom-num">{{ p.count }}</td>
            </tr>
          </tbody>
          <tfoot>
            <tr>
              <td colspan="2" style="border-top: 1px solid var(--line); font-weight: 600">合计</td>
              <td class="bom-num" style="border-top: 1px solid var(--line); font-weight: 700">
                {{ data.stats?.bricks ?? 0 }}
              </td>
            </tr>
            <tr>
              <td colspan="3" style="color: var(--ink-4); font-size: 10.5px">
                料号参考 LEGO 官方编号 · 预估重量 {{ totalGrams.toFixed(1) }} g
              </td>
            </tr>
          </tfoot>
        </table>
      </div>
    </template>
  </aside>
</template>

<style scoped>
.panel-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 8px 14px 12px;
  border-top: 1px solid var(--line);
  font-size: 10.5px;
  color: var(--ink-4);
}
</style>
