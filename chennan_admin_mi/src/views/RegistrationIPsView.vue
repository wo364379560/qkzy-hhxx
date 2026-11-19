<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { listRegistrationIPs, changeRegistrationIP, type RegistrationIPEntry } from '@/api/registration-ip'
import { useAuthStore } from '@/stores/auth'
const auth = useAuthStore()
const items = ref<RegistrationIPEntry[]>([])
const loading = ref(false)
const saving = ref(false)
const showDialog = ref(false)
const loadError = ref('')
const form = reactive({ ip: '', note: '', current_password: '', remove: false })
async function load() {
  if (!auth.isSuperAdmin) return
  loading.value = true
  loadError.value = ''
  try { items.value = (await listRegistrationIPs()).data.data.items }
  catch (e: any) { loadError.value = e.message || '加载失败，请重试' }
  finally { loading.value = false }
}
function open(entry?: RegistrationIPEntry) {
  Object.assign(form, { ip: entry?.ip || '', note: entry?.note || '', current_password: '', remove: !!entry })
  showDialog.value = true
}
function clearPassword() { form.current_password = '' }
async function save() {
  if (saving.value || !auth.isSuperAdmin) return
  if (!form.ip.trim() || !form.current_password) { ElMessage.warning('请填写 IP 和当前管理员密码'); return }
  saving.value = true
  try {
    await changeRegistrationIP({ ...form, ip: form.ip.trim() })
    showDialog.value = false
    ElMessage.success(form.remove ? '已取消该 IP 的注册配额豁免' : '已添加注册 IP 放行')
    await load()
  } catch (e: any) { ElMessage.error(e.message || '操作失败') }
  finally { clearPassword(); saving.value = false }
}
onMounted(load)
</script>

<template>
  <el-card v-if="auth.isSuperAdmin" class="registration-ips">
    <template #header>
      <div class="heading"><h2>注册 IP 放行</h2><div><el-button :loading="loading" @click="load">刷新</el-button><el-button type="primary" @click="open()">添加 IP</el-button></div></div>
    </template>
    <el-alert title="普通 IP 每小时最多 50 次注册请求。放行仅豁免注册的 IP 小时配额，账号、设备、短信及其他防护仍生效。" type="info" :closable="false" />
    <p class="description">仅支持精确公网 IPv4 / IPv6；来源需经可信代理核实。此处不会修改用户的登录白名单。</p>
    <el-alert v-if="loadError" :title="loadError" type="error" :closable="false" />
    <el-table :data="items" v-loading="loading" empty-text="暂无注册 IP 放行规则">
      <el-table-column prop="ip" label="公网 IP" min-width="230" />
      <el-table-column prop="note" label="备注" min-width="160" />
      <el-table-column prop="updated_by" label="操作管理员 ID" width="140" />
      <el-table-column label="更新时间" min-width="180"><template #default="{ row }">{{ new Date(row.updated_at).toLocaleString() }}</template></el-table-column>
      <el-table-column label="操作" width="90"><template #default="{ row }"><el-button link type="danger" @click="open(row)">删除</el-button></template></el-table-column>
    </el-table>
    <el-dialog v-model="showDialog" :title="form.remove ? '确认删除注册 IP 放行' : '添加注册 IP 放行'" width="min(480px, 95vw)" :close-on-click-modal="false" :close-on-press-escape="!saving" :show-close="!saving" @closed="clearPassword">
      <el-form label-position="top" @submit.prevent="save">
        <el-form-item label="公网 IP"><el-input v-model="form.ip" :disabled="form.remove || saving" maxlength="64" placeholder="填写单个公网 IP，不支持网段" /></el-form-item>
        <el-form-item label="备注"><el-input v-model="form.note" :disabled="form.remove || saving" maxlength="200" show-word-limit /></el-form-item>
        <el-form-item label="当前管理员密码"><el-input v-model="form.current_password" type="password" autocomplete="current-password" :disabled="saving" maxlength="128" /></el-form-item>
      </el-form>
      <template #footer><el-button :disabled="saving" @click="showDialog = false">取消</el-button><el-button :type="form.remove ? 'danger' : 'primary'" :loading="saving" @click="save">{{ form.remove ? '确认删除' : '确认添加' }}</el-button></template>
    </el-dialog>
  </el-card>
  <el-result v-else icon="warning" title="仅超级管理员可操作" />
</template>
<style scoped>
.heading { display: flex; align-items: center; justify-content: space-between; gap: 12px; flex-wrap: wrap; }
h2 { font-size: 20px; margin: 0; }
.description { color: #606266; font-size: 14px; line-height: 1.6; }
</style>
