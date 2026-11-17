<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import {
  updateUser,
  resetUserPassword,
  updateUserPhone,
  saveUserIPWhitelist,
  setUserRealnameStatus,
  listUserLoginDevices,
  type UpdateUserParams,
  type ResetPasswordParams,
  type UserDetail,
  type UserLoginDeviceItem,
} from '@/api/user'

const props = defineProps<{
  visible: boolean
  user: UserDetail | null
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  'success': []
}>()

const activeTab = ref('basic')
const loading = ref(false)

const basicForm = ref<UpdateUserParams>({})
const passwordForm = ref<ResetPasswordParams>({
  mode: 'default',
  must_change: true
})
const phoneForm = ref({ new_phone: '' })
const whitelistForm = ref({
  enabled: false,
  ip: '',
})
const realnameStatus = ref({
  verified: false,
  real_name: '',
  id_number_masked: '',
})
const loginDevices = ref<UserLoginDeviceItem[]>([])
const loginDevicesLoading = ref(false)

const statusNotice = computed(() => {
  if (!props.user) return null
  if (props.user.login_reason === 'owner_banned') {
    return {
      tone: 'warning',
      title: '当前无法登录',
      text: '原因：归属助理已禁用。请先解封归属助理后，再恢复该用户使用。',
    }
  }
  if (props.user.status === 'banned') {
    return {
      tone: 'danger',
      title: '当前无法登录',
      text: '原因：该用户已被封禁。',
    }
  }
  if (props.user.status === 'deleted') {
    return {
      tone: 'danger',
      title: '账号已注销',
      text: '该用户账号已注销，当前不可恢复登录。',
    }
  }
  return null
})

watch(() => props.visible, (val) => {
  if (val && props.user) {
    const nextBasicForm: UpdateUserParams = {
      wanan_id: props.user.wanan_id || '',
      nickname: props.user.nickname,
      user_tag: props.user.user_tag || '',
      avatar_url: props.user.avatar_url,
      self_signature: props.user.self_signature,
      gender: props.user.gender,
      birthday: props.user.birthday || undefined,
      can_create_group: props.user.can_create_group,
    }
    basicForm.value = nextBasicForm
    phoneForm.value.new_phone = props.user.phone
    whitelistForm.value = {
      enabled: props.user.ip_whitelist_enabled,
      ip: props.user.ip_whitelist_ip || '',
    }
    passwordForm.value = { mode: 'default', must_change: true }
    realnameStatus.value = {
      verified: !!props.user.real_name_verified,
      real_name: props.user.real_name || '',
      id_number_masked: props.user.id_number_masked || '',
    }
    loginDevices.value = []
    activeTab.value = 'basic'
    void loadLoginDevices()
  }
})

const loadLoginDevices = async () => {
  if (!props.user) return
  loginDevicesLoading.value = true
  try {
    const res = await listUserLoginDevices(props.user.id)
    loginDevices.value = res.data.data || []
  } catch {
    loginDevices.value = []
  } finally {
    loginDevicesLoading.value = false
  }
}

watch(
  () => whitelistForm.value.enabled,
  (enabled) => {
    if (
      enabled &&
      !whitelistForm.value.ip.trim() &&
      props.user?.current_login_ip
    ) {
      whitelistForm.value.ip = props.user.current_login_ip.trim()
    }
  },
)

const handleClose = () => {
  emit('update:visible', false)
}

const handleToggleRealname = async (verified: boolean) => {
  if (!props.user) return
  loading.value = true
  try {
    await setUserRealnameStatus(props.user.id, verified)
    ElMessage.success(verified ? '已标记为已实名' : '已标记为未实名')
    if (verified) {
      realnameStatus.value = {
        verified: true,
        real_name: '李**',
        id_number_masked: '510*************19',
      }
    } else {
      realnameStatus.value = { verified: false, real_name: '', id_number_masked: '' }
    }
    emit('success')
  } catch (error: any) {
    ElMessage.error(error.response?.data?.message || error.message || '操作失败')
  } finally {
    loading.value = false
  }
}

const handleSaveBasic = async () => {
  if (!props.user) return
  loading.value = true
  try {
    await updateUser(props.user.id, basicForm.value)
    ElMessage.success('更新成功')
    emit('success')
    handleClose()
  } catch (error: any) {
    ElMessage.error(error.response?.data?.message || error.message || '更新失败')
  } finally {
    loading.value = false
  }
}

const handleResetPassword = async () => {
  if (!props.user) return
  if (passwordForm.value.mode === 'set' && !passwordForm.value.new_password) {
    ElMessage.warning('请输入新密码')
    return
  }
  loading.value = true
  try {
    await resetUserPassword(props.user.id, passwordForm.value)
    ElMessage.success('密码重置成功')
    emit('success')
    handleClose()
  } catch (error: any) {
    ElMessage.error(error.response?.data?.message || '重置失败')
  } finally {
    loading.value = false
  }
}

