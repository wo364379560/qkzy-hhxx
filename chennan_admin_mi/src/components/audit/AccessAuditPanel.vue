<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { ElMessage } from "element-plus";
import { exportAdminAccessAudits, getAdminAccessAudits } from "@/api/audit";
import type { AdminAccessAudit } from "@/types";
import AuditDetailDialog from "@/components/audit/AuditDetailDialog.vue";
import AuditCleanupDialog from "@/components/audit/AuditCleanupDialog.vue";
import { useResponsive } from "@/composables/useResponsive";
import { useAuthStore } from "@/stores/auth";
import { getUserDetail } from "@/api/user";
import { getGroupDetail } from "@/api/group";

const loading = ref(false);
const rows = ref<AdminAccessAudit[]>([]);
const total = ref(0);
const detailVisible = ref(false);
const cleanupVisible = ref(false);
const selectedRow = ref<AdminAccessAudit | null>(null);
const targetDetailVisible = ref(false);
const targetDetailTitle = ref("");
const targetDetailItems = ref<{ label: string; value: string; code?: boolean; multiline?: boolean }[]>([]);
const { isMobile } = useResponsive();
const authStore = useAuthStore();
const exporting = ref(false);
const actionKeyOptions = [
  { label: "创建用户", value: "user.create" },
  { label: "更新用户资料", value: "user.update_profile" },
  { label: "封禁用户", value: "user.ban" },
  { label: "解封用户", value: "user.unban" },
  { label: "踢出用户", value: "user.kick" },
  { label: "注销用户", value: "user.delete" },
  { label: "转移用户归属", value: "user.transfer_owner" },
  { label: "批量建群权限", value: "user.batch_set_create_group" },
  { label: "创建群组", value: "group.create" },
  { label: "添加群成员", value: "group.add_member" },
  { label: "移除群成员", value: "group.remove_member" },
  { label: "群组禁言", value: "group.mute" },
  { label: "群组解散", value: "group.dismiss" },
  { label: "创建助理", value: "assistant.create" },
  { label: "禁用助理", value: "assistant.ban" },
  { label: "启用助理", value: "assistant.unban" },
  { label: "创建管理员", value: "admin_user.create" },
  { label: "禁用管理员", value: "admin_user.ban" },
  { label: "启用管理员", value: "admin_user.unban" },
  { label: "修改后台域名池", value: "config.update_backend_domain_pool" },
  { label: "修改万能验证码", value: "config.update_sms_bypass" },
];

const filters = reactive({
  keyword: "",
  role: "",
  accessType: "",
  method: "",
  targetType: "",
  actionKey: "",
  targetKeyword: "",
});

const pagination = reactive({
  page: 1,
  pageSize: 20,
});
const pageSizeOptions = [10, 20, 50, 100];
const cleanupFilters = computed(() => ({
  keyword: filters.keyword || undefined,
  role: filters.role || undefined,
  accessType: filters.accessType || undefined,
  method: filters.method || undefined,
}));

async function fetchRows() {
  loading.value = true;
  try {
    const res = await getAdminAccessAudits({
      page: pagination.page,
      pageSize: pagination.pageSize,
      keyword: filters.keyword || undefined,
      role: filters.role || undefined,
      accessType: filters.accessType || undefined,
      method: filters.method || undefined,
      targetType: filters.targetType || undefined,
      actionKey: filters.actionKey || undefined,
      targetKeyword: filters.targetKeyword || undefined,
    });
    rows.value = res.data.data?.list ?? [];
    total.value = res.data.data?.total ?? 0;
  } catch (error: any) {
    ElMessage.error(error.message || "加载访问审计失败");
  } finally {
    loading.value = false;
  }
}

function handleSearch() {
  pagination.page = 1;
  fetchRows();
}

function resetFilters() {
  filters.keyword = "";
  filters.role = "";
  filters.accessType = "";
  filters.method = "";
  filters.targetType = "";
  filters.actionKey = "";
  filters.targetKeyword = "";
  handleSearch();
}

function targetTypeLabel(type: string) {
  switch (type) {
    case "user":
      return "用户";
    case "group":
      return "群组";
    case "admin_user":
      return "管理员";
    case "assistant":
      return "助理";
    case "system_config":
      return "配置";
    default:
      return type || "—";
  }
}

function roleLabel(role: string) {
  return role === "super_admin"
    ? "超级管理员"
    : role === "admin"
      ? "管理员"
      : role === "assistant"
        ? "助理"
        : "未知";
}

