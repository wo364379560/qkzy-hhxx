<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  blockIP,
  dismissQuietHoursGroupRisk,
  getIPAccessBlocks,
  getLoginRisks,
  getQuietHoursGroupRisks,
  getSecurityRiskSummary,
  resolveLoginRisk,
  resolveQuietHoursGroupRisk,
  syncQuietHoursGroupRisk,
  unblockIP,
  type IPAccessBlock,
  type LoginRisk,
  type QuietHoursGroupRisk,
  type SecurityRiskSummary,
} from '@/api/operation'
import { banUser, unbanUser } from '@/api/user'

type SecuritySectionKey = 'account' | 'pending-ip' | 'ip-management' | 'quiet-groups'
type SummaryCard = {
  id: string
  key: SecuritySectionKey
  label: string
  value: number
  desc: string
  accountStatus?: string
}

const IP_BLOCK_PAGE_SIZE = 100
const IP_RISK_PAGE_SIZE = 100
const loading = ref(false)
const accountRiskLoading = ref(false)
const activeSection = ref<SecuritySectionKey>('account')
const activeSummaryCard = ref('account-pending')
const accountRisks = ref<LoginRisk[]>([])
const ipRisks = ref<LoginRisk[]>([])
const ipBlocks = ref<IPAccessBlock[]>([])
const accountRiskTotal = ref(0)
const ipRiskTotal = ref(0)
const ipBlockTotal = ref(0)
const riskSummary = ref<SecurityRiskSummary>({
  accountPending: 0,
  accountWatching: 0,
  accountCooldown: 0,
  accountAlerted: 0,
  ipOpen: 0,
  ipWatching: 0,
  ipAlerted: 0,
  ipBlocked: 0,
  manualBannedAccounts: 0,
  quietGroupPending: 0,
})
const searchedIPBlocks = ref<IPAccessBlock[]>([])
const groupRisks = ref<QuietHoursGroupRisk[]>([])
const groupRiskTotal = ref(0)
const blockForm = reactive({ ip: '', reason: '' })
const accountRiskFilters = reactive({
  keyword: '',
  status: 'pending',
  source: 'all',
  page: 1,
  pageSize: 20,
})
const groupRiskFilters = reactive({ keyword: '', status: 'open' })
const ipQueryLoading = ref(false)

const activeIPBlockMap = computed(() => {
  return new Map(
    ipBlocks.value
      .filter(item => item.status === 'active')
      .map(item => [item.ipAddress, item]),
  )
})
const blockedIPRows = computed(() => ipBlocks.value.filter(item => item.status === 'active'))
const pendingIPRisks = computed(() => ipRisks.value.filter(item => !isBlockedIPRisk(item)))
const summaryCards = computed<SummaryCard[]>(() => [
  { id: 'account-pending', key: 'account', label: '待处理账号风险', value: riskSummary.value.accountPending, desc: `冷却 ${riskSummary.value.accountCooldown} / 告警 ${riskSummary.value.accountAlerted}`, accountStatus: 'pending' },
  { id: 'account-watching', key: 'account', label: '观察中账号', value: riskSummary.value.accountWatching, desc: '仅记录风险，默认不进入待处理列表', accountStatus: 'watching' },
  { id: 'ip-open', key: 'pending-ip', label: 'IP 风险', value: riskSummary.value.ipOpen || ipRiskTotal.value, desc: `告警 ${riskSummary.value.ipAlerted} / 观察 ${riskSummary.value.ipWatching}` },
  { id: 'ip-blocked', key: 'ip-management', label: '已封禁 IP', value: ipBlockTotal.value, desc: '应用层封禁，可随时解封' },
  { id: 'manual-banned', key: 'account', label: '已封禁账号', value: riskSummary.value.manualBannedAccounts, desc: '用户管理/清理风控中的封禁账号' },
  { id: 'quiet-groups', key: 'quiet-groups', label: '违规群待处理', value: riskSummary.value.quietGroupPending || groupRiskTotal.value, desc: '宵禁或静默时段建群风险' },
])
const securitySections = computed(() => [
  { key: 'account' as const, label: '账号风险', count: accountRiskTotal.value },
  { key: 'pending-ip' as const, label: '待处理IP风险', count: riskSummary.value.ipOpen || ipRiskTotal.value },
  { key: 'ip-management' as const, label: '已封禁IP列表（IP封禁管理）', count: ipBlockTotal.value },
  { key: 'quiet-groups' as const, label: '宵禁违规群治理', count: riskSummary.value.quietGroupPending || groupRiskTotal.value },
])

