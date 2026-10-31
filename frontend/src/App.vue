<script setup lang="ts">
import { onMounted } from 'vue'
import TopBar from './components/TopBar.vue'
import ControlPanel from './components/ControlPanel.vue'
import StudioCanvas from './components/StudioCanvas.vue'
import InspectorPanel from './components/InspectorPanel.vue'
import StatusBar from './components/StatusBar.vue'
import DemoGallery from './components/DemoGallery.vue'
import { boot, toasts, dismissToast } from './lib/store'

onMounted(() => {
  void boot()
})
</script>

<template>
  <div class="app-shell">
    <TopBar />
    <main class="workspace">
      <ControlPanel />
      <StudioCanvas />
      <InspectorPanel />
    </main>
    <StatusBar />
    <DemoGallery />
    <div class="toast-stack">
      <div v-for="t in toasts" :key="t.id" class="toast" :class="t.kind" @click="dismissToast(t.id)">
        <span>{{ t.text }}</span>
        <button
          v-if="t.action"
          class="toast-action"
          @click.stop="
            t.action.run();
            dismissToast(t.id)
          "
        >
          {{ t.action.label }}
        </button>
      </div>
    </div>
  </div>
</template>