function accessTypeLabel(type: string) {
  return type === "illegal_access" ? "非法访问" : "系统用户访问";
}

function formatChangeSet(value: string) {
  if (!value) return "—";
  try {
    const parsed = JSON.parse(value);
    if (!parsed || typeof parsed !== "object") {
      return value;
    }
    return Object.entries(parsed as Record<string, any>)
      .map(([key, item]) => {
        const before = item?.before ?? "—";
        const after = item?.after ?? "—";
        return `${key}: ${before} -> ${after}`;
      })
      .join("\n");
  } catch {
    return value;
  }
}

function resolveTargetType(row: AdminAccessAudit | null) {
  if (!row?.targetType) return null;
  return row.targetType as const;
}

async function openTargetDetail() {
  const row = selectedRow.value;
  if (!row?.targetId) return;

  const targetType = resolveTargetType(row);
  if (!targetType) {
    ElMessage.info("当前记录暂不支持自动查看目标详情");
    return;
  }

  try {
    if (targetType === "user") {
      const res = await getUserDetail(row.targetId);
      const data = res.data.data;
      targetDetailTitle.value = `用户详情 · #${data.id}`;
      targetDetailItems.value = [
        { label: "用户ID", value: String(data.id), code: true },
        { label: "IM用户ID", value: data.im_user_id || "—", code: true },
        { label: "昵称", value: data.nickname || "—" },
        { label: "手机号", value: data.phone || "—", code: true },
        { label: "当前登录IP", value: data.current_login_ip || "—", code: true },
        { label: "归属人", value: data.owner_name || "—" },
        { label: "状态", value: data.status || "—" },
        { label: "登录状态", value: data.login_status || "—" },
        { label: "注册时间", value: data.created_at || "—" },
      ];
    } else {
      const res = await getGroupDetail(row.targetId);
      const data = res.data.data;
      targetDetailTitle.value = `群组详情 · #${data.id}`;
      targetDetailItems.value = [
        { label: "群组ID", value: String(data.id), code: true },
        { label: "IM群ID", value: data.imGroupId || "—", code: true },
        { label: "群名称", value: data.name || "—" },
        { label: "群类型", value: data.groupType || "—" },
        { label: "归属人", value: data.ownerName || "—" },
        { label: "群主昵称", value: data.ownerNickname || "—" },
        { label: "成员数", value: String(data.memberCount ?? 0), code: true },
        { label: "状态", value: data.status || "—" },
        { label: "创建时间", value: data.createdAt || "—" },
      ];
    }

    targetDetailVisible.value = true;
  } catch (error: any) {
    ElMessage.error(error.message || "获取目标详情失败");
  }
}

const detailItems = computed(() => {
  if (!selectedRow.value) return [];

  const targetType = resolveTargetType(selectedRow.value);

  return [
    { label: "时间", value: selectedRow.value.createdAt },
    { label: "访问类型", value: accessTypeLabel(selectedRow.value.accessType) },
    { label: "账号", value: selectedRow.value.adminUsername || "未识别" },
    { label: "角色", value: roleLabel(selectedRow.value.adminRole) },
    { label: "模块", value: selectedRow.value.module },
    { label: "动作", value: selectedRow.value.actionSummary || selectedRow.value.actionLabel },
    { label: "动作码", value: selectedRow.value.actionKey || "—", code: true },
    { label: "对象类型", value: targetType ? targetTypeLabel(targetType) : "—" },
    { label: "对象账号", value: selectedRow.value.targetUsername || "—", code: true },
    { label: "对象名称", value: selectedRow.value.targetDisplayName || "—" },
    {
      label: "目标ID",
      value: selectedRow.value.targetId == null ? "—" : String(selectedRow.value.targetId),
      code: true,
      onClick:
        selectedRow.value.targetId != null && targetType != null
          ? openTargetDetail
          : undefined,
    },
    { label: "方法", value: selectedRow.value.method, code: true },
    {
      label: "状态码",
      value: String(selectedRow.value.statusCode),
      code: true,
    },
    { label: "IP", value: selectedRow.value.ipAddress, code: true },
    { label: "结果", value: selectedRow.value.success ? "成功" : "失败" },
    {
      label: "字段变更",
      value: formatChangeSet(selectedRow.value.changeSet || selectedRow.value.actionDetail || ""),
      multiline: true,
    },
    {
      label: "路由",
      value: selectedRow.value.route,
      code: true,
      multiline: true,
    },
    {
      label: "失败原因",
      value: selectedRow.value.failureReason || "—",
      multiline: true,
    },
    {
      label: "User-Agent",
      value: selectedRow.value.userAgent || "—",
      multiline: true,
    },
  ];
});