async function fetchAll() {
  loading.value = true
  try {
    const [summary, accounts, ips, blocks, quietGroups] = await Promise.all([
      getSecurityRiskSummary(),
      fetchAccountRisks(false),
      fetchOpenIPRisks(),
      fetchActiveIPBlocks(),
      getQuietHoursGroupRisks({ page: 1, pageSize: 20, status: 'open' }),
    ])
    accountRisks.value = accounts.data.data?.list ?? []
    ipRisks.value = ips.list
    ipBlocks.value = blocks.list
    accountRiskTotal.value = accounts.data.data?.total ?? 0
    ipRiskTotal.value = ips.total
    ipBlockTotal.value = blocks.total
    riskSummary.value = summary.data.data ?? riskSummary.value
    groupRisks.value = quietGroups.data.data?.list ?? []
    groupRiskTotal.value = quietGroups.data.data?.total ?? 0
  } catch (error: any) {
    ElMessage.error(error.message || '加载服务器防护失败')
  } finally {
    loading.value = false
  }
}

function accountRiskQueryParams() {
  return {
    page: accountRiskFilters.page,
    pageSize: accountRiskFilters.pageSize,
    scope: 'account' as const,
    status: accountRiskFilters.status || 'open',
    source: accountRiskFilters.source,
    keyword: accountRiskFilters.keyword.trim() || undefined,
  }
}

async function fetchAccountRisks(showLoading = true) {
  if (showLoading) accountRiskLoading.value = true
  try {
    const res = await getLoginRisks(accountRiskQueryParams())
    accountRisks.value = res.data.data?.list ?? []
    accountRiskTotal.value = res.data.data?.total ?? 0
    return res
  } catch (error: any) {
    ElMessage.error(error.message || '加载账号风险失败')
    throw error
  } finally {
    if (showLoading) accountRiskLoading.value = false
  }
}

async function searchAccountRisks() {
  accountRiskFilters.page = 1
  await fetchAccountRisks()
}

async function resetAccountRiskFilters() {
  accountRiskFilters.keyword = ''
  accountRiskFilters.status = 'pending'
  accountRiskFilters.source = 'all'
  accountRiskFilters.page = 1
  accountRiskFilters.pageSize = 20
  await fetchAccountRisks()
}

async function handleAccountRiskPageChange(page: number) {
  accountRiskFilters.page = page
  await fetchAccountRisks()
}

async function handleAccountRiskPageSizeChange(pageSize: number) {
  accountRiskFilters.pageSize = pageSize
  accountRiskFilters.page = 1
  await fetchAccountRisks()
}

async function selectSummaryCard(item: SummaryCard) {
  activeSummaryCard.value = item.id
  activeSection.value = item.key
  if (item.accountStatus) {
    accountRiskFilters.status = item.accountStatus
    accountRiskFilters.page = 1
    await fetchAccountRisks()
  }
}

function selectSection(section: SecuritySectionKey) {
  activeSection.value = section
  const firstCard = summaryCards.value.find(item => item.key === section)
  if (firstCard) activeSummaryCard.value = firstCard.id
}

async function fetchOpenIPRisks() {
  const first = await getLoginRisks({ page: 1, pageSize: IP_RISK_PAGE_SIZE, scope: 'ip', status: 'open' })
  const firstPage = first.data.data
  return { list: firstPage?.list ?? [], total: firstPage?.total ?? 0 }
}

