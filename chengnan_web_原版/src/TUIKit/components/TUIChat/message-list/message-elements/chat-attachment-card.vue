<script lang="ts" setup>
import { computed, ref, withDefaults } from "../../../../adapter-vue";
import type { ChatAttachmentInfo } from "../../utils/chatAttachmentMessage";
import { formatChatAttachmentSize } from "../../utils/chatAttachmentMessage";
import ChatAttachmentPreviewDialog from "./chat-attachment-preview-dialog.vue";

const props = withDefaults(
  defineProps<{
    attachment: ChatAttachmentInfo;
  }>(),
  {}
);

const previewVisible = ref(false);
const sizeText = computed(() =>
  formatChatAttachmentSize(props.attachment.size)
);
const isImage = computed(() => {
  const type = String(props.attachment.contentType || "").toLowerCase();
  const name = props.attachment.name || "";
  return type.startsWith("image/") || /\.(jpe?g|png|gif|webp)$/i.test(name);
});
const typeLabel = computed(() => {
  if (props.attachment.isVideo) {
    return "视频";
  }
  const name = props.attachment.name || "";
  const ext = name.includes(".") ? name.split(".").pop() || "" : "";
  return ext ? ext.slice(0, 4).toUpperCase() : "FILE";
});
const actionText = computed(() =>
  props.attachment.isVideo
    ? "点击预览，后台下载"
    : isImage.value
    ? "点击预览或下载"
    : "点击下载文件"
);

function openPreview() {
  previewVisible.value = true;
}

function closePreview() {
  previewVisible.value = false;
}
</script>

<template>
  <div>
    <button class="chat-attachment-card" type="button" @click="openPreview">
      <span class="chat-attachment-card__icon">
        {{ typeLabel }}
      </span>
      <span class="chat-attachment-card__content">
        <span class="chat-attachment-card__name">{{ attachment.name }}</span>
        <span class="chat-attachment-card__meta">
          {{ actionText }}
          <template v-if="sizeText"> · {{ sizeText }}</template>
        </span>
      </span>
      <span class="chat-attachment-card__arrow">›</span>
    </button>
    <ChatAttachmentPreviewDialog
      :visible="previewVisible"
      :attachment="attachment"
      @close="closePreview"
    />
  </div>
</template>

<style lang="scss" scoped>
.chat-attachment-card {
  width: 260px;
  max-width: min(260px, calc(100vw - 96px));
  min-height: 68px;
  border: 1px solid rgba(148, 163, 184, 0.28);
  border-radius: 8px;
  padding: 10px;
  display: grid;
  grid-template-columns: 42px minmax(0, 1fr) 16px;
  align-items: center;
  gap: 10px;
  background: #fff;
  color: #111827;
  text-align: left;
  cursor: pointer;
}

.chat-attachment-card__icon {
  width: 42px;
  height: 42px;
  border-radius: 6px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: #2563eb;
  color: #fff;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0;
}

.chat-attachment-card__content {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.chat-attachment-card__name {
  overflow: hidden;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  color: #111827;
  font-size: 14px;
  font-weight: 600;
  line-height: 1.25;
  word-break: break-word;
}

.chat-attachment-card__meta {
  overflow: hidden;
  color: #6b7280;
  font-size: 12px;
  line-height: 1.25;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.chat-attachment-card__arrow {
  color: #9ca3af;
  font-size: 20px;
  line-height: 1;
}
</style>
