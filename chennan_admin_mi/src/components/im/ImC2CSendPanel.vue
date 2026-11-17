<script setup lang="ts">
import { reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { sendC2CMessages } from '@/api/im'
import ImUserSearchSelect from '@/components/im/ImUserSearchSelect.vue'
import type { BatchSendC2CResult, IMUserOption } from '@/types'
import { useResponsive } from '@/composables/useResponsive'

const sending = ref(false)
const { isMobile } = useResponsive()
const form = reactive({
  fromUser: null as IMUserOption | null,
  receivers: [] as IMUserOption[],
  content: '',
  syncToSender: true,
})
const result = ref<BatchSendC2CResult | null>(null)

async function handleSubmit() {
  const receivers = form.receivers.map(item => item.imUserID)
  if (!form.fromUser?.imUserID || receivers.length === 0 || !form.content.trim()) {
    ElMessage.warning('请填写发送方、接收方和消息内容')
    return
  }

  sending.value = true
  try {
    const res = await sendC2CMessages({
      fromUserID: form.fromUser.imUserID,
      toUserIDs: receivers,
      content: form.content.trim(),
      syncToSender: form.syncToSender,
    })
    result.value = res.data.data
    const successCount = result.value?.results.filter(item => item.success).length ?? 0
    ElMessage.success(`发送完成：${successCount}/${receivers.length}`)
  } catch (error: any) {
    ElMessage.error(error.message || '发送失败')
  } finally {
    sending.value = false
  }
}
</script>

<template>
  <div class="panel">
    <div class="card form-card">
      <ImUserSearchSelect
        v-model="form.fromUser"
        label="发送方用户"
        placeholder="输入昵称模糊搜索发送方"
      />

      <ImUserSearchSelect
        v-model="form.receivers"
        label="接收方用户"
        placeholder="输入昵称模糊搜索接收方，可多选"
        multiple
      />

      <div class="form-field">
        <label class="form-label">消息内容</label>
        <el-input v-model="form.content" type="textarea" :rows="4" placeholder="请输入文本消息内容" />
      </div>

      <div class="form-field form-switch">
        <div>
          <p class="form-label">同步给发送方其他设备</p>
          <p class="form-hint">开启后，发送方的其他终端也会同步到这条消息。</p>
        </div>
        <el-switch v-model="form.syncToSender" />
      </div>

      <div class="form-actions">
        <el-button type="primary" :loading="sending" @click="handleSubmit">
          <template #icon><el-icon><Promotion /></el-icon></template>
          发送消息
        </el-button>
      </div>

      <p class="form-hint">点击输入框可先查看待选昵称，也可以继续输入昵称片段做模糊搜索，然后直接点选用户。</p>
    </div>

    <div v-if="result" class="card result-card">
      <div class="result-head">
        <div>
          <p class="section-title">发送结果</p>
          <p class="section-sub">消息主键：<span class="font-code">{{ result.msgKey || '-' }}</span></p>
        </div>
      </div>

      <div class="table-scroll-x result-table-wrap">
        <el-table :data="result.results" stripe size="small" style="width: 100%">
        <el-table-column label="接收方" min-width="180">
          <template #default="{ row }">
            <span class="font-code cell-muted">{{ row.toUserID }}</span>
          </template>
        </el-table-column>

        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.success ? 'success' : 'danger'" size="small" effect="light" round>
              {{ row.success ? '成功' : '失败' }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column v-if="!isMobile" label="详情" min-width="180">
          <template #default="{ row }">
            <span class="cell-muted">{{ row.message }}</span>
          </template>
        </el-table-column>
        </el-table>
      </div>
    </div>
  </div>
</template>

<style scoped>
.panel {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.form-card,
.result-card {
  padding: 20px 22px;
}

.form-card {
  display: flex;
  flex-direction: column;
  gap: 16px;
  overflow: visible;
}

.form-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.form-switch {
  flex-direction: row;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.form-label {
  margin: 0;
  font-size: 12px;
  font-weight: 500;
  color: #475569;
}

.form-hint {
  margin: 4px 0 0;
  font-size: 12px;
  color: #94a3b8;
}

.form-actions {
  display: flex;
  justify-content: flex-start;
}

.result-head {
  margin-bottom: 12px;
}

.result-table-wrap {
  border-bottom: 1px solid #f0f0f0;
}

.section-title {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
  color: #1e293b;
}

.section-sub {
  margin: 4px 0 0;
  font-size: 12px;
  color: #94a3b8;
}

.cell-muted {
  font-size: 12px;
  color: #64748b;
}

@media (max-width: 767px) {
  .form-card,
  .result-card {
    padding: 16px 14px;
  }

  .form-switch,
  .form-actions {
    flex-direction: column;
    align-items: stretch;
  }

  .form-actions :deep(.el-button) {
    width: 100%;
  }
}
</style>
