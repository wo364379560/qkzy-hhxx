<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { useResponsive } from '@/composables/useResponsive'
import {
  clearAssistantChatQuietHoursPolicy,
  clearManagerChatQuietHoursPolicy,
  getAssistantChatQuietHoursPolicy,
  getManagerChatQuietHoursPolicy,
  saveAssistantChatQuietHoursPolicy,
  saveManagerChatQuietHoursPolicy,
  type AccountChatQuietHoursPolicy,
} from '@/api/chatQuietHoursPolicy'
import type { ChatQuietHoursSettings } from '@/api/auth'

type PolicySubject = 'manager' | 'assistant'

interface PolicyAccountIdentity {
  id: number
  username: string
}

const props = defineProps<{
  visible: boolean
  account: PolicyAccountIdentity | null
  subject: PolicySubject
  editable?: boolean
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  success: []
}>()

const { isMobile } = useResponsive()
const loading = ref(false)
const saving = ref(false)
const clearing = ref(false)
const hasPolicy = ref(false)
const policyRole = ref<'admin' | 'assistant'>('assistant')

const form = reactive<ChatQuietHoursSettings & { current_password: string }>({
  enabled: false,
  timezone: 'Asia/Shanghai',
  start: '00:00',
  end: '08:00',
  block_register: true,
  block_send_message: true,
  block_create_group: true,
  block_add_friend: true,
  current_password: '',
})

const subjectLabel = computed(() => props.subject === 'manager' ? '管理员' : '助理')
const dialogTitle = computed(() => props.account ? `定时策略 · ${props.account.username}` : '定时策略')
const readonlyHint = computed(() => props.editable ? '未配置时该管理域默认不受时段限制。' : '当前账号只能查看该策略。')
const actionSummary = computed(() => {
  const actions: string[] = []
  if (form.block_register) actions.push('注册')
  if (form.block_send_message) actions.push('发消息')
  if (form.block_create_group) actions.push('建群 / 入群 / 拉人入群')
  if (form.block_add_friend) actions.push('加好友')
  return actions.length > 0 ? actions.join(' / ') : '未启用任何限制项'
})

watch(
  () => props.visible,
  async (visible) => {
    if (!visible || !props.account) {
      return
    }
    await loadPolicy()
  },
)

function resetForm(policy?: ChatQuietHoursSettings) {
  form.enabled = policy?.enabled ?? false
  form.timezone = policy?.timezone ?? 'Asia/Shanghai'
  form.start = policy?.start ?? '00:00'
  form.end = policy?.end ?? '08:00'
  form.block_register = policy?.block_register ?? true
  form.block_send_message = policy?.block_send_message ?? true
  form.block_create_group = policy?.block_create_group ?? true
  form.block_add_friend = policy?.block_add_friend ?? true
  form.current_password = ''
}

function closeDialog() {
  emit('update:visible', false)
}

async function loadPolicy() {
  if (!props.account) return

  loading.value = true
  try {
    const res = props.subject === 'manager'
      ? await getManagerChatQuietHoursPolicy(props.account.id)
      : await getAssistantChatQuietHoursPolicy(props.account.id)
    const data: AccountChatQuietHoursPolicy | undefined = res.data.data
    hasPolicy.value = Boolean(data?.hasPolicy)
    policyRole.value = data?.role === 'admin' ? 'admin' : 'assistant'
    resetForm(data?.policy)
  } catch (error: any) {
    ElMessage.error(error.message || '加载定时策略失败')
  } finally {
    loading.value = false
  }
}

async function handleSave() {
  if (!props.account || !props.editable) return
  if (!form.current_password.trim()) {
    ElMessage.warning('请输入当前密码')
    return
  }

  saving.value = true
  try {
    const payload = {
      enabled: form.enabled,
      timezone: form.timezone.trim(),
      start: form.start.trim(),
      end: form.end.trim(),
      block_register: form.block_register,
      block_send_message: form.block_send_message,
      block_create_group: form.block_create_group,
      block_add_friend: form.block_add_friend,
      current_password: form.current_password.trim(),
    }
    if (props.subject === 'manager') {
      await saveManagerChatQuietHoursPolicy(props.account.id, payload)
    } else {
      await saveAssistantChatQuietHoursPolicy(props.account.id, payload)
    }
    ElMessage.success('定时策略已保存')
    emit('success')
    closeDialog()
  } catch (error: any) {
    ElMessage.error(error.message || '保存定时策略失败')
  } finally {
    saving.value = false
  }
}

async function handleClear() {
  if (!props.account || !props.editable || !hasPolicy.value) return
  if (!form.current_password.trim()) {
    ElMessage.warning('请输入当前密码')
    return
  }

  clearing.value = true
  try {
    if (props.subject === 'manager') {
      await clearManagerChatQuietHoursPolicy(props.account.id, form.current_password.trim())
    } else {
      await clearAssistantChatQuietHoursPolicy(props.account.id, form.current_password.trim())
    }
    ElMessage.success('已清除定时策略')
    emit('success')
    closeDialog()
  } catch (error: any) {
    ElMessage.error(error.message || '清除定时策略失败')
  } finally {
    clearing.value = false
  }
}
</script>

