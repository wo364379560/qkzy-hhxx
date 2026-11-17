<template>
  <button v-if="notice?.pinned && notice.content" class="group-notice-bar" type="button" @click="expanded = true" aria-label="查看群公告">
    <strong>群公告</strong>
    <div class="notice-window"><div :key="groupID + ':' + notice.content" class="notice-track" :style="{ animationDuration: duration, animationPlayState: visible ? 'running' : 'paused' }">{{ scrollingText }}</div></div>
  </button>
  <div v-if="expanded && notice?.content" class="notice-mask" @click.self="expanded = false">
    <section role="dialog" aria-modal="true" aria-label="群公告" class="notice-dialog">
      <header><strong>群公告</strong><button type="button" @click="expanded = false">关闭</button></header>
      <p>{{ notice.content }}</p>
    </section>
  </div>
</template>
<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from '../../adapter-vue';
import { getGroupNotice, type GroupNotice } from '@/api';
import { GROUP_NOTICE_UPDATED } from '@/features/group/notice';
const props = withDefaults(defineProps<{ groupID?: string; active?: boolean }>(), { active: true });
const notice = ref<GroupNotice>();
const expanded = ref(false);
const pageVisible = ref(!document.hidden);
const visible = computed(() => props.active && pageVisible.value);
// The ticker stays on one line; the detail dialog retains the original line breaks.
const scrollingText = computed(() => (notice.value?.content || '').replace(/[\s\u0085]+/g, ' ').trim());
const duration = computed(() => `${Math.max(16, (notice.value?.content.length || 0) / 4)}s`);
let epoch = 0;
let pendingEpoch: number | undefined;
let timer: ReturnType<typeof setInterval> | undefined;
async function refresh() {
  const id = props.groupID;
  const generation = epoch;
  if (!id || !visible.value || pendingEpoch === generation) return;
  pendingEpoch = generation;
  try {
    const result = await getGroupNotice(id);
    if (generation === epoch && id === props.groupID) notice.value = result;
  } catch { /* Keep the last confirmed notice on temporary failures. */ }
  finally { if (pendingEpoch === generation) pendingEpoch = undefined; }
}
function updated(event: Event) {
  const detail = (event as CustomEvent).detail;
  if (detail?.groupID !== props.groupID) return;
  epoch += 1; notice.value = detail.notice;
}
watch(() => props.groupID, () => { epoch += 1; notice.value = undefined; expanded.value = false; void refresh(); }, { immediate: true });
function syncVisibility() { pageVisible.value = !document.hidden; }
watch(visible, (active) => {
  clearInterval(timer);
  timer = undefined;
  if (active) {
    void refresh();
    timer = setInterval(refresh, 60_000);
  } else {
    expanded.value = false;
  }
}, { immediate: true });
onMounted(() => {
  window.addEventListener(GROUP_NOTICE_UPDATED, updated);
  document.addEventListener('visibilitychange', syncVisibility);
});
onUnmounted(() => {
  epoch += 1;
  clearInterval(timer);
  window.removeEventListener(GROUP_NOTICE_UPDATED, updated);
  document.removeEventListener('visibilitychange', syncVisibility);
});
</script>
<style scoped>
.group-notice-bar { display:flex; align-items:center; gap:16px; flex-shrink:0; width:100%; min-width:0; overflow:hidden; border:0; border-bottom:1px solid #f4e8d5; background:#fff9ee; color:#765024; height:34px; padding:0 16px; text-align:left; cursor:pointer; }
.group-notice-bar strong { flex-shrink:0; font-size:12px; }
.notice-window { display:block; position:relative; overflow:hidden; contain:paint; flex:1; min-width:0; height:20px; max-height:20px; line-height:20px; white-space:nowrap; }
.notice-track { display:block; position:absolute; top:0; left:0; width:max-content; min-width:100%; height:20px; line-height:20px; white-space:nowrap; overflow-wrap:normal; word-break:normal; animation:notice-scroll linear infinite; font-size:13px; }
@keyframes notice-scroll { from { transform:translateX(0); } to { transform:translateX(-100%); } }
@media (prefers-reduced-motion: reduce) { .notice-track { animation:none; } }
.notice-mask { position:fixed; inset:0; z-index:1000; background:#0005; display:grid; place-items:center; }
.notice-dialog { background:white; border-radius:12px; padding:20px; width:min(560px,90vw); max-height:75vh; overflow:auto; }
.notice-dialog header { display:flex; justify-content:space-between; gap:24px; }
.notice-dialog p { white-space:pre-wrap; overflow-wrap:anywhere; line-height:1.7; }
.notice-dialog button { border:0; background:transparent; color:#1677ff; cursor:pointer; }
</style>