async function fetchActiveIPBlocks() {
  const first = await getIPAccessBlocks({ page: 1, pageSize: IP_BLOCK_PAGE_SIZE, status: 'active' })
  const firstPage = first.data.data
  return { list: firstPage?.list ?? [], total: firstPage?.total ?? 0 }
}

async function fetchGroupRisks() {
  try {
    const res = await getQuietHoursGroupRisks({
      page: 1,
      pageSize: 50,
      keyword: groupRiskFilters.keyword || undefined,
      status: groupRiskFilters.status || undefined,
    })
    groupRisks.value = res.data.data?.list ?? []
    groupRiskTotal.value = res.data.data?.total ?? 0
  } catch (error: any) {
    ElMessage.error(error.message || '加载宵禁违规群失败')
  }
}

async function refreshIPBlocks() {
  const blocks = await fetchActiveIPBlocks()
  ipBlocks.value = blocks.list
  ipBlockTotal.value = blocks.total
}

async function promptReason(title: string) {
  const res = await ElMessageBox.prompt('请输入处理原因', title, {
    confirmButtonText: '确认',
    cancelButtonText: '取消',
    inputType: 'textarea',
  })
  return String(res.value || '')
}

function isCancelAction(error: any) {
  return error === 'cancel' || error?.action === 'cancel' || error?.action === 'close'
}

function isBlockedIPRisk(row: LoginRisk) {
  return row.status === 'blocked' || activeIPBlockMap.value.has(row.ipAddress)
}

async function resolveRisk(row: LoginRisk) {
  try {
    const reason = await promptReason(row.scope === 'account' && row.status === 'cooldown' ? '解除账号冷却' : '解除风控')
    await resolveLoginRisk(row.id, reason)
    ElMessage.success(row.scope === 'account' && row.status === 'cooldown' ? '已解除账号冷却' : '已解除风控')
    await fetchAll()
  } catch (error: any) {
    if (!isCancelAction(error)) ElMessage.error(error.message || '操作失败')
  }
}

async function handleBanUser(row: LoginRisk) {
  if (!row.userId) {
    ElMessage.warning('该风险未绑定本地用户')
    return
  }
  try {
    await ElMessageBox.confirm('封禁后会影响该用户登录，可在用户管理或清理风控中恢复。', '封禁账号', { type: 'warning' })
    await banUser(row.userId)
    ElMessage.success('已封禁账号')
    await fetchAll()
  } catch (error: any) {
    if (!isCancelAction(error)) ElMessage.error(error.message || '操作失败')
  }
}

async function handleUnbanUser(row: LoginRisk) {
  if (!row.userId) {
    ElMessage.warning('该风险未绑定本地用户')
    return
  }
  try {
    await unbanUser(row.userId)
    ElMessage.success('已恢复账号')
    await fetchAll()
  } catch (error: any) {
    ElMessage.error(error.message || '操作失败')
  }
}

function accountAction(row: LoginRisk) {
  if (isAdminAccountRisk(row)) {
    return { label: '后台账号', type: 'info' as const, action: () => ElMessage.info('后台账号请用“解除冷却/解除风控”处理') }
  }
  if (row.userStatus === 'banned') {
    return { label: '恢复账号', type: 'success' as const, action: () => handleUnbanUser(row) }
  }
  return { label: '封禁账号', type: 'danger' as const, action: () => handleBanUser(row) }
}

async function queryIPStatus() {
  const targetIP = blockForm.ip.trim()
  if (!targetIP) {
    ElMessage.warning('请填写 IP')
    return
  }
  ipQueryLoading.value = true
  try {
    const res = await getIPAccessBlocks({ page: 1, pageSize: 100, ipAddress: targetIP })
    searchedIPBlocks.value = res.data.data?.list ?? []
    if (searchedIPBlocks.value.length === 0) {
      ElMessage.info('该 IP 暂无封禁记录')
    }
  } catch (error: any) {
    ElMessage.error(error.message || '查询失败')
  } finally {
    ipQueryLoading.value = false
  }
}

