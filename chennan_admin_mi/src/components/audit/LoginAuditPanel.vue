<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { ElMessage } from "element-plus";
import { getAdminLoginAudits } from "@/api/audit";
import type { AdminLoginAudit } from "@/types";
import AuditDetailDialog from "@/components/audit/AuditDetailDialog.vue";
import AuditCleanupDialog from "@/components/audit/AuditCleanupDialog.vue";
import { useResponsive } from "@/composables/useResponsive";
import { useAuthStore } from "@/stores/auth";

const loading = ref(false);
const rows = ref<AdminLoginAudit[]>([]);
const total = ref(0);
const detailVisible = ref(false);
const cleanupVisible = ref(false);
const selectedRow = ref<AdminLoginAudit | null>(null);
const { isMobile } = useResponsive();
const authStore = useAuthStore();

const filters = reactive({
  keyword: "",
  role: "",
  stage: "",
  result: "",
});

const pagination = reactive({
  page: 1,
  pageSize: 20,
});
const pageSizeOptions = [10, 20, 50, 100];
const cleanupFilters = computed(() => ({
  keyword: filters.keyword || undefined,
  role: filters.role || undefined,
  stage: filters.stage || undefined,
  result: filters.result || undefined,
}));

async function fetchRows() {
  loading.value = true;
  try {
    const res = await getAdminLoginAudits({
      page: pagination.page,
      pageSize: pagination.pageSize,
      keyword: filters.keyword || undefined,
      role: filters.role || undefined,
      stage: filters.stage || undefined,
      result: filters.result || undefined,
    });
    rows.value = res.data.data?.list ?? [];
    total.value = res.data.data?.total ?? 0;
  } catch (error: any) {
    ElMessage.error(error.message || "加载登录审计失败");
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
  filters.stage = "";
  filters.result = "";
  handleSearch();
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

function resultLabel(result: string) {
  switch (result) {
    case "success":
      return "成功";
    case "require_totp_setup":
      return "待绑定2FA";
    case "require_totp_verify":
      return "待验证2FA";
    default:
      return "失败";
  }
}

function stageLabel(stage: string, result: string) {
  switch (stage) {
    case "turnstile":
      return "人机验证";
    case "whitelist":
      return "白名单校验";
    case "totp_setup":
      return "密码 + 2FA绑定";
    case "totp_verify":
      return "密码 + 2FA验证";
    default:
      if (result === "require_totp_setup") return "待2FA绑定";
      if (result === "require_totp_verify") return "待2FA验证";
      return "密码校验";
  }
}

const detailItems = computed(() => {
  if (!selectedRow.value) return [];

  return [
    { label: "时间", value: selectedRow.value.createdAt },
    { label: "用户名", value: selectedRow.value.username },
    { label: "角色", value: roleLabel(selectedRow.value.adminRole) },
    {
      label: "阶段",
      value: stageLabel(selectedRow.value.stage, selectedRow.value.result),
    },
    { label: "结果", value: resultLabel(selectedRow.value.result) },
    { label: "IP", value: selectedRow.value.ipAddress, code: true },
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

function openDetail(row: AdminLoginAudit) {
  selectedRow.value = row;
  detailVisible.value = true;
}

function handleCleanupSuccess() {
  pagination.page = 1;
  fetchRows();
}

onMounted(fetchRows);
</script>

<template>
  <div class="audit-panel">
    <div class="card-toolbar">
      <div class="audit-filters">
        <el-input
          v-model="filters.keyword"
          placeholder="用户名 / IP"
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
          v-model="filters.stage"
          placeholder="阶段"
          clearable
          class="filter-select"
        >
          <el-option label="人机验证" value="turnstile" />
          <el-option label="白名单校验" value="whitelist" />
          <el-option label="密码校验" value="password" />
          <el-option label="密码 + 2FA绑定" value="totp_setup" />
          <el-option label="密码 + 2FA验证" value="totp_verify" />
        </el-select>
        <el-select
          v-model="filters.result"
          placeholder="结果"
          clearable
          class="filter-select"
        >
          <el-option label="成功" value="success" />
          <el-option label="失败" value="failed" />
          <el-option label="待绑定2FA" value="require_totp_setup" />
          <el-option label="待验证2FA" value="require_totp_verify" />
        </el-select>
        <el-button type="primary" @click="handleSearch">筛选</el-button>
        <el-button @click="resetFilters">重置</el-button>
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
            <span
              class="status-badge"
              :class="
                row.result === 'success'
                  ? 'status-success'
                  : row.result === 'failed'
                    ? 'status-danger'
                    : 'status-info'
              "
            >
              {{ resultLabel(row.result) }}
            </span>
          </div>
          <div class="audit-mobile-title-row">
            <span class="cell-username">{{ row.username }}</span>
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
          <div class="audit-mobile-meta">
            <span class="status-badge status-stage status-stage-wide">{{
              stageLabel(row.stage, row.result)
            }}</span>
          </div>
          <div class="audit-mobile-ip">{{ row.ipAddress }}</div>
        </button>
        <div v-if="!loading && rows.length === 0" class="audit-mobile-empty">
          暂无登录审计记录
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
        <el-table-column label="用户名" min-width="120">
          <template #default="{ row }">
            <span class="cell-username">{{ row.username }}</span>
          </template>
        </el-table-column>
        <el-table-column label="角色" width="120">
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
        <el-table-column label="阶段" min-width="180">
          <template #default="{ row }">
            <span class="status-badge status-stage status-stage-wide">{{
              stageLabel(row.stage, row.result)
            }}</span>
          </template>
        </el-table-column>
        <el-table-column label="结果" width="130">
          <template #default="{ row }">
            <span
              class="status-badge"
              :class="
                row.result === 'success'
                  ? 'status-success'
                  : row.result === 'failed'
                    ? 'status-danger'
                    : 'status-info'
              "
            >
              {{ resultLabel(row.result) }}
            </span>
          </template>
        </el-table-column>
        <el-table-column label="IP" min-width="160">
          <template #default="{ row }">
            <span class="cell-code cell-muted">{{ row.ipAddress }}</span>
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
      title="登录审计详情"
      :items="detailItems"
    />
    <AuditCleanupDialog
      v-model="cleanupVisible"
      default-scope="login"
      :cleanup-filters="cleanupFilters"
      @cleanup-success="handleCleanupSuccess"
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

.cell-code {
  font-family: "Fira Code", "Courier New", monospace;
  font-size: 13px;
  color: #1e293b;
}

.cell-muted {
  font-size: 12px;
  color: #64748b;
}

.status-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 84px;
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

.status-stage {
  color: #d46b08;
  border-color: #ffd591;
  background: #fff7e6;
}

.status-stage-wide {
  min-width: 148px;
  white-space: normal;
  line-height: 1.25;
  text-align: center;
  padding: 4px 10px;
  height: auto;
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