const handleUpdatePhone = async () => {
  if (!props.user) return
  if (!phoneForm.value.new_phone) {
    ElMessage.warning('请输入新手机号')
    return
  }
  loading.value = true
  try {
    await updateUserPhone(props.user.id, phoneForm.value.new_phone)
    ElMessage.success('手机号更新成功')
    emit('success')
    handleClose()
  } catch (error: any) {
    ElMessage.error(error.response?.data?.message || '更新失败')
  } finally {
    loading.value = false
  }
}

const handleSaveWhitelist = async () => {
  if (!props.user) return
  if (whitelistForm.value.enabled && !whitelistForm.value.ip.trim()) {
    ElMessage.warning('启用白名单后请填写至少 1 个 IP 地址')
    return
  }
  loading.value = true
  try {
    const res = await saveUserIPWhitelist(props.user.id, {
      enabled: whitelistForm.value.enabled,
      ip: whitelistForm.value.ip.trim() || undefined,
    })
    const result = res.data.data
    if (result?.forced_logout) {
      ElMessage.success('白名单已保存，用户当前登录 IP 不匹配，已被强制下线')
    } else {
      ElMessage.success('登录白名单已保存')
    }
    emit('success')
    handleClose()
  } catch (error: any) {
    ElMessage.error(error.response?.data?.message || '保存失败')
  } finally {
    loading.value = false
  }
}

</script>

