<script setup lang="ts">
import { reactive } from 'vue';
import { ElMessage } from 'element-plus';
import { submitReport } from '@/api';
import { closeReportDialog, useReportDialog } from '@/features/report/dialog';
import { useSingleFlight } from '@/composables/useSingleFlight';

const reportDialog = useReportDialog();
const submitFlight = useSingleFlight();
const submitting = submitFlight.running;

const form = reactive({
  category: 'spam' as 'spam' | 'abuse' | 'fraud' | 'other',
  description: '',
});

const categories = [
  { value: 'spam', label: '垃圾信息' },
  { value: 'abuse', label: '违规内容' },
  { value: 'fraud', label: '诈骗' },
  { value: 'other', label: '其他' },
];

async function handleSubmit() {
  if (!reportDialog.state.targetId) {
    return;
  }

  try {
    await submitFlight.run(async () => {
      await submitReport({
        targetId: reportDialog.state.targetId,
        targetType: reportDialog.state.targetType,
        targetName: reportDialog.state.targetName,
        category: form.category,
        description: form.description.trim() || undefined,
      });
      ElMessage.success('举报已提交，我们将尽快处理');
      handleClose();
    });
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '提交举报失败');
  }
}

function handleClose() {
  form.category = 'spam';
  form.description = '';
  closeReportDialog();
}
</script>

<template>
  <el-dialog
    :model-value="reportDialog.state.visible"
    title="举报"
    width="420px"
    @close="handleClose"
  >
    <div class="report-dialog">
      <p class="report-dialog-target">
        举报对象：{{ reportDialog.state.targetName || reportDialog.state.targetId }}
      </p>
      <el-radio-group
        v-model="form.category"
        class="report-dialog-category"
      >
        <el-radio
          v-for="item in categories"
          :key="item.value"
          :label="item.value"
        >
          {{ item.label }}
        </el-radio>
      </el-radio-group>
      <el-input
        v-model="form.description"
        type="textarea"
        :rows="4"
        maxlength="200"
        show-word-limit
        placeholder="补充说明（选填）"
      />
    </div>
    <template #footer>
      <span class="report-dialog-footer">
        <el-button @click="handleClose">取消</el-button>
        <el-button
          type="primary"
          :loading="submitting"
          @click="handleSubmit"
        >
          提交举报
        </el-button>
      </span>
    </template>
  </el-dialog>
</template>

<style scoped lang="scss">
.report-dialog {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.report-dialog-target {
  margin: 0;
  color: #334155;
  font-size: 14px;
}

.report-dialog-category {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 8px;
}

.report-dialog-footer {
  display: flex;
  justify-content: flex-end;
}
</style>
