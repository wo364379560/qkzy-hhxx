<script setup lang="ts">
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { verifyTOTP } from '@/api/auth'
import type { AdminUser } from '@/types'

const props = defineProps<{
  visible: boolean
  tempToken: string
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  'success': [data: { token: string; user: AdminUser }]
}>()

const useBackup = ref(false)
const code = ref('')
const backupCode = ref('')
const loading = ref(false)

watch(() => [props.visible, props.tempToken], () => {
  useBackup.value = false
  code.value = ''
  backupCode.value = ''
  loading.value = false
})

const handleVerify = async () => {
  if (loading.value) return
  const submittedToken = props.tempToken
  code.value = code.value.replace(/\s/g, '')
  const payload: any = {}
  if (useBackup.value) {
    if (!backupCode.value) {
      ElMessage.warning('请输入备份码')
      return
    }
    payload.backup_code = backupCode.value
  } else {
    if (!/^\d{6}$/.test(code.value)) {
      ElMessage.warning('请输入 6 位验证码')
      return
    }
    payload.code = code.value
  }

  loading.value = true
  try {
    const res = await verifyTOTP(submittedToken, payload)
    if (!props.visible || props.tempToken !== submittedToken) return
    emit('success', {
      token: res.data.data.token,
      user: res.data.data.user,
    })
    emit('update:visible', false)
  } catch (e: any) {
    if (!props.visible || props.tempToken !== submittedToken) return
    ElMessage.error(e.message || '验证失败')
  } finally {
    if (props.tempToken === submittedToken) loading.value = false
  }
}
</script>

<template>
  <el-dialog :model-value="visible" title="输入验证码" width="400px" :close-on-click-modal="false" @close="emit('update:visible', false)">
    <div v-if="!useBackup">
      <el-input v-model="code" placeholder="输入 6 位验证码" maxlength="8" inputmode="numeric" autocomplete="one-time-code" />
      <p style="margin-top: 12px; text-align: center">
        <a href="#" style="color: #2563eb; font-size: 14px" @click.prevent="useBackup = true">使用备份码登录</a>
      </p>
    </div>
    <div v-else>
      <el-input v-model="backupCode" placeholder="输入备份码（XXXX-XXXX）" />
      <p style="margin-top: 12px; text-align: center">
        <a href="#" style="color: #2563eb; font-size: 14px" @click.prevent="useBackup = false">使用验证码登录</a>
      </p>
    </div>
    <template #footer>
      <el-button @click="emit('update:visible', false)">取消</el-button>
      <el-button type="primary" :loading="loading" @click="handleVerify">登录</el-button>
    </template>
  </el-dialog>
</template>
