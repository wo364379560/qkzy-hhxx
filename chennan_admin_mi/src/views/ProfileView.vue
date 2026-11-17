<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '@/stores/auth'
import BackupCodesDialog from '@/components/BackupCodesDialog.vue'
import {
  getBackupCodes,
  regenerateBackupCodes,
  type BackupCodesStatus,
} from '@/api/auth'

const authStore = useAuthStore()
const loading = ref(false)
const regenerating = ref(false)
const showRegenerateDialog = ref(false)
const showNewCodesDialog = ref(false)
const password = ref('')
const newCodes = ref<string[]>([])

function emptyBackupStatus(): BackupCodesStatus {
  return {
    totp_enabled: false,
    remaining_count: 0,
    placeholders: [],
  }
}

const backupStatus = ref<BackupCodesStatus>(emptyBackupStatus())

async function fetchCodes() {
  loading.value = true
  try {
    const res = await getBackupCodes()
    backupStatus.value = res.data.data ?? emptyBackupStatus()
  } catch (e: any) {
    backupStatus.value = emptyBackupStatus()
    ElMessage.error(e.message || '获取备份码失败')
  } finally {
    loading.value = false
  }
}

async function handleRegenerate() {
  if (!backupStatus.value.totp_enabled) {
    ElMessage.warning('当前账号尚未绑定双因素认证')
    return
  }
  if (!password.value) {
    ElMessage.warning('请输入密码')
    return
  }

  regenerating.value = true
  try {
    const res = await regenerateBackupCodes(password.value)
    newCodes.value = res.data.data?.backup_codes ?? []
    ElMessage.success('备份码已重新生成')
    showRegenerateDialog.value = false
    showNewCodesDialog.value = true
    password.value = ''
    await fetchCodes()
  } catch (e: any) {
    ElMessage.error(e.message || '重新生成失败')
  } finally {
    regenerating.value = false
  }
}

onMounted(fetchCodes)
</script>

<template>
  <div class="page">
    <div class="card">
      <h1>个人中心</h1>
      <div class="profile-section">
        <div class="profile-item">
          <span class="profile-label">用户名</span>
          <span class="profile-value">{{ authStore.user?.username }}</span>
        </div>
        <div class="profile-item">
          <span class="profile-label">角色</span>
          <span class="profile-value">
            {{
              authStore.user?.role === 'super_admin'
                ? '超级管理员'
                : authStore.user?.role === 'admin'
                  ? '管理员'
                  : '助理'
            }}
          </span>
        </div>
      </div>
    </div>

    <div class="card" style="margin-top: 16px">
      <div class="card-header">
        <h2>备份码管理</h2>
        <el-button
          v-if="backupStatus.totp_enabled"
          size="small"
          type="primary"
          @click="showRegenerateDialog = true"
        >
          重新生成
        </el-button>
      </div>
      <p class="desc" v-if="backupStatus.totp_enabled">
        当前剩余 {{ backupStatus.remaining_count }} 个备份码，明文仅在生成时展示一次。
      </p>
      <p class="desc" v-else>
        当前账号尚未绑定双因素认证。
      </p>

      <div v-if="backupStatus.totp_enabled" v-loading="loading" class="codes-grid">
        <code v-for="(code, i) in backupStatus.placeholders" :key="i" class="backup-code">{{ code }}</code>
        <div v-if="!backupStatus.placeholders.length" class="codes-empty">暂无可用备份码</div>
      </div>
      <div v-else class="codes-empty">未绑定双因素认证</div>
    </div>

    <el-dialog v-model="showRegenerateDialog" title="重新生成备份码" width="400px" :close-on-click-modal="false">
      <p class="dialog-danger">重新生成后，旧的备份码将全部失效</p>
      <el-input v-model="password" type="password" placeholder="请输入密码确认" show-password />
      <template #footer>
        <el-button @click="showRegenerateDialog = false">取消</el-button>
        <el-button type="primary" :loading="regenerating" @click="handleRegenerate">确认生成</el-button>
      </template>
    </el-dialog>

    <BackupCodesDialog v-model:visible="showNewCodesDialog" :codes="newCodes" />
  </div>
</template>

<style scoped>
.profile-section {
  display: flex;
  flex-direction: column;
  gap: 16px;
  margin-top: 24px;
}

.profile-item {
  display: flex;
  align-items: center;
  gap: 16px;
}

.profile-label {
  font-size: 14px;
  color: #64748b;
  width: 80px;
}

.profile-value {
  font-size: 14px;
  color: #1e293b;
  font-weight: 500;
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}

.desc {
  color: #64748b;
  font-size: 14px;
  margin-bottom: 16px;
}

.codes-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 12px;
}

.backup-code {
  background: #f8fafc;
  padding: 12px;
  border-radius: 6px;
  text-align: center;
  font-family: 'Fira Code', monospace;
  font-size: 14px;
  color: #1e293b;
  border: 1px solid #e2e8f0;
}

.codes-empty {
  padding: 16px;
  text-align: center;
  color: #94a3b8;
  background: #f8fafc;
  border: 1px dashed #dbe2ea;
  border-radius: 8px;
}

.dialog-danger {
  color: #ef4444;
  margin-bottom: 16px;
}
</style>
