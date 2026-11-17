<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { listOwnershipOperations, retryOwnershipOperation, type OwnershipOperation } from '@/api/ownership'
const rows = ref<OwnershipOperation[]>([]), total = ref(0), page = ref(1), state = ref(''), loading = ref(false), retrying = ref(0)
const labels: Record<string,string> = { pending: '待同步', migrated: '已划分', unchanged: '归属一致', blocked: '暂不可操作', conflict: '归属冲突', manual: '待人工处理' }
const reasons: Record<string,string> = { historical_owner_conflict: '历史好友涉及多个经营空间', cross_owner: '双方已属于不同空间', group_owner_conflict: '群归属存在独立转移或历史冲突', group_creator_ambiguous: '历史群创建者身份需要核对', owner_inactive: '归属人已停用', owner_missing: '归属人不存在', owner_role_unknown: '归属角色不可确认', friendship_not_confirmed: '尚未确认双向好友', friendship_lookup_unavailable: '好友关系暂无法核验', assignment_database_unavailable: '归属保存暂不可用', post_commit_effect_failed: '归属已提交，后续资料同步待重试' }
async function load() {
  loading.value = true
  try { const result = (await listOwnershipOperations(page.value,state.value)).data.data; rows.value = result?.list || []; total.value = result?.total || 0 }
  catch(e: any) { ElMessage.error(e.message || '读取归属记录失败') }
  finally { loading.value = false }
}
async function retry(row: OwnershipOperation) {
  retrying.value = row.id
  try { await retryOwnershipOperation(row.id); ElMessage.success('已安排重新核验，将按当前归属规则处理'); await load() }
  catch(e: any) { ElMessage.error(e.message || '重试未完成') }
  finally { retrying.value = 0 }
}
onMounted(load)
</script>
<template>
  <section class="ownership-page">
    <h2>好友归属处理</h2>
    <p>好友关系与归属同步分别处理。冲突记录需先核对业务依据；重新核验不会强制覆盖已有归属。</p>
    <el-select v-model="state" placeholder="全部状态" clearable @change="page=1;load()"><el-option v-for="(label,key) in labels" :key="key" :label="label" :value="key" /></el-select>
    <el-button @click="load">刷新</el-button>
    <el-table :data="rows" v-loading="loading">
      <el-table-column prop="operation_id" label="请求编号" min-width="190" show-overflow-tooltip />
      <el-table-column prop="caller" label="操作方" min-width="140" />
      <el-table-column prop="target" label="好友" min-width="140" />
      <el-table-column label="状态" width="150"><template #default="{row}">{{ labels[row.state] || row.state }}{{ row.needs_review ? ' · 需核对' : '' }}</template></el-table-column>
      <el-table-column label="原因" min-width="220"><template #default="{row}">{{ reasons[row.reason_code] || row.reason_code || '—' }}</template></el-table-column>
      <el-table-column prop="attempts" label="尝试次数" width="100" />
      <el-table-column label="操作" width="110"><template #default="{row}"><el-button v-if="['pending','blocked','conflict','manual'].includes(row.state) || row.effects_pending" :loading="retrying===row.id" @click="retry(row)">重新核验</el-button></template></el-table-column>
      <el-table-column type="expand"><template #default="{row}"><p>来源：{{ row.source }} · 创建：{{ row.created_at }}</p><p>人工重试 {{ row.manual_retries || 0 }} 次<span v-if="row.last_retried_by"> · 最近操作人 {{ row.last_retried_by }} · {{ row.last_retried_at }}</span></p><pre>{{ row.before }}</pre><pre>{{ row.after }}</pre><pre v-if="row.retry_history">{{ row.retry_history }}</pre></template></el-table-column>
    </el-table>
    <el-pagination v-model:current-page="page" :page-size="25" :total="total" layout="prev,pager,next" @current-change="load" />
  </section>
</template>
<style scoped>.ownership-page{padding:24px}.ownership-page p{color:#606266;line-height:1.6}.ownership-page pre{white-space:pre-wrap;overflow-wrap:anywhere;padding:12px;background:#f6f7f9}</style>
