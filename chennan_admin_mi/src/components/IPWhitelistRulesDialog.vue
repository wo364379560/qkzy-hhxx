<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { getAdminIPWhitelist, saveAdminIPWhitelist } from '@/api/admin'
import { getIPWhitelistDetail, saveIPWhitelist } from '@/api/ipWhitelist'
import { useResponsive } from '@/composables/useResponsive'

type WhitelistSubject = 'assistant' | 'manager'

interface WhitelistAccountIdentity {
  id: number
  username: string
}

interface EditableRule {
  id?: number
  cidr: string
  note: string
}

const props = defineProps<{
  visible: boolean
  account: WhitelistAccountIdentity | null
  subject?: WhitelistSubject
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  'success': []
}>()

const loading = ref(false)
const saving = ref(false)
const { isMobile } = useResponsive()
const accountLabel = computed(() => props.subject === 'manager' ? '管理员' : '助理')
const form = reactive<{
  rules: EditableRule[]
}>({
  rules: [],
})

watch(
  () => props.visible,
  async (visible) => {
    if (!visible || !props.account) {
      return
    }
    await loadWhitelist()
  },
)

function closeDialog() {
  emit('update:visible', false)
}

async function loadWhitelist() {
  if (!props.account) return

  loading.value = true
  try {
    const res = props.subject === 'manager'
      ? await getAdminIPWhitelist(props.account.id)
      : await getIPWhitelistDetail(props.account.id)
    const data = res.data.data
    form.rules = (data?.rules ?? []).map((rule) => ({
      id: rule.id,
      cidr: rule.cidr,
      note: rule.note,
    }))
    if (form.rules.length === 0) {
      form.rules.push({ cidr: '', note: '' })
    }
  } catch (error: any) {
    ElMessage.error(error.message || '加载 IP 白名单失败')
  } finally {
    loading.value = false
  }
}

function addRule() {
  form.rules.push({ cidr: '', note: '' })
}

function removeRule(index: number) {
  form.rules.splice(index, 1)
  if (form.rules.length === 0) {
    form.rules.push({ cidr: '', note: '' })
  }
}

async function handleSave() {
  if (!props.account) return

  const rules = form.rules
    .map((rule) => ({
      cidr: rule.cidr.trim(),
      note: rule.note.trim(),
    }))
    .filter((rule) => rule.cidr)

  saving.value = true
  try {
    if (props.subject === 'manager') {
      await saveAdminIPWhitelist(props.account.id, { rules })
    } else {
      await saveIPWhitelist(props.account.id, { rules })
    }
    ElMessage.success('IP 白名单已保存')
    emit('success')
    closeDialog()
  } catch (error: any) {
    ElMessage.error(error.message || '保存 IP 白名单失败')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <el-dialog
    :model-value="visible"
    :title="account ? `IP 白名单 · ${account.username}` : 'IP 白名单'"
    :width="isMobile ? '100%' : '720px'"
    :fullscreen="isMobile"
    :close-on-click-modal="false"
    @close="closeDialog"
  >
    <div v-loading="loading" class="whitelist-dialog">
      <div class="whitelist-summary">
        <div>
          <div class="whitelist-title">填写规则后自动启用{{ accountLabel }} IP 白名单</div>
          <div class="whitelist-subtitle">
            支持单 IP 或 CIDR。保留规则时自动生效，清空全部规则则自动关闭限制。
          </div>
        </div>
      </div>

      <div class="whitelist-toolbar">
        <div class="whitelist-toolbar-text">
          规则列表
          <span class="whitelist-toolbar-note">单 IP 会由服务端自动标准化为 /32 或 /128</span>
        </div>
        <el-button plain type="primary" @click="addRule">新增规则</el-button>
      </div>

      <div class="whitelist-table">
        <div
          v-for="(rule, index) in form.rules"
          :key="`${rule.id ?? 'new'}-${index}`"
          class="whitelist-row"
        >
          <el-input
            v-model="rule.cidr"
            placeholder="例如 1.2.3.4 或 10.0.0.0/24"
            class="whitelist-cidr"
          />
          <el-input
            v-model="rule.note"
            placeholder="备注，可选"
            class="whitelist-note"
          />
          <el-button
            plain
            type="danger"
            @click="removeRule(index)"
          >
            删除
          </el-button>
        </div>
      </div>
    </div>

    <template #footer>
      <el-button @click="closeDialog">取消</el-button>
      <el-button type="primary" :loading="saving" @click="handleSave">保存</el-button>
    </template>
  </el-dialog>
</template>

<style scoped>
.whitelist-dialog {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.whitelist-summary {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 16px 18px;
  border-radius: 16px;
  background: linear-gradient(135deg, #f5f9ff 0%, #eef4ff 100%);
  border: 1px solid #dbe7ff;
}

.whitelist-title {
  font-size: 15px;
  font-weight: 700;
  color: #123257;
}

.whitelist-subtitle {
  margin-top: 6px;
  font-size: 13px;
  line-height: 1.6;
  color: #5b6b7f;
}

.whitelist-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.whitelist-toolbar-text {
  font-size: 14px;
  font-weight: 700;
  color: #1f2a37;
}

.whitelist-toolbar-note {
  margin-left: 8px;
  font-size: 12px;
  font-weight: 500;
  color: #6b7280;
}

.whitelist-table {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.whitelist-row {
  display: grid;
  grid-template-columns: minmax(220px, 1.2fr) minmax(180px, 1fr) auto;
  gap: 12px;
}

.whitelist-cidr,
.whitelist-note {
  width: 100%;
}

@media (max-width: 767px) {
  .whitelist-summary,
  .whitelist-toolbar {
    flex-direction: column;
    align-items: stretch;
  }

  .whitelist-row {
    grid-template-columns: 1fr;
  }
}
</style>
