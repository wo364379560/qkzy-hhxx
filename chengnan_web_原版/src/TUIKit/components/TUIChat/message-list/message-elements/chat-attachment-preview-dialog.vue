<script lang="ts" setup>
import {
  computed,
  onBeforeUnmount,
  ref,
  watch,
  withDefaults,
} from "../../../../adapter-vue";
import { getChatAttachmentAccessUrl } from "@/api";
import type { ChatAttachmentInfo } from "../../utils/chatAttachmentMessage";
import { formatChatAttachmentSize } from "../../utils/chatAttachmentMessage";
import { Toast, TOAST_TYPE } from "../../../common/Toast/index";
import Icon from "../../../common/Icon.vue";
import closeSVG from "../../../../assets/icon/icon-close.svg";
import downloadSVG from "../../../../assets/icon/download.svg";

const props = withDefaults(
  defineProps<{
    visible: boolean;
    attachment: ChatAttachmentInfo | null;
  }>(),
  {
    visible: false,
    attachment: null,
  }
);

const emit = defineEmits<{
  (e: "close"): void;
}>();

const accessUrl = ref("");
const blobUrl = ref("");
const loading = ref(false);
const downloading = ref(false);
const downloadProgress = ref(0);
const errorMessage = ref("");
const downloadedBlob = ref<Blob | null>(null);
let activeBlobUrl = "";
let requestVersion = 0;

const fileName = computed(() => props.attachment?.name || "文件");
const sizeText = computed(() =>
  formatChatAttachmentSize(Number(props.attachment?.size || 0))
);
const contentType = computed(() =>
  String(props.attachment?.contentType || "").toLowerCase()
);
const isVideo = computed(
  () => !!props.attachment?.isVideo || contentType.value.startsWith("video/")
);
const isImage = computed(
  () =>
    contentType.value.startsWith("image/") ||
    /\.(jpe?g|png|gif|webp)$/i.test(fileName.value)
);
const hasDownloadedBlob = computed(() => !!downloadedBlob.value);
const canSave = computed(
  () => hasDownloadedBlob.value || (isVideo.value && !!accessUrl.value)
);
const videoSrc = computed(() => {
  if (!isVideo.value) {
    return "";
  }
  return blobUrl.value || accessUrl.value;
});

watch(
  () => [props.visible, props.attachment?.fileID],
  () => {
    if (props.visible && props.attachment) {
      void openAttachment();
    } else {
      resetState();
    }
  },
  { immediate: true }
);

onBeforeUnmount(() => {
  revokeBlobUrl();
});

async function openAttachment() {
  const attachment = props.attachment;
  if (!attachment) {
    return;
  }
  const version = ++requestVersion;
  resetState(false);
  loading.value = true;
  try {
    const access = await getChatAttachmentAccessUrl(attachment.fileID);
    if (version !== requestVersion) {
      return;
    }
    accessUrl.value = access.url;
    loading.value = false;
    // 视频直接使用带签名的 COS URL 流式播放。不要再并行 fetch 整个视频，
    // 否则会重复下载百兆文件；当 COS 未开放 GET CORS 时，fetch 还会先
    // 报 "Load failed"，但 video 标签本身其实可以正常播放。
    if (!isVideo.value) {
      void downloadForPreview(access.url, version);
    }
  } catch (error) {
    if (version !== requestVersion) {
      return;
    }
    loading.value = false;
    errorMessage.value =
      error instanceof Error && error.message
        ? error.message
        : "文件加载失败，请稍后重试";
  }
}

