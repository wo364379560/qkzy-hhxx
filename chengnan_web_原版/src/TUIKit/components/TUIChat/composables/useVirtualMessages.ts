import { computed, nextTick, onMounted, onUnmounted, ref, shallowRef, watch, type Ref } from '../../../adapter-vue';
import { messageIndexAt, messageOffsets, messageWindow } from '../utils/messageWindow';

// Variable-height desktop window. The data stays available for forwarding and
// jumps; only the rows around the viewport have Vue components/DOM nodes.
export function useVirtualMessages<T extends { ID: string }>(
  messages: Ref<T[]>, root: Ref<HTMLElement | undefined>,
  rows: Ref<HTMLElement[] | null | undefined>, enabled: boolean,
) {
  const topSpacer = ref<HTMLElement>();
  const top = ref(0);
  const viewport = ref(600);
  const listTop = ref(0);
  const offsets = shallowRef<number[]>([0]);
  let ids: string[] = [];
  const heights = new Map<string, number>();
  let observer: ResizeObserver | undefined;
  let frame = 0;
  let disposed = false;
  let revision = 0;
  let width = 0;
  let followBottom = false;
  let pinnedAnchor: ReturnType<typeof captureAnchor> | undefined;
  const range = computed(() => enabled
    ? messageWindow(offsets.value, top.value - listTop.value, viewport.value)
    : { start: 0, end: messages.value.length, top: 0, bottom: 0 });

  function syncScroll() {
    const element = root.value;
    if (!enabled || !element) return;
    top.value = element.scrollTop;
    viewport.value = element.clientHeight;
    if (topSpacer.value) {
      listTop.value = topSpacer.value.getBoundingClientRect().top
        - element.getBoundingClientRect().top - element.clientTop + element.scrollTop;
    }
  }

  function onUserScroll() { followBottom = false; pinnedAnchor = undefined; }

  function onScroll() {
    const element = root.value;
    if (!element) return;
    if (element.scrollHeight - element.clientHeight - element.scrollTop < 4) followBottom = true;
    syncScroll();
  }

  function captureAnchor() {
    const element = root.value;
    const position = (element?.scrollTop || 0) - listTop.value;
    const index = messageIndexAt(offsets.value, Math.max(0, position));
    return {
      id: ids[index], offset: position - offsets.value[index],
      // Use the previous layout: resized DOM may already have grown before
      // ResizeObserver runs, while the old offsets still describe the anchor.
      bottom: followBottom || Boolean(element && offsets.value[offsets.value.length - 1] + listTop.value
        - element.clientHeight - element.scrollTop < 4),
    };
  }

  function restoreAnchor(anchor: ReturnType<typeof captureAnchor>, forcePosition = false) {
    if (forcePosition) { followBottom = false; anchor = { ...anchor, bottom: false }; }
    pinnedAnchor = anchor.id ? anchor : undefined;
    const ticket = ++revision;
    void nextTick(() => {
      if (disposed || ticket !== revision || !root.value) return;
      syncScroll();
      const index = ids.indexOf(anchor.id);
      if (anchor.bottom && !forcePosition) root.value.scrollTop = root.value.scrollHeight;
      else if (index >= 0) root.value.scrollTop = listTop.value + offsets.value[index] + anchor.offset;
      syncScroll();
    });
  }

  watch(() => messages.value.map(message => message.ID), (nextIDs) => {
    if (!enabled) return;
    const anchor = pinnedAnchor && ids.includes(pinnedAnchor.id) ? pinnedAnchor : captureAnchor();
    const prepended = Boolean(ids.length && nextIDs.indexOf(ids[0]) > 0);
    ids = nextIDs;
    const alive = new Set(ids);
    for (const id of heights.keys()) if (!alive.has(id)) heights.delete(id);
    offsets.value = messageOffsets(ids, heights);
    restoreAnchor(anchor, prepended);
  }, { immediate: true, flush: 'sync' });

  function measure() {
    frame = 0;
    if (disposed || !enabled || !root.value) return;
    const anchor = pinnedAnchor && ids.includes(pinnedAnchor.id) ? pinnedAnchor : captureAnchor();
    let changed = false;
    if (width !== root.value.clientWidth) {
      width = root.value.clientWidth;
      heights.clear();
      changed = true;
    }
    for (const row of rows.value || []) {
      const id = row.dataset.messageId;
      const height = row.getBoundingClientRect().height;
      if (id && height > 0 && Math.abs((heights.get(id) || 0) - height) > 0.5) {
        heights.set(id, height);
        changed = true;
      }
    }
    if (changed) {
      offsets.value = messageOffsets(ids, heights);
      restoreAnchor(anchor);
    }
    syncScroll();
  }

  function scheduleMeasure() {
    if (!frame && !disposed) frame = requestAnimationFrame(measure);
  }

  watch(() => [...(rows.value || [])], () => {
    if (!enabled) return;
    observer?.disconnect();
    if (root.value) observer?.observe(root.value);
    for (const row of rows.value || []) observer?.observe(row);
    scheduleMeasure();
  }, { flush: 'post' });

  onMounted(() => {
    if (!enabled) return;
    if (typeof ResizeObserver !== 'undefined') observer = new ResizeObserver(scheduleMeasure);
    root.value?.addEventListener('scroll', onScroll, { passive: true });
    for (const event of ['wheel', 'pointerdown', 'touchstart', 'keydown']) {
      root.value?.addEventListener(event, onUserScroll, { passive: true });
    }
    if (root.value) observer?.observe(root.value);
    for (const row of rows.value || []) observer?.observe(row);
    syncScroll();
    scheduleMeasure();
  });
  onUnmounted(() => {
    disposed = true;
    revision += 1;
    observer?.disconnect();
    root.value?.removeEventListener('scroll', onScroll);
    for (const event of ['wheel', 'pointerdown', 'touchstart', 'keydown']) {
      root.value?.removeEventListener(event, onUserScroll);
    }
    cancelAnimationFrame(frame);
    heights.clear();
  });

  function reset() {
    pinnedAnchor = undefined;
    followBottom = false;
    revision += 1;
    heights.clear();
    ids = [];
    offsets.value = [0];
    top.value = 0;
    if (root.value) root.value.scrollTop = 0;
  }
  function scrollToMessage(id: string) {
    pinnedAnchor = undefined;
    followBottom = false;
    if (!enabled || !root.value) return;
    revision += 1;
    const index = ids.indexOf(id);
    if (index < 0) return;
    root.value.scrollTop = Math.max(0, listTop.value + offsets.value[index] - root.value.clientHeight / 2);
    syncScroll();
  }
  function scrollToBottom() {
    pinnedAnchor = undefined;
    followBottom = true;
    if (!enabled || !root.value) return;
    revision += 1;
    root.value.scrollTop = root.value.scrollHeight;
    syncScroll();
  }

  return { range, topSpacer, reset, scrollToMessage, scrollToBottom, syncScroll };
}