async function submitBlockIP(ip?: string) {
  const targetIP = (ip || blockForm.ip).trim()
  if (!targetIP) {
    ElMessage.warning('请填写 IP')
    return
  }
  try {
    const reason = (ip ? await promptReason('封禁 IP') : blockForm.reason).trim() || '后台手动封禁'
    await blockIP(targetIP, reason)
    const shouldQuery = !ip
    blockForm.ip = ''
    blockForm.reason = ''
    ElMessage.success('已封禁 IP')
    await fetchAll()
    if (shouldQuery) {
      blockForm.ip = targetIP
      await queryIPStatus()
    }
  } catch (error: any) {
    if (!isCancelAction(error)) ElMessage.error(error.message || '操作失败')
  }
}

async function removeBlock(row: IPAccessBlock) {
  try {
    const reason = await promptReason('解封 IP')
    await unblockIP(row.id, reason)
    ElMessage.success('已解封 IP')
    await refreshIPBlocks()
    if (blockForm.ip.trim()) await queryIPStatus()
  } catch (error: any) {
    if (!isCancelAction(error)) ElMessage.error(error.message || '操作失败')
  }
}

function scopeObject(row: LoginRisk) {
  if (row.scope === 'ip') return row.ipAddress
  if (isAdminAccountRisk(row)) return row.username || row.riskKey.replace(/^admin:/, '')
  return row.username || row.imUserID || row.riskKey
}

function scopeSubObject(row: LoginRisk) {
  if (isAdminAccountRisk(row)) return `后台账号 / ${row.imUserID || '未知角色'}`
  return row.imUserID || '-'
}

function isAdminAccountRisk(row: LoginRisk) {
  return row.scope === 'account' && row.riskKey.startsWith('admin:')
}

function accountRiskSourceLabel(row: LoginRisk) {
  return isAdminAccountRisk(row) ? '后台账号' : '用户账号'
}

function riskStatusLabel(status: string) {
  switch (status) {
    case 'cooldown': return '冷却中'
    case 'watching': return '观察中'
    case 'alerted': return '已告警'
    case 'blocked': return '已自动封禁'
    case 'review_pending': return '待复核'
    case 'resolved': return '已解除'
    default: return status || '未知'
  }
}

function ipBlockStatusLabel(status: string) {
  switch (status) {
    case 'active': return '已封禁'
    case 'removed': return '已解封'
    default: return status || '未知'
  }
}

function groupRiskStatusLabel(status: string) {
  switch (status) {
    case 'pending': return '待处理'
    case 'auto_dismissed': return '已自动解散'
    case 'manual_dismissed': return '已手动解散'
    case 'resolved': return '已解除标记'
    case 'protected': return '已保护'
    case 'risk_only': return '只记录风险'
    case 'api_failed': return '接口失败'
    default: return status || '未知'
  }
}

function groupRiskSourceLabel(source: string) {
  switch (source) {
    case 'im_after_create_group': return '建群后回调'
    case 'im_before_send_group_msg': return '发消息前兜底'
    default: return source || '未知'
  }
}

async function clearGroupRisk(row: QuietHoursGroupRisk) {
  try {
    const reason = await promptReason('解除违规群风险标记')
    await resolveQuietHoursGroupRisk(row.id, reason)
    ElMessage.success('已解除风险标记')
    await fetchGroupRisks()
  } catch (error: any) {
    if (!isCancelAction(error)) ElMessage.error(error.message || '操作失败')
  }
}

async function syncGroupRisk(row: QuietHoursGroupRisk) {
  try {
    await syncQuietHoursGroupRisk(row.id)
    ElMessage.success('已补同步群资料')
    await fetchGroupRisks()
  } catch (error: any) {
    ElMessage.error(error.message || '补同步失败')
  }
}