async function downloadForPreview(url: string, version: number) {
  if (!url) {
    return;
  }
  downloading.value = true;
  downloadProgress.value = 0;
  try {
    const response = await fetch(url, { mode: "cors" });
    if (!response.ok) {
      throw new Error(`下载失败 (${response.status})`);
    }
    const total = Number(response.headers.get("content-length") || 0);
    if (!response.body) {
      const blob = await response.blob();
      applyDownloadedBlob(blob, version);
      return;
    }
    const reader = response.body.getReader();
    const chunks: Uint8Array[] = [];
    let received = 0;
    while (true) {
      const { done, value } = await reader.read();
      if (done) {
        break;
      }
      if (value) {
        chunks.push(value);
        received += value.byteLength;
        if (total > 0 && version === requestVersion) {
          downloadProgress.value = Math.min(1, received / total);
        }
      }
    }
    const blob = new Blob(chunks, {
      type: props.attachment?.contentType || "application/octet-stream",
    });
    applyDownloadedBlob(blob, version);
  } catch (error) {
    if (version !== requestVersion) {
      return;
    }
    errorMessage.value =
      error instanceof Error && error.message
        ? error.message
        : "文件下载失败，请稍后重试";
  } finally {
    if (version === requestVersion) {
      downloading.value = false;
    }
  }
}

function applyDownloadedBlob(blob: Blob, version: number) {
  if (version !== requestVersion) {
    return;
  }
  revokeBlobUrl();
  downloadedBlob.value = blob;
  activeBlobUrl = window.URL.createObjectURL(blob);
  blobUrl.value = activeBlobUrl;
  downloadProgress.value = 1;
}

async function saveFile() {
  const blob = downloadedBlob.value;
  if (!blob && isVideo.value && props.attachment) {
    try {
      const access = await getChatAttachmentAccessUrl(
        props.attachment.fileID,
        { download: true }
      );
      const link = document.createElement("a");
      link.href = access.url;
      link.rel = "noopener";
      link.click();
    } catch (error) {
      Toast({
        message:
          error instanceof Error && error.message
            ? error.message
            : "文件下载失败，请稍后重试",
        type: TOAST_TYPE.ERROR,
      });
    }
    return;
  }
  if (!blob) {
    return;
  }
  const url = blobUrl.value || window.URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = fileName.value;
  link.click();
  if (!blobUrl.value) {
    window.URL.revokeObjectURL(url);
  }
}

function retry() {
  if (!props.attachment) {
    return;
  }
  void openAttachment();
}

function close() {
  emit("close");
}

function resetState(incrementVersion = true) {
  if (incrementVersion) {
    requestVersion += 1;
  }
  accessUrl.value = "";
  blobUrl.value = "";
  loading.value = false;
  downloading.value = false;
  downloadProgress.value = 0;
  errorMessage.value = "";
  downloadedBlob.value = null;
  revokeBlobUrl();
}

function revokeBlobUrl() {
  if (activeBlobUrl) {
    window.URL.revokeObjectURL(activeBlobUrl);
    activeBlobUrl = "";
  }
}

function showSaveUnavailable() {
  Toast({
    message: "文件仍在下载，请稍后保存",
    type: TOAST_TYPE.WARNING,
  });
}
</script>

