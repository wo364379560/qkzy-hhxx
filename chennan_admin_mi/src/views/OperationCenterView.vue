<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { getGovernanceOverview, type GovernanceOverview } from '@/api/operation'

const router = useRouter()
const loading = ref(false)
const overview = ref<GovernanceOverview>({
  userCount: 0,
  groupCount: 0,
  cleanupCandidateCount: 0,
  loginRiskCount: 0,
  ipBlockCount: 0,
  pendingReportCount: 0,
  whitelistCount: 0,
  manualBannedCount: 0,
  restoredCount: 0,
})

const metrics = computed(() => [
  { label: '用户数量', value: overview.value.userCount, tone: 'blue' },
  { label: '群组数量', value: overview.value.groupCount, tone: 'green' },
  { label: '清理候选', value: overview.value.cleanupCandidateCount, tone: overview.value.cleanupCandidateCount > 0 ? 'red' : 'slate' },
  { label: '登录风控', value: overview.value.loginRiskCount, tone: overview.value.loginRiskCount > 0 ? 'orange' : 'slate' },
  { label: 'IP 封禁', value: overview.value.ipBlockCount, tone: overview.value.ipBlockCount > 0 ? 'orange' : 'slate' },
  { label: '待处理举报', value: overview.value.pendingReportCount, tone: overview.value.pendingReportCount > 0 ? 'red' : 'slate' },
])

const modules = [
  { title: '清理风控', desc: '无邀请码普通注册号、白名单、封禁与恢复', path: '/operation/cleanup', status: '已接入' },
  { title: '服务器防护', desc: '账号风险、IP 风险、应用层 IP 封禁', path: '/operation/server-security', status: '已接入' },
  { title: '用户状态管理', desc: '封禁、恢复、踢下线和归属管理', path: '/users', status: '复用现有' },
  { title: '审计中心', desc: '后台访问审计、登录审计、敏感操作留痕', path: '/audits/access', status: '复用现有' },
  { title: '业务应急开关', desc: '短信限流、静默时段和全局设置', path: '/settings', status: '复用现有' },
  { title: '数据同步中心', desc: '用户、群组与腾讯 IM 同步入口', path: '/users', status: '复用现有' },
]

async function fetchOverview() {
  loading.value = true
  try {
    const res = await getGovernanceOverview()
    overview.value = res.data.data ?? overview.value
  } catch (error: any) {
    ElMessage.error(error.message || '加载运营治理中心失败')
  } finally {
    loading.value = false
  }
}

onMounted(fetchOverview)
</script>

<template>
  <div class="governance-page">
    <section class="hero">
      <div>
        <h1>运营治理中心</h1>
        <p>集中查看睿之禾的账号治理、安全风控、审计和同步状态。</p>
      </div>
      <div class="hero-actions">
        <el-button @click="fetchOverview">刷新</el-button>
        <el-button type="primary" @click="router.push('/operation/server-security')">服务器防护</el-button>
      </div>
    </section>

    <section v-loading="loading" class="metric-grid">
      <article v-for="item in metrics" :key="item.label" class="metric-card">
        <div class="metric-label">{{ item.label }}</div>
        <div class="metric-value" :class="`tone-${item.tone}`">{{ item.value }}</div>
      </article>
    </section>

    <section class="module-grid">
      <article v-for="item in modules" :key="item.title" class="module-card" @click="router.push(item.path)">
        <div>
          <div class="module-title">{{ item.title }}</div>
          <div class="module-desc">{{ item.desc }}</div>
        </div>
        <span class="module-status">{{ item.status }}</span>
      </article>
    </section>

    <section class="rule-strip">
      <span>仅超级管理员可见</span>
      <span>危险操作进入对应板块确认</span>
      <span>封禁、恢复、解除风控保留审计</span>
      <span>生产变更按主仓部署流程执行</span>
    </section>
  </div>
</template>

<style scoped>
.governance-page { display: flex; flex-direction: column; gap: 16px; }
.hero { display: flex; justify-content: space-between; gap: 16px; align-items: flex-start; padding: 4px 0 8px; }
.hero h1 { margin: 0 0 8px; font-size: 26px; color: #111827; }
.hero p { margin: 0; color: #667085; }
.hero-actions { display: flex; gap: 8px; flex-wrap: wrap; justify-content: flex-end; }
.metric-grid { display: grid; grid-template-columns: repeat(6, minmax(0, 1fr)); gap: 12px; }
.metric-card, .module-card { border: 1px solid #e6eaf2; background: #fff; border-radius: 8px; }
.metric-card { padding: 16px; min-height: 86px; }
.metric-label { color: #667085; font-weight: 600; font-size: 13px; }
.metric-value { margin-top: 10px; font-size: 28px; font-weight: 800; }
.tone-blue { color: #2563eb; }
.tone-green { color: #16a34a; }
.tone-orange { color: #d97706; }
.tone-red { color: #dc2626; }
.tone-slate { color: #475569; }
.module-grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 12px; }
.module-card { min-height: 116px; padding: 16px; cursor: pointer; display: flex; flex-direction: column; justify-content: space-between; transition: border-color .15s, box-shadow .15s; }
.module-card:hover { border-color: #93c5fd; box-shadow: 0 10px 24px rgba(15, 23, 42, .08); }
.module-title { color: #111827; font-size: 17px; font-weight: 800; }
.module-desc { margin-top: 8px; color: #667085; line-height: 1.6; }
.module-status { align-self: flex-start; color: #2563eb; background: #eff6ff; border: 1px solid #bfdbfe; border-radius: 999px; padding: 3px 10px; font-size: 12px; }
.rule-strip { display: flex; gap: 10px; flex-wrap: wrap; color: #667085; font-size: 13px; }
.rule-strip span { background: #f8fafc; border: 1px solid #e6eaf2; border-radius: 8px; padding: 8px 10px; }
@media (max-width: 1100px) {
  .metric-grid { grid-template-columns: repeat(3, minmax(0, 1fr)); }
  .module-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
}
@media (max-width: 760px) {
  .hero { flex-direction: column; }
  .metric-grid, .module-grid { grid-template-columns: 1fr; }
}
</style>