async function dismissGroupRisk(row: QuietHoursGroupRisk) {
  try {
    await ElMessageBox.confirm(
      `确认解散群 ${row.groupName || row.imGroupId}？该操作会调用腾讯 IM 解散单个群，不能批量执行。`,
      '手动解散违规群',
      { type: 'warning', confirmButtonText: '确认解散', cancelButtonText: '取消' },
    )
    const reason = await promptReason('手动解散违规群')
    await dismissQuietHoursGroupRisk(row.id, reason)
    ElMessage.success('已手动解散群')
    await fetchGroupRisks()
  } catch (error: any) {
    if (!isCancelAction(error)) ElMessage.error(error.message || '操作失败')
  }
}

onMounted(fetchAll)
</script>

<template>
  <div class="security-page">
    <div class="page-head">
      <div>
        <h1>服务器防护</h1>
        <p>账号风险和 IP 风险分开处理；IP 封禁只做应用层控制，不直接修改防火墙。</p>
      </div>
      <el-button @click="fetchAll">刷新</el-button>
    </div>

    <section v-loading="loading" class="overview-panel">
      <div class="section-heading">
        <div>
          <h2>服务器防护信息</h2>
          <p>汇总当前账号、IP、账号封禁和违规群治理状态。</p>
        </div>
      </div>
      <div class="overview-grid">
        <button
          v-for="item in summaryCards"
          :key="item.id"
          class="overview-card"
          type="button"
          :class="{ active: activeSummaryCard === item.id }"
          @click="selectSummaryCard(item)"
        >
          <span>{{ item.label }}</span>
          <strong>{{ item.value }}</strong>
          <small>{{ item.desc }}</small>
        </button>
      </div>
    </section>

    <nav class="section-nav" aria-label="服务器防护板块">
      <button
        v-for="item in securitySections"
        :key="item.key"
        type="button"
        :class="{ active: activeSection === item.key }"
        @click="selectSection(item.key)"
      >
        <span>{{ item.label }}</span>
        <em>{{ item.count }}</em>
      </button>
    </nav>

    <section v-show="activeSection === 'account'" class="panel">
      <div class="panel-head">
        <div>
          <h2>账号风险</h2>
          <p class="muted">默认展示冷却中和已告警账号；观察中账号可通过状态筛选查看。</p>
        </div>
        <div class="account-risk-filters">
          <el-input
            v-model="accountRiskFilters.keyword"
            placeholder="账号 / IMUserID / IP / user_id"
            clearable
            @keyup.enter="searchAccountRisks"
          />
          <el-select v-model="accountRiskFilters.status" placeholder="状态" @change="searchAccountRisks">
            <el-option label="待处理" value="pending" />
            <el-option label="冷却中" value="cooldown" />
            <el-option label="已告警" value="alerted" />
            <el-option label="观察中" value="watching" />
            <el-option label="全部未解除" value="open" />
            <el-option label="已解除" value="resolved" />
          </el-select>
          <el-select v-model="accountRiskFilters.source" placeholder="来源" @change="searchAccountRisks">
            <el-option label="全部来源" value="all" />
            <el-option label="用户账号" value="user" />
            <el-option label="后台账号" value="admin" />
          </el-select>
          <el-button type="primary" @click="searchAccountRisks">筛选</el-button>
          <el-button @click="resetAccountRiskFilters">重置</el-button>
        </div>
      </div>
      <el-table v-loading="accountRiskLoading" :data="accountRisks" stripe size="small">
        <el-table-column label="风险等级" width="100" prop="level" />
        <el-table-column label="来源" width="100">
          <template #default="{ row }">
            <span class="status-badge">{{ accountRiskSourceLabel(row) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="账号 / 类型" min-width="180">
          <template #default="{ row }">
            <div class="strong">{{ scopeObject(row) }}</div>
            <div class="muted">{{ scopeSubObject(row) }}</div>
          </template>
        </el-table-column>
        <el-table-column label="原因" min-width="220" prop="reason" />
        <el-table-column label="失败次数" width="96" prop="failureCount" />
        <el-table-column label="状态 / 冷却至" min-width="170">
          <template #default="{ row }">
            <div class="strong">{{ riskStatusLabel(row.status) }}</div>
            <div class="muted">{{ row.cooldownUntil || '未冷却' }}</div>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="280" fixed="right">
          <template #default="{ row }">
            <el-button
              size="small"
              :type="accountAction(row).type"
              plain
              @click="accountAction(row).action()"
            >
              {{ accountAction(row).label }}
            </el-button>
            <el-button size="small" @click="resolveRisk(row)">
              {{ row.status === 'cooldown' ? '解除冷却' : '解除风控' }}
            </el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="table-pagination">
        <el-pagination
          v-model:current-page="accountRiskFilters.page"
          v-model:page-size="accountRiskFilters.pageSize"
          :page-sizes="[20, 50, 100]"
          :total="accountRiskTotal"
          layout="total, sizes, prev, pager, next, jumper"
          background
          @current-change="handleAccountRiskPageChange"
          @size-change="handleAccountRiskPageSizeChange"
        />
      </div>
    </section>

    <section v-show="activeSection === 'pending-ip'" class="panel">
      <div class="panel-title-row">
        <h2>待处理 IP 风险</h2>
        <span class="muted">当前展示前 {{ IP_RISK_PAGE_SIZE }} 条；更多风险可通过刷新或后续分页处理。</span>
      </div>
      <el-table :data="pendingIPRisks" stripe size="small" empty-text="暂无待处理 IP 风险">
        <el-table-column label="风险等级" width="100" prop="level" />
        <el-table-column label="IP" min-width="150" prop="ipAddress" />
        <el-table-column label="原因" min-width="220" prop="reason" />
        <el-table-column label="失败次数" width="96" prop="failureCount" />
        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{ row }">
            <el-button size="small" type="danger" plain @click="submitBlockIP(row.ipAddress)">封禁 IP</el-button>
            <el-button size="small" @click="resolveRisk(row)">解除风控</el-button>
          </template>
        </el-table-column>
      </el-table>
    </section>

    <section v-show="activeSection === 'ip-management'" class="panel">
      <div class="panel-head">
        <div>
          <h2>已封禁 IP 列表（IP 封禁管理）</h2>
          <p class="muted">查看当前已封禁 IP、手动封禁新 IP，或查询单个 IP 历史记录。</p>
        </div>
        <div class="block-form">
          <el-input v-model="blockForm.ip" placeholder="IP 地址" />
          <el-input v-model="blockForm.reason" placeholder="封禁原因" />
          <el-button :loading="ipQueryLoading" @click="queryIPStatus">查询</el-button>
          <el-button type="primary" @click="submitBlockIP()">封禁</el-button>
        </div>
      </div>

      <div class="table-block">
        <h3>当前已封禁 IP</h3>
        <el-table :data="blockedIPRows" stripe size="small" empty-text="暂无已封禁 IP">
          <el-table-column label="IP" min-width="150" prop="ipAddress" />
          <el-table-column label="状态" width="110">
            <template #default="{ row }">
              <span class="status-badge danger">{{ ipBlockStatusLabel(row.status) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="触发账号" min-width="130">
            <template #default="{ row }">{{ row.triggerUsername || '—' }}</template>
          </el-table-column>
          <el-table-column label="失败次数" width="100">
            <template #default="{ row }">
              {{ row.triggerFailCount ? `${row.triggerFailCount}次` : '—' }}
            </template>
          </el-table-column>
          <el-table-column label="封禁原因" min-width="280" prop="reason" />
          <el-table-column label="创建时间" min-width="180" prop="createdAt" />
          <el-table-column label="更新时间" min-width="180" prop="updatedAt" />
          <el-table-column label="操作" width="120" fixed="right">
            <template #default="{ row }">
              <el-button size="small" @click="removeBlock(row)">解封 IP</el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>

      <div v-if="searchedIPBlocks.length > 0" class="sub-panel">
        <h3>该 IP 历史封禁记录</h3>
        <el-table :data="searchedIPBlocks" stripe size="small">
          <el-table-column label="IP" min-width="150" prop="ipAddress" />
          <el-table-column label="状态" width="100" prop="status" />
          <el-table-column label="触发账号" min-width="130">
            <template #default="{ row }">{{ row.triggerUsername || '—' }}</template>
          </el-table-column>
          <el-table-column label="失败次数" width="100">
            <template #default="{ row }">
              {{ row.triggerFailCount ? `${row.triggerFailCount}次` : '—' }}
            </template>
          </el-table-column>
          <el-table-column label="原因" min-width="220" prop="reason" />
          <el-table-column label="更新时间" min-width="180" prop="updatedAt" />
          <el-table-column label="操作" width="120">
            <template #default="{ row }">
              <el-button v-if="row.status === 'active'" size="small" @click="removeBlock(row)">解封</el-button>
              <span v-else class="muted">已解封</span>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </section>

    <section v-show="activeSection === 'quiet-groups'" class="panel">
      <div class="panel-head">
        <div>
          <h2>宵禁违规群治理</h2>
          <p class="muted">治理静默时段创建或触发兜底识别的群风险。</p>
        </div>
        <div class="group-risk-filters">
          <el-input v-model="groupRiskFilters.keyword" placeholder="群 ID / 群名 / owner / 原因" clearable />
          <el-select v-model="groupRiskFilters.status" placeholder="状态" clearable>
            <el-option label="待处理" value="open" />
            <el-option label="已自动解散" value="auto_dismissed" />
            <el-option label="已手动解散" value="manual_dismissed" />
            <el-option label="只记录风险" value="risk_only" />
            <el-option label="接口失败" value="api_failed" />
            <el-option label="已解除标记" value="resolved" />
          </el-select>
          <el-button @click="fetchGroupRisks">筛选</el-button>
        </div>
      </div>
      <el-table :data="groupRisks" stripe size="small">
        <el-table-column label="群" min-width="220">
          <template #default="{ row }">
            <div class="strong">{{ row.groupName || row.imGroupId }}</div>
            <div class="muted">{{ row.imGroupId }}</div>
          </template>
        </el-table-column>
        <el-table-column label="owner / 归属" min-width="180">
          <template #default="{ row }">
            <div class="strong">{{ row.ownerImUserId || '—' }}</div>
            <div class="muted">{{ row.ownerName || row.ownerId || '—' }}</div>
          </template>
        </el-table-column>
        <el-table-column label="真实创建时间" min-width="170" prop="tencentCreateTime" />
        <el-table-column label="来源" width="130">
          <template #default="{ row }">{{ groupRiskSourceLabel(row.triggerSource) }}</template>
        </el-table-column>
        <el-table-column label="状态" width="120">
          <template #default="{ row }">
            <span class="status-badge">{{ groupRiskStatusLabel(row.status) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="原因" min-width="260" prop="reason" />
        <el-table-column label="操作" width="280" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="syncGroupRisk(row)">补同步</el-button>
            <el-button size="small" @click="clearGroupRisk(row)">解标记</el-button>
            <el-button
              v-if="!['auto_dismissed', 'manual_dismissed'].includes(row.status)"
              size="small"
              type="danger"
              plain
              @click="dismissGroupRisk(row)"
            >解散</el-button>
          </template>
        </el-table-column>
      </el-table>
    </section>

    <section class="note-panel">
      <strong>备份与回滚</strong>
      <span>账号封禁可通过恢复账号回滚；IP 封禁可通过解封 IP 回滚；所有操作写入后台行为审计。</span>
    </section>
  </div>
</template>

<style scoped>
.security-page { display: flex; flex-direction: column; gap: 14px; }
.page-head, .panel-head { display: flex; justify-content: space-between; align-items: flex-start; gap: 12px; }
.page-head h1 { margin: 0 0 8px; font-size: 26px; color: #111827; }
.page-head p, .muted { color: #667085; }
.section-heading { display: flex; align-items: flex-start; justify-content: space-between; gap: 12px; margin-bottom: 12px; }
.section-heading h2, .panel h2 { margin: 0 0 6px; font-size: 18px; color: #111827; }
.section-heading p, .panel-head p { margin: 0; }
.panel-title-row { display: flex; align-items: center; justify-content: space-between; gap: 12px; margin-bottom: 12px; }
.overview-panel, .panel, .note-panel { border: 1px solid #e6eaf2; background: #fff; border-radius: 8px; }
.overview-panel { padding: 14px; }
.overview-grid { display: grid; grid-template-columns: repeat(5, minmax(0, 1fr)); gap: 10px; }
.overview-card { min-height: 110px; padding: 14px; border: 1px solid #e6eaf2; border-radius: 8px; background: #f8fafc; text-align: left; cursor: pointer; transition: border-color .15s, background .15s, box-shadow .15s; }
.overview-card:hover, .overview-card.active { border-color: #93c5fd; background: #eff6ff; box-shadow: 0 6px 18px rgba(37, 99, 235, .08); }
.overview-card span { display: block; color: #667085; font-weight: 700; }
.overview-card strong { display: block; margin-top: 8px; font-size: 28px; line-height: 1; color: #2563eb; }
.overview-card small { display: block; margin-top: 10px; color: #667085; line-height: 1.45; }
.section-nav { display: flex; gap: 8px; flex-wrap: wrap; padding: 10px; border: 1px solid #e6eaf2; border-radius: 8px; background: #fff; }
.section-nav button { display: inline-flex; align-items: center; gap: 8px; min-height: 38px; padding: 0 12px; border: 1px solid #e6eaf2; border-radius: 8px; background: #fff; color: #475569; font-weight: 700; cursor: pointer; }
.section-nav button.active { color: #2563eb; border-color: #93c5fd; background: #eff6ff; }
.section-nav em { min-width: 24px; padding: 2px 7px; border-radius: 999px; background: #f1f5f9; color: #64748b; font-style: normal; text-align: center; font-size: 12px; }
.section-nav button.active em { background: #dbeafe; color: #2563eb; }
.panel { padding: 14px; }
.panel h2 { margin-bottom: 12px; }
.strong { font-weight: 700; color: #111827; }
.account-risk-filters { display: grid; grid-template-columns: 260px 130px 130px 72px 72px; gap: 8px; }
.block-form { display: grid; grid-template-columns: 160px 240px 72px 72px; gap: 8px; }
.group-risk-filters { display: grid; grid-template-columns: 260px 150px 72px; gap: 8px; }
.table-block { margin-top: 14px; }
.table-pagination { display: flex; justify-content: flex-end; margin-top: 12px; }
.sub-panel { margin-top: 14px; border-top: 1px solid #eef2f7; padding-top: 12px; }
.sub-panel h3, .table-title { margin: 0 0 10px; font-size: 15px; color: #334155; }
.table-block h3 { margin: 0 0 10px; font-size: 15px; color: #334155; }
.note-panel { padding: 14px; display: flex; gap: 10px; color: #667085; }
.note-panel strong { color: #111827; }
.status-badge { display: inline-flex; align-items: center; padding: 3px 8px; border-radius: 6px; background: #eef4ff; color: #2563eb; font-weight: 600; }
.status-badge.danger { background: #fff1f2; color: #e11d48; }
button { font: inherit; }
@media (max-width: 900px) {
  .overview-grid, .account-risk-filters, .block-form, .group-risk-filters { grid-template-columns: 1fr; }
  .page-head, .panel-head, .panel-title-row, .note-panel, .section-heading { flex-direction: column; align-items: flex-start; }
}
@media (min-width: 901px) and (max-width: 1300px) {
  .overview-grid { grid-template-columns: repeat(3, minmax(0, 1fr)); }
}
</style>
