<script setup lang="ts">
import { computed, reactive, ref, watch } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import {
  cleanupAdminAudits,
  previewAdminAuditCleanup,
  type AdminAuditCleanupScope,
  type AuditCleanupPreview,
} from "@/api/audit";

const props = withDefaults(
  defineProps<{
    modelValue: boolean;
    defaultScope?: AdminAuditCleanupScope;
    cleanupFilters?: {
      keyword?: string;
      role?: string;
      accessType?: string;
      method?: string;
      stage?: string;
      result?: string;
    };
  }>(),
  {
    defaultScope: "access",
    cleanupFilters: () => ({}),
  },
);

const emit = defineEmits<{
  "update:modelValue": [value: boolean];
  "cleanup-success": [];
}>();

const previewLoading = ref(false);
const cleanupLoading = ref(false);
const previewResult = ref<AuditCleanupPreview | null>(null);

const form = reactive({
  scope: props.defaultScope as AdminAuditCleanupScope,
  beforeTime: "",
});

const scopeOptions: Array<{ label: string; value: AdminAuditCleanupScope }> = [
  { label: "访问审计", value: "access" },
  { label: "登录审计", value: "login" },
  { label: "全部审计", value: "all" },
];

const scopeLabelMap: Record<AdminAuditCleanupScope, string> = {
  access: "访问审计",
  login: "登录审计",
  all: "全部审计",
};

const canSubmitCleanup = computed(() => {
  return Boolean(
    previewResult.value &&
    previewResult.value.totalCount > 0 &&
    previewResult.value.scope === form.scope &&
    previewResult.value.beforeTime === form.beforeTime,
  );
});

watch(
  () => props.modelValue,
  (visible) => {
    if (!visible) {
      return;
    }
    form.scope = props.defaultScope;
    form.beforeTime = "";
    previewResult.value = null;
  },
);

watch(
  () => props.defaultScope,
  (scope) => {
    if (props.modelValue) {
      form.scope = scope;
      previewResult.value = null;
    }
  },
);

watch(
  () => [form.scope, form.beforeTime],
  () => {
    previewResult.value = null;
  },
);

watch(
  () => props.cleanupFilters,
  () => {
    previewResult.value = null;
  },
  { deep: true },
);

function buildCleanupPayload() {
  return {
    scope: form.scope,
    before_time: form.beforeTime,
    keyword: props.cleanupFilters.keyword || undefined,
    role: props.cleanupFilters.role || undefined,
    accessType: props.cleanupFilters.accessType || undefined,
    method: props.cleanupFilters.method || undefined,
    stage: props.cleanupFilters.stage || undefined,
    result: props.cleanupFilters.result || undefined,
  };
}

function closeDialog() {
  emit("update:modelValue", false);
}

async function handlePreview() {
  if (!form.beforeTime) {
    ElMessage.warning("请先选择清理时间");
    return;
  }

  previewLoading.value = true;
  try {
    const res = await previewAdminAuditCleanup(buildCleanupPayload());
    previewResult.value = {
      ...res.data.data,
      beforeTime: form.beforeTime,
    };
  } catch (error: any) {
    ElMessage.error(error.message || "预览清理失败");
  } finally {
    previewLoading.value = false;
  }
}

async function handleCleanup() {
  if (!form.beforeTime) {
    ElMessage.warning("请先选择清理时间");
    return;
  }
  if (!previewResult.value) {
    ElMessage.warning("请先预览待清理数量");
    return;
  }
  if (previewResult.value.totalCount <= 0) {
    ElMessage.info("当前条件下没有可清理的审计日志");
    return;
  }

  try {
    await ElMessageBox.confirm(
      `确认清理 ${scopeLabelMap[form.scope]}中 ${form.beforeTime} 之前的 ${previewResult.value.totalCount} 条日志？此操作不可恢复。`,
      "确认清理",
      {
        type: "warning",
        confirmButtonText: "确认清理",
        cancelButtonText: "取消",
      },
    );
  } catch {
    return;
  }

  cleanupLoading.value = true;
  try {
    const res = await cleanupAdminAudits(buildCleanupPayload());
    const result = res.data.data;
    ElMessage.success(`已清理 ${result.totalDeleted} 条审计日志`);
    emit("cleanup-success");
    closeDialog();
  } catch (error: any) {
    ElMessage.error(error.message || "清理审计日志失败");
  } finally {
    cleanupLoading.value = false;
  }
}
</script>

<template>
  <el-dialog
    :model-value="modelValue"
    title="清理审计日志"
    width="520px"
    @close="closeDialog"
    @update:model-value="(value: boolean) => emit('update:modelValue', value)"
  >
    <div class="cleanup-dialog">
      <el-alert
        title="仅支持清理指定时间之前的审计日志，执行后不可恢复。"
        type="warning"
        :closable="false"
        show-icon
      />

      <el-form label-width="92px" class="cleanup-form">
        <el-form-item label="清理范围">
          <el-select v-model="form.scope" class="form-control">
            <el-option
              v-for="option in scopeOptions"
              :key="option.value"
              :label="option.label"
              :value="option.value"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="截止时间">
          <el-date-picker
            v-model="form.beforeTime"
            type="datetime"
            value-format="YYYY-MM-DD HH:mm:ss"
            format="YYYY-MM-DD HH:mm:ss"
            placeholder="选择截止时间（清理该时间之前的日志）"
            class="form-control"
          />
        </el-form-item>
      </el-form>

      <div class="preview-actions">
        <el-button :loading="previewLoading" @click="handlePreview"
          >预览数量</el-button
        >
      </div>

      <div v-if="previewResult" class="preview-result">
        <div class="preview-row">
          <span class="preview-label">访问审计</span>
          <span class="preview-value">{{ previewResult.accessCount }}</span>
        </div>
        <div class="preview-row">
          <span class="preview-label">登录审计</span>
          <span class="preview-value">{{ previewResult.loginCount }}</span>
        </div>
        <div class="preview-row preview-row-total">
          <span class="preview-label">合计</span>
          <span class="preview-value">{{ previewResult.totalCount }}</span>
        </div>
      </div>
    </div>

    <template #footer>
      <div class="dialog-footer">
        <el-button @click="closeDialog">取消</el-button>
        <el-button
          type="danger"
          :loading="cleanupLoading"
          :disabled="!canSubmitCleanup"
          @click="handleCleanup"
        >
          确认清理
        </el-button>
      </div>
    </template>
  </el-dialog>
</template>

<style scoped>
.cleanup-dialog {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.cleanup-form {
  margin-top: 4px;
}

.form-control {
  width: 100%;
}

.preview-actions {
  display: flex;
  justify-content: flex-end;
}

.preview-result {
  border: 1px solid var(--el-border-color-light);
  border-radius: 12px;
  background: var(--el-fill-color-blank);
  padding: 12px 14px;
}

.preview-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 0;
  color: var(--el-text-color-regular);
}

.preview-row + .preview-row {
  border-top: 1px dashed var(--el-border-color-lighter);
}

.preview-row-total {
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.preview-value {
  font-variant-numeric: tabular-nums;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}
</style>