function openDetail(row: AdminAccessAudit) {
  selectedRow.value = row;
  detailVisible.value = true;
}

function handleCleanupSuccess() {
  pagination.page = 1;
  fetchRows();
}

async function handleExport() {
  exporting.value = true;
  try {
    const res = await exportAdminAccessAudits({
      keyword: filters.keyword || undefined,
      role: filters.role || undefined,
      accessType: filters.accessType || undefined,
      method: filters.method || undefined,
      targetType: filters.targetType || undefined,
      actionKey: filters.actionKey || undefined,
      targetKeyword: filters.targetKeyword || undefined,
    });
    const blob = new Blob([res.data], { type: "text/csv;charset=utf-8;" });
    const url = window.URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = `admin_access_audits_${Date.now()}.csv`;
    link.click();
    window.URL.revokeObjectURL(url);
  } catch (error: any) {
    ElMessage.error(error.message || "导出行为审计失败");
  } finally {
    exporting.value = false;
  }
}

onMounted(fetchRows);
</script>

<template>
  <div class="audit-panel">
    <div class="card-toolbar">
      <div class="audit-filters">
        <el-input
          v-model="filters.keyword"
          placeholder="用户名 / 路由 / IP"
          clearable
          class="filter-keyword"
          @keyup.enter="handleSearch"
        />
        <el-input
          v-model="filters.targetKeyword"
          placeholder="对象账号 / 名称"
          clearable
          class="filter-keyword"
          @keyup.enter="handleSearch"
        />
        <el-select
          v-model="filters.role"
          placeholder="角色"
          clearable
          class="filter-select"
        >
          <el-option label="超级管理员" value="super_admin" />
          <el-option label="管理员" value="admin" />
          <el-option label="助理" value="assistant" />
        </el-select>
        <el-select
          v-model="filters.accessType"
          placeholder="访问类型"
          clearable
          class="filter-select"
        >
          <el-option label="系统用户访问" value="system_user" />
          <el-option label="非法访问" value="illegal_access" />
        </el-select>
        <el-select
          v-model="filters.method"
          placeholder="方法"
          clearable
          class="filter-select"
        >
          <el-option label="GET" value="GET" />
          <el-option label="POST" value="POST" />
          <el-option label="PUT" value="PUT" />
          <el-option label="PATCH" value="PATCH" />
          <el-option label="DELETE" value="DELETE" />
        </el-select>
        <el-select
          v-model="filters.targetType"
          placeholder="对象类型"
          clearable
          class="filter-select"
        >
          <el-option label="用户" value="user" />
          <el-option label="群组" value="group" />
          <el-option label="管理员" value="admin_user" />
          <el-option label="助理" value="assistant" />
          <el-option label="配置" value="system_config" />
        </el-select>
        <el-select
          v-model="filters.actionKey"
          placeholder="动作类型"
          clearable
          class="filter-select"
        >
          <el-option
            v-for="item in actionKeyOptions"
            :key="item.value"
            :label="item.label"
            :value="item.value"
          />
        </el-select>
        <el-button type="primary" @click="handleSearch">筛选</el-button>
        <el-button @click="resetFilters">重置</el-button>
        <el-button :loading="exporting" @click="handleExport">导出</el-button>
        <el-button
          v-if="authStore.isSuperAdmin"
          type="danger"
          plain
          @click="cleanupVisible = true"
        >
          清理日志
        </el-button>
      </div>
      <div class="toolbar-right">
        <span v-if="!isMobile" class="toolbar-hint">点击行查看详情</span>
        <span class="total-count">共 {{ total }} 条</span>
      </div>
    </div>

    <template v-if="isMobile">
      <div v-loading="loading" class="audit-mobile-list">
        <button
          v-for="row in rows"
          :key="row.id"
          type="button"
          class="audit-mobile-card"
          @click="openDetail(row)"
        >
          <div class="audit-mobile-card-top">
            <span class="audit-mobile-time">{{ row.createdAt }}</span>
            <div class="audit-mobile-badges">
              <span
                class="status-badge"
                :class="
                  row.accessType === 'illegal_access'
                    ? 'status-danger'
                    : 'status-info'
                "
              >
                {{ accessTypeLabel(row.accessType) }}
              </span>
              <span
                class="status-badge"
                :class="row.success ? 'status-success' : 'status-danger'"
              >
                {{ row.success ? "成功" : "失败" }}
              </span>
            </div>
          </div>
          <div class="audit-mobile-title-row">
            <span class="cell-action">{{ row.actionSummary || row.actionLabel }}</span>
            <span class="status-badge status-module">{{ row.module }}</span>
          </div>
          <div class="audit-mobile-meta">
            <span class="cell-username">{{
              row.adminUsername || "未识别"
            }}</span>
            <span
              class="status-badge"
              :class="
                row.adminRole === 'assistant'
                  ? 'status-assistant'
                  : 'status-admin'
              "
            >
              {{ roleLabel(row.adminRole) }}
            </span>
          </div>
          <div class="audit-mobile-ip">{{ row.ipAddress }}</div>
          <div class="audit-mobile-ip">
            {{ row.targetDisplayName || row.targetUsername || "—" }}
          </div>
        </button>
        <div v-if="!loading && rows.length === 0" class="audit-mobile-empty">
          暂无访问审计记录
        </div>
      </div>
    </template>
    <div v-else class="table-scroll-x audit-table-wrap">
      <el-table
        :data="rows"
        v-loading="loading"
        stripe
        size="small"
        style="width: 100%"
        class="audit-table"
        @row-click="openDetail"
      >
        <el-table-column label="序号" width="60" align="center">
          <template #default="{ $index }">
            <span class="cell-id">{{
              (pagination.page - 1) * pagination.pageSize + $index + 1
            }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="createdAt" label="时间" min-width="160" />
        <el-table-column label="访问类型" width="110">
          <template #default="{ row }">
            <span
              class="status-badge"
              :class="
                row.accessType === 'illegal_access'
                  ? 'status-danger'
                  : 'status-info'
              "
            >
              {{ accessTypeLabel(row.accessType) }}
            </span>
          </template>
        </el-table-column>
        <el-table-column label="模块" width="110">
          <template #default="{ row }">
            <span class="status-badge status-module">{{ row.module }}</span>
          </template>
        </el-table-column>
        <el-table-column label="动作" min-width="150">
          <template #default="{ row }">
            <span class="cell-action">{{ row.actionSummary || row.actionLabel }}</span>
          </template>
        </el-table-column>
        <el-table-column label="对象类型" width="100">
          <template #default="{ row }">
            <span class="status-badge status-module">{{ targetTypeLabel(row.targetType) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="账号" min-width="120">
          <template #default="{ row }">
            <span v-if="row.adminUsername" class="cell-username">{{
              row.adminUsername
            }}</span>
            <span v-else class="cell-empty">未识别</span>
          </template>
        </el-table-column>
        <el-table-column label="对象" min-width="150">
          <template #default="{ row }">
            <span v-if="row.targetDisplayName" class="cell-username">{{
              row.targetDisplayName
            }}</span>
            <span v-else-if="row.targetUsername" class="cell-code cell-muted">{{ row.targetUsername }}</span>
            <span v-else class="cell-empty">—</span>
          </template>
        </el-table-column>
        <el-table-column label="角色" width="90">
          <template #default="{ row }">
            <span
              class="status-badge"
              :class="
                row.adminRole === 'assistant'
                  ? 'status-assistant'
                  : 'status-admin'
              "
            >
              {{ roleLabel(row.adminRole) }}
            </span>
          </template>
        </el-table-column>
        <el-table-column label="IP" min-width="130">
          <template #default="{ row }">
            <span class="cell-code cell-muted">{{ row.ipAddress }}</span>
          </template>
        </el-table-column>
        <el-table-column label="结果" width="90">
          <template #default="{ row }">
            <span
              class="status-badge"
              :class="row.success ? 'status-success' : 'status-danger'"
            >
              {{ row.success ? "成功" : "失败" }}
            </span>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <div class="pagination-bar">
      <el-pagination
        v-model:current-page="pagination.page"
        :page-size="pagination.pageSize"
        :page-sizes="pageSizeOptions"
        :total="total"
        layout="sizes, prev, pager, next"
        background
        small
        @size-change="
          (size: number) => {
            pagination.pageSize = size;
            pagination.page = 1;
            fetchRows();
          }
        "
        @current-change="fetchRows"
      />
    </div>

    <AuditDetailDialog
      v-model:visible="detailVisible"
      title="行为审计详情"
      :items="detailItems"
    />
    <AuditCleanupDialog
      v-model="cleanupVisible"
      default-scope="access"
      :cleanup-filters="cleanupFilters"
      @cleanup-success="handleCleanupSuccess"
    />
    <AuditDetailDialog
      v-model:visible="targetDetailVisible"
      :title="targetDetailTitle"
      :items="targetDetailItems"
    />
  </div>
</template>

<style scoped>
.audit-panel {
  display: flex;
  flex-direction: column;
}

.card-toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  border-bottom: 1px solid #f0f0f0;
}

.audit-filters {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  flex: 1;
}

.filter-keyword {
  width: 260px;
}

.filter-select {
  width: 150px;
}

.toolbar-right {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-left: auto;
  white-space: nowrap;
}

.audit-table-wrap {
  border-bottom: 1px solid #f0f0f0;
}

.audit-mobile-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 12px;
}

.audit-mobile-card {
  display: flex;
  flex-direction: column;
  gap: 10px;
  width: 100%;
  padding: 14px;
  border: 1px solid #edf1f7;
  border-radius: 14px;
  background: #fff;
  text-align: left;
  cursor: pointer;
}

.audit-mobile-card-top,
.audit-mobile-title-row,
.audit-mobile-meta {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 10px;
}

.audit-mobile-badges {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 6px;
}

.audit-mobile-time {
  font-size: 12px;
  color: #64748b;
}

.audit-mobile-ip {
  font-family: "Fira Code", "Courier New", monospace;
  font-size: 12px;
  color: #64748b;
}

.audit-mobile-empty {
  padding: 28px 14px;
  border: 1px dashed #dbe4ef;
  border-radius: 12px;
  text-align: center;
  font-size: 13px;
  color: #94a3b8;
}

.toolbar-hint {
  font-size: 12px;
  color: #94a3b8;
}

.pagination-bar {
  display: flex;
  justify-content: flex-end;
  padding: 12px 16px;
  border-top: 1px solid #f0f0f0;
}

.total-count {
  font-size: 12px;
  color: #94a3b8;
  white-space: nowrap;
}

.cell-id {
  font-size: 13px;
  color: #94a3b8;
}

.cell-username {
  font-size: 13px;
  font-weight: 500;
  color: #1677ff;
}

.cell-action {
  font-size: 13px;
  font-weight: 500;
  color: #1e293b;
}

.cell-code {
  font-family: "Fira Code", "Courier New", monospace;
  font-size: 13px;
  color: #1e293b;
}

.cell-muted {
  font-size: 12px;
  color: #64748b;
}

.cell-empty {
  color: #d1d5db;
}

.status-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 72px;
  padding: 0 9px;
  height: 24px;
  border-radius: 6px;
  font-size: 12px;
  font-weight: 500;
  border: 1px solid transparent;
}