<template>
  <el-dialog
    :model-value="visible"
    title="编辑用户"
    width="800px"
    @close="handleClose"
  >
    <div
      v-if="statusNotice"
      class="status-notice"
      :class="statusNotice.tone === 'danger' ? 'status-notice--danger' : 'status-notice--warning'"
    >
      <div class="status-notice-title">{{ statusNotice.title }}</div>
      <div class="status-notice-text">{{ statusNotice.text }}</div>
    </div>

    <el-tabs v-model="activeTab">
      <el-tab-pane label="基本资料" name="basic">
        <el-form :model="basicForm" label-width="100px">
          <el-form-item label="IM用户ID">
            <span class="readonly-value">{{ props.user?.im_user_id || '—' }}</span>
          </el-form-item>
          <el-form-item label="当前登录IP">
            <span class="readonly-value">{{ props.user?.current_login_ip || '未获取' }}</span>
          </el-form-item>
          <el-form-item label="信号">
            <el-input
              v-model.trim="basicForm.wanan_id"
              maxlength="20"
              placeholder="留空可清空；6-20 位，支持纯数字"
            />
          </el-form-item>
          <el-form-item label="昵称">
            <el-input v-model="basicForm.nickname" />
          </el-form-item>
          <el-form-item label="用户标签">
            <el-input
              v-model.trim="basicForm.user_tag"
              maxlength="12"
              placeholder="如：客服、专员；留空则清空"
            />
          </el-form-item>
          <el-form-item label="头像URL">
            <el-input v-model="basicForm.avatar_url" />
          </el-form-item>
          <el-form-item label="个性签名">
            <el-input v-model="basicForm.self_signature" type="textarea" :rows="3" />
          </el-form-item>
          <el-form-item label="性别">
            <el-radio-group v-model="basicForm.gender">
              <el-radio :label="0">未知</el-radio>
              <el-radio :label="1">男</el-radio>
              <el-radio :label="2">女</el-radio>
            </el-radio-group>
          </el-form-item>
          <el-form-item label="生日">
            <el-date-picker v-model="basicForm.birthday" type="date" value-format="YYYY-MM-DD" />
          </el-form-item>
          <el-form-item label="允许建群">
            <el-switch
              v-model="basicForm.can_create_group"
              inline-prompt
              active-text="允许"
              inactive-text="禁止"
            />
          </el-form-item>
        </el-form>
        <div style="text-align: right">
          <el-button @click="handleClose">取消</el-button>
          <el-button type="primary" :loading="loading" @click="handleSaveBasic">保存</el-button>
        </div>
      </el-tab-pane>

      <el-tab-pane label="重置密码" name="password">
        <el-form :model="passwordForm" label-width="100px">
          <el-form-item label="重置方式">
            <el-radio-group v-model="passwordForm.mode">
              <el-radio label="default">重置为默认密码 (123456)</el-radio>
              <el-radio label="set">设置指定密码</el-radio>
            </el-radio-group>
          </el-form-item>
          <el-form-item v-if="passwordForm.mode === 'set'" label="新密码">
            <el-input v-model="passwordForm.new_password" type="password" show-password />
          </el-form-item>
          <el-form-item label="强制修改">
            <el-checkbox v-model="passwordForm.must_change">下次登录时强制修改密码</el-checkbox>
          </el-form-item>
        </el-form>
        <div style="text-align: right">
          <el-button @click="handleClose">取消</el-button>
          <el-button type="primary" :loading="loading" @click="handleResetPassword">重置密码</el-button>
        </div>
      </el-tab-pane>

      <el-tab-pane label="修改手机号" name="phone">
        <el-form :model="phoneForm" label-width="100px">
          <el-form-item label="新手机号">
            <el-input v-model="phoneForm.new_phone" />
          </el-form-item>
        </el-form>
        <div style="text-align: right">
          <el-button @click="handleClose">取消</el-button>
          <el-button type="primary" :loading="loading" @click="handleUpdatePhone">更新手机号</el-button>
        </div>
      </el-tab-pane>

      <el-tab-pane label="登录白名单" name="whitelist">
        <el-form :model="whitelistForm" label-width="120px">
          <el-form-item label="启用白名单">
            <el-switch v-model="whitelistForm.enabled" />
          </el-form-item>
          <el-form-item label="白名单 IP">
            <el-input
              v-model="whitelistForm.ip"
              type="textarea"
              :rows="4"
              placeholder="每行一个 IP 地址，支持填写多个"
              :disabled="!whitelistForm.enabled"
            />
          </el-form-item>
          <el-form-item label="当前登录 IP">
            <span class="readonly-value">{{ props.user?.current_login_ip || '未获取' }}</span>
          </el-form-item>
        </el-form>
        <div class="community-hint">
          启用时默认带出当前登录 IP，也可手动填写多个 IP。支持换行、逗号或分号分隔。保存后若用户当前活动登录 IP 与白名单均不匹配，系统会立即强制下线。
        </div>
        <div style="text-align: right">
          <el-button @click="handleClose">取消</el-button>
          <el-button type="primary" :loading="loading" @click="handleSaveWhitelist">保存白名单</el-button>
        </div>
      </el-tab-pane>

      <el-tab-pane label="实名认证" name="realname">
        <el-form label-width="120px">
          <el-form-item label="实名状态">
            <el-switch
              :model-value="realnameStatus.verified"
              inline-prompt
              active-text="已实名"
              inactive-text="未实名"
              :loading="loading"
              @change="(val: string | number | boolean) => handleToggleRealname(Boolean(val))"
            />
          </el-form-item>
          <el-form-item v-if="realnameStatus.verified" label="姓名">
            <span class="readonly-value">{{ realnameStatus.real_name || '—' }}</span>
          </el-form-item>
          <el-form-item v-if="realnameStatus.verified" label="证件号">
            <span class="readonly-value">{{ realnameStatus.id_number_masked || '—' }}</span>
          </el-form-item>
        </el-form>
        <div class="community-hint">
          归属到助理/管理员的用户会自动视为已实名；归属超管默认未实名。也可在此手动开关。展示为脱敏：李** / 510*************19（前 3 后 2）。
        </div>
        <div style="text-align: right; margin-top: 12px">
          <el-button @click="handleClose">关闭</el-button>
        </div>
      </el-tab-pane>

      <el-tab-pane label="历史登录设备" name="devices">
        <div class="community-hint" style="margin-bottom: 12px">
          含 App 与 Web 登录记录。Web 登录不会出现在 App「登录设备管理 / 安全提醒」中。
        </div>
        <el-table v-loading="loginDevicesLoading" :data="loginDevices" size="small" empty-text="暂无登录设备">
          <el-table-column label="设备" min-width="140">
            <template #default="{ row }">
              {{ row.device_name || '—' }}
              <el-tag v-if="row.is_web" size="small" type="warning" style="margin-left: 6px">Web</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="platform" label="平台" width="90" />
          <el-table-column prop="ip" label="IP" min-width="120" />
          <el-table-column label="状态" width="80">
            <template #default="{ row }">
              <el-tag :type="row.is_online ? 'success' : 'info'" size="small">
                {{ row.is_online ? '在线' : '离线' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="last_login_at" label="最近登录" min-width="160" />
        </el-table>
        <div style="text-align: right; margin-top: 12px">
          <el-button @click="loadLoginDevices" :loading="loginDevicesLoading">刷新</el-button>
          <el-button @click="handleClose">关闭</el-button>
        </div>
      </el-tab-pane>
    </el-tabs>
  </el-dialog>
</template>

<style scoped>
:deep(.el-dialog) {
  max-width: calc(100vw - 32px);
}

.status-notice {
  margin-bottom: 14px;
  padding: 12px 14px;
  border-radius: 10px;
  border: 1px solid transparent;
}

.status-notice--warning {
  background: #fff7e6;
  border-color: #ffd591;
}

.status-notice--danger {
  background: #fff2f0;
  border-color: #ffb3b3;
}

.status-notice-title {
  font-size: 13px;
  font-weight: 600;
  color: #262626;
}

.status-notice-text {
  margin-top: 4px;
  font-size: 12px;
  line-height: 1.6;
  color: #595959;
}

.readonly-value {
  font-family: 'Fira Code', 'Courier New', monospace;
  font-size: 13px;
  color: #475569;
  word-break: break-all;
}

.community-hint {
  margin: 6px 0 18px;
  padding: 12px 14px;
  border-radius: 10px;
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  color: #64748b;
  font-size: 12px;
  line-height: 1.7;
}
</style>