<template>
  <el-dialog
    :model-value="visible"
    :title="dialogTitle"
    :width="isMobile ? '100%' : '700px'"
    :fullscreen="isMobile"
    :close-on-click-modal="false"
    @close="closeDialog"
  >
    <div v-loading="loading" class="policy-dialog">
      <div class="policy-summary">
        <div>
          <div class="policy-title">
            {{ subjectLabel }}策略
            <span class="policy-role">{{ policyRole === 'admin' ? '管理员' : '助理' }}</span>
          </div>
          <div class="policy-subtitle">
            {{ hasPolicy ? '当前已配置单独策略，将仅影响该管理域。' : '当前未配置单独策略。' }}
          </div>
        </div>
        <div class="policy-status" :class="hasPolicy ? 'status-active' : 'status-default'">
          {{ hasPolicy ? (form.enabled ? '已启用' : '已配置但关闭') : '未配置' }}
        </div>
      </div>

      <div class="policy-grid">
        <div class="policy-field">
          <label class="policy-label">是否启用</label>
          <el-switch v-model="form.enabled" :disabled="!editable" />
        </div>
        <div class="policy-field">
          <label class="policy-label">时区</label>
          <el-input v-model="form.timezone" :disabled="!editable" />
        </div>
        <div class="policy-field">
          <label class="policy-label">开始时间</label>
          <el-input v-model="form.start" placeholder="00:00" :disabled="!editable" />
        </div>
        <div class="policy-field">
          <label class="policy-label">结束时间</label>
          <el-input v-model="form.end" placeholder="08:00" :disabled="!editable" />
        </div>
      </div>

      <div class="policy-section">
        <div class="policy-section-title">限制动作</div>
        <div class="policy-checkboxes">
          <el-checkbox v-model="form.block_register" :disabled="!editable">禁止注册</el-checkbox>
          <el-checkbox v-model="form.block_send_message" :disabled="!editable">禁止发消息</el-checkbox>
          <el-checkbox v-model="form.block_create_group" :disabled="!editable">禁止建群 / 入群 / 拉人入群</el-checkbox>
          <el-checkbox v-model="form.block_add_friend" :disabled="!editable">禁止加好友</el-checkbox>
        </div>
        <div class="policy-note">当前动作集合：{{ actionSummary }}</div>
      </div>

      <div v-if="editable" class="policy-section">
        <div class="policy-section-title">安全确认</div>
        <el-input
          v-model="form.current_password"
          type="password"
          show-password
          placeholder="请输入当前后台密码"
        />
      </div>

      <div class="policy-readonly-hint">
        {{ readonlyHint }}
      </div>
    </div>

    <template #footer>
      <el-button @click="closeDialog">关闭</el-button>
      <template v-if="editable">
        <el-button
          v-if="hasPolicy"
          plain
          type="danger"
          :loading="clearing"
          @click="handleClear"
        >
          清除策略
        </el-button>
        <el-button type="primary" :loading="saving" @click="handleSave">
          保存策略
        </el-button>
      </template>
    </template>
  </el-dialog>
</template>

<style scoped>
.policy-dialog {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.policy-summary {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 16px 18px;
  border-radius: 16px;
  background: linear-gradient(135deg, #f7fbff 0%, #eef4ff 100%);
  border: 1px solid #dbe7ff;
}

.policy-title {
  font-size: 15px;
  font-weight: 700;
  color: #123257;
}

.policy-role {
  display: inline-flex;
  align-items: center;
  margin-left: 8px;
  padding: 2px 8px;
  border-radius: 999px;
  font-size: 11px;
  font-weight: 600;
  color: #475569;
  background: rgba(255, 255, 255, 0.8);
}

.policy-subtitle {
  margin-top: 6px;
  font-size: 13px;
  line-height: 1.6;
  color: #5b6b7f;
}

.policy-status {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 92px;
  height: 32px;
  padding: 0 12px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 600;
  border: 1px solid transparent;
}

.status-active {
  color: #16a34a;
  border-color: #bbf7d0;
  background: #f0fdf4;
}

.status-default {
  color: #64748b;
  border-color: #e2e8f0;
  background: #f8fafc;
}

.policy-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
}

.policy-field {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.policy-label,
.policy-section-title {
  font-size: 13px;
  font-weight: 600;
  color: #334155;
}

.policy-section {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.policy-checkboxes {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px 14px;
}

.policy-note,
.policy-readonly-hint {
  font-size: 12px;
  line-height: 1.6;
  color: #64748b;
}

@media (max-width: 767px) {
  .policy-summary {
    flex-direction: column;
    align-items: flex-start;
  }

  .policy-grid,
  .policy-checkboxes {
    grid-template-columns: 1fr;
  }
}
</style>