.status-success {
  color: #52c41a;
  border-color: #b7eb8f;
  background: #f6ffed;
}

.status-danger {
  color: #ff4d4f;
  border-color: #ffa39e;
  background: #fff2f0;
}

.status-info {
  color: #1677ff;
  border-color: #91caff;
  background: #e6f4ff;
}

.status-admin {
  color: #1677ff;
  border-color: #91caff;
  background: #e6f4ff;
}

.status-assistant {
  color: #9333ea;
  border-color: #e9d5ff;
  background: #faf5ff;
}

.status-module {
  color: #0f766e;
  border-color: #99f6e4;
  background: #ecfeff;
}

:deep(.audit-table .el-table__row) {
  cursor: pointer;
}

:deep(.audit-table .el-table__row:hover > td.el-table__cell) {
  background: #f7fbff !important;
}

:deep(.audit-table .el-table__row:hover .cell-username) {
  text-decoration: underline;
}

@media (max-width: 767px) {
  .card-toolbar,
  .audit-filters,
  .toolbar-right,
  .pagination-bar {
    flex-direction: column;
    align-items: stretch;
  }

  .filter-keyword,
  .filter-select {
    width: 100%;
  }

  .toolbar-right {
    margin-left: 0;
    gap: 8px;
  }

  .pagination-bar {
    padding: 12px;
  }
}
</style>