<template>
  <div v-if="visible && attachment" class="chat-attachment-preview">
    <div class="chat-attachment-preview__header">
      <button
        class="chat-attachment-preview__icon-button"
        type="button"
        @click="close"
      >
        <Icon :file="closeSVG" />
      </button>
      <div class="chat-attachment-preview__title">
        <span>{{ fileName }}</span>
        <small v-if="sizeText">{{ sizeText }}</small>
      </div>
      <button
        class="chat-attachment-preview__icon-button"
        type="button"
        :disabled="!canSave"
        @click="canSave ? saveFile() : showSaveUnavailable()"
      >
        <Icon :file="downloadSVG" />
      </button>
    </div>

    <div
      v-if="downloading || hasDownloadedBlob || errorMessage"
      class="chat-attachment-preview__status"
    >
      <span v-if="downloading">
        {{ isVideo ? "正在后台下载" : "正在下载" }}
        {{ Math.round(downloadProgress * 100) }}%
      </span>
      <span v-else-if="hasDownloadedBlob">已下载，可保存到本机</span>
      <span v-else>{{ errorMessage }}</span>
      <button v-if="errorMessage" type="button" @click="retry">重试</button>
      <div v-if="downloading" class="chat-attachment-preview__progress">
        <i :style="{ width: `${Math.max(downloadProgress * 100, 4)}%` }" />
      </div>
    </div>

    <div class="chat-attachment-preview__body">
      <video
        v-if="isVideo && videoSrc"
        :key="videoSrc"
        class="chat-attachment-preview__video"
        :src="videoSrc"
        controls
        autoplay
        playsinline
        preload="metadata"
      />
      <img
        v-else-if="isImage && blobUrl"
        class="chat-attachment-preview__image"
        :src="blobUrl"
        :alt="fileName"
      />
      <div v-else class="chat-attachment-preview__empty">
        <div class="chat-attachment-preview__file-icon">
          {{ isVideo ? "MP4" : "FILE" }}
        </div>
        <strong>{{ loading ? "正在准备文件" : fileName }}</strong>
        <p v-if="!loading && !isVideo">
          当前文件类型不支持预览，可下载后按普通文件处理。
        </p>
      </div>
    </div>
  </div>
</template>

<style lang="scss" scoped>
.chat-attachment-preview {
  position: fixed;
  inset: 0;
  z-index: 30;
  display: flex;
  flex-direction: column;
  background: #0f172a;
  color: #f8fafc;
}

.chat-attachment-preview__header {
  display: grid;
  grid-template-columns: 44px minmax(0, 1fr) 44px;
  align-items: center;
  gap: 8px;
  padding: 10px 12px;
  background: rgba(15, 23, 42, 0.96);
  border-bottom: 1px solid rgba(148, 163, 184, 0.24);
}

.chat-attachment-preview__icon-button {
  width: 36px;
  height: 36px;
  border: 0;
  border-radius: 6px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: rgba(255, 255, 255, 0.08);
  cursor: pointer;

  &:disabled {
    cursor: not-allowed;
    opacity: 0.45;
  }

  :deep(.icon) {
    width: 18px;
    height: 18px;
  }
}

.chat-attachment-preview__title {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
  font-size: 14px;
  font-weight: 600;

  span {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  small {
    color: #cbd5e1;
    font-size: 12px;
    font-weight: 400;
  }
}

.chat-attachment-preview__status {
  padding: 8px 16px 10px;
  background: rgba(15, 23, 42, 0.92);
  color: #cbd5e1;
  font-size: 12px;

  button {
    margin-left: 10px;
    border: 0;
    background: transparent;
    color: #93c5fd;
    cursor: pointer;
  }
}

.chat-attachment-preview__progress {
  height: 3px;
  margin-top: 7px;
  overflow: hidden;
  border-radius: 999px;
  background: rgba(148, 163, 184, 0.24);

  i {
    display: block;
    height: 100%;
    border-radius: inherit;
    background: #60a5fa;
    transition: width 160ms ease;
  }
}

.chat-attachment-preview__body {
  min-height: 0;
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
}

.chat-attachment-preview__video,
.chat-attachment-preview__image {
  width: 100%;
  height: 100%;
  border: 0;
  object-fit: contain;
  background: #020617;
}

.chat-attachment-preview__empty {
  width: min(320px, calc(100vw - 40px));
  padding: 22px;
  border: 1px solid rgba(148, 163, 184, 0.28);
  border-radius: 8px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  text-align: center;
  color: #cbd5e1;

  strong {
    max-width: 100%;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: #f8fafc;
    font-size: 14px;
  }

  p {
    margin: 0;
    color: #94a3b8;
    font-size: 13px;
    line-height: 1.45;
  }
}

.chat-attachment-preview__file-icon {
  width: 54px;
  height: 54px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #2563eb;
  color: #fff;
  font-size: 12px;
  font-weight: 700;
}
</style>
