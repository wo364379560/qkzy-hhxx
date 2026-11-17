<script setup lang="ts">
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import QrcodeVue from 'qrcode.vue'
import { bindTOTP } from '@/api/auth'
import type { AdminUser } from '@/types'

const props = defineProps<{
  visible: boolean
  totpUri: string
  totpSecret: string
  tempToken: string
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  'success': [data: { token: string; user: AdminUser; backupCodes: string[] }]
}>()

const code = ref('')
const loading = ref(false)

watch(() => [props.visible, props.tempToken], () => {
  code.value = ''
  loading.value = false
})

const handleBind = async () => {
  if (loading.value) return
  const submittedToken = props.tempToken
  code.value = code.value.replace(/\s/g, '')
  if (!/^\d{6}$/.test(code.value)) {
    ElMessage.warning('请输入 6 位验证码')
    return
  }
  loading.value = true
  try {
    const res = await bindTOTP(submittedToken, code.value)
    if (!props.visible || props.tempToken !== submittedToken) return
    emit('success', {
      token: res.data.data.token,
      user: res.data.data.user,
      backupCodes: res.data.data.backup_codes,
    })
    emit('update:visible', false)
  } catch (e: any) {
    if (!props.visible || props.tempToken !== submittedToken) return
    ElMessage.error(e.message || '绑定失败')
  } finally {
    if (props.tempToken === submittedToken) loading.value = false
  }
}
</script>

<template>
  <el-dialog :model-value="visible" title="绑定双因素认证" width="500px" :close-on-click-modal="false" @close="emit('update:visible', false)">
    <div style="text-align: center">
      <p style="margin-bottom: 16px; color: #64748b">请使用 Google Authenticator 或 Microsoft Authenticator 扫描二维码</p>
      <div style="display: inline-block; padding: 16px; background: white; border-radius: 8px; border: 1px solid #e2e8f0">
        <qrcode-vue :key="tempToken" :value="totpUri" :size="200" level="H" />
      </div>
      <p style="margin-top: 16px; font-size: 12px; color: #94a3b8">密钥（手动输入）：<code style="background: #f1f5f9; padding: 2px 6px; border-radius: 4px">{{ totpSecret }}</code></p>
      <p style="margin-top: 12px; color: #64748b; font-size: 12px">请使用本次扫码新增条目的验证码，并开启手机自动校准时间。</p>
      <el-input v-model="code" placeholder="输入 6 位验证码" maxlength="8" inputmode="numeric" autocomplete="one-time-code" style="margin-top: 24px" />
    </div>
    <template #footer>
      <el-button @click="emit('update:visible', false)">取消</el-button>
      <el-button type="primary" :loading="loading" @click="handleBind">完成绑定</el-button>
    </template>
  </el-dialog>
</template>
