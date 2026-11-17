<script lang="ts" setup>
import { computed, withDefaults } from "../../../adapter-vue";
import Icon from "../../common/Icon.vue";
import fileIcon from "../../../assets/icon/file-light.svg";
import type { LargeAttachmentUploadTask } from "../utils/largeAttachmentUploadTasks";

const props = withDefaults(
  defineProps<{
    task: LargeAttachmentUploadTask;
  }>(),
  {
    task: () => ({} as LargeAttachmentUploadTask),
  }
);

const emit = defineEmits<{
  (key: "cancel", taskID: string): void;
  (key: "retry", taskID: string): void;
  (key: "close", taskID: string): void;
}>();

const percent = computed(() =>
  Math.round(Math.max(0, Math.min(1, props.task.progress || 0)) * 100)
);

const statusText = computed(() => {
  switch (props.task.status) {
    case "preparing":
      return "准备上传";
    case "sending":
      return "正在发送";
    case "failed":
      return props.task.errorMessage || "上传失败";
    case "uploading":
    default:
      return "上传中";
  }
});

const canCancel = computed(
  () =>
    Boolean(props.task.cancel) &&
    (props.task.status === "preparing" || props.task.status === "uploading")
);

function formatFileSize(size: number): string {
  if (!Number.isFinite(size) || size <= 0) {
    return "";
  }
  const mb = size / 1024 / 1024;
  if (mb >= 1) {
    return `${mb.toFixed(mb >= 10 ? 0 : 1)}MB`;
  }
  const kb = size / 1024;
  return `${kb.toFixed(kb >= 10 ? 0 : 1)}KB`;
}
</script>

<template>
  <div class="large-attachment-upload-message">
    <div
      class="large-attachment-upload-card"
      :class="{
        'large-attachment-upload-card-failed': task.status === 'failed',
      }"
      role="status"
      aria-live="polite"
    >
      <div class="large-attachment-upload-main">
        <Icon class="large-attachment-upload-icon" :file="fileIcon" />
        <div class="large-attachment-upload-content">
          <div class="large-attachment-upload-title" :title="task.fileName">
            {{ task.fileName || "文件" }}
          </div>
          <div class="large-attachment-upload-meta">
            <span>{{ formatFileSize(task.fileSize) }}</span>
            <span>{{ statusText }}</span>
            <span v-if="task.status !== 'failed'">{{ percent }}%</span>
          </div>
        </div>
      </div>

      <div
        v-if="task.status !== 'failed'"
        class="large-attachment-upload-track"
      >
        <div
          class="large-attachment-upload-bar"
          :style="{ width: `${percent}%` }"
        />
      </div>

      <div class="large-attachment-upload-actions">
        <button
          v-if="canCancel"
          class="large-attachment-upload-action"
          type="button"
          @click.stop="emit('cancel', task.id)"
        >
          取消
        </button>
        <button
          v-if="task.status === 'failed'"
          class="large-attachment-upload-action"
          type="button"
          @click.stop="emit('retry', task.id)"
        >
          重试
        </button>
        <button
          v-if="task.status === 'failed'"
          class="large-attachment-upload-action"
          type="button"
          @click.stop="emit('close', task.id)"
        >
          关闭
        </button>
      </div>
    </div>
  </div>
</template>

<style lang="scss" scoped>
.large-attachment-upload-message {
  display: flex;
  justify-content: flex-end;
  padding: 0 16px;
}

.large-attachment-upload-card {
  box-sizing: border-box;
  width: min(340px, 78vw);
  padding: 12px;
  border-radius: 8px;
  background: #d9e8ff;
  color: #1f2329;
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.08);
}

.large-attachment-upload-card-failed {
  background: #fff1f0;
}

.large-attachment-upload-main {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.large-attachment-upload-icon {
  flex: 0 0 auto;
  width: 36px;
  height: 36px;
}

.large-attachment-upload-content {
  min-width: 0;
  flex: 1;
}

.large-attachment-upload-title {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 14px;
  line-height: 20px;
  font-weight: 500;
}

.large-attachment-upload-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 3px;
  color: #606266;
  font-size: 12px;
  line-height: 17px;
}

.large-attachment-upload-track {
  height: 4px;
  margin-top: 10px;
  overflow: hidden;
  border-radius: 2px;
  background: rgba(255, 255, 255, 0.82);
}

.large-attachment-upload-bar {
  height: 100%;
  border-radius: 2px;
  background: #006eff;
  transition: width 120ms ease;
}

.large-attachment-upload-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 10px;
}

.large-attachment-upload-action {
  border: 0;
  padding: 0;
  background: transparent;
  color: #006eff;
  font-size: 13px;
  line-height: 18px;
  cursor: pointer;
}
</style>
