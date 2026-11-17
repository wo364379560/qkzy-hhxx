<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { ElMessage } from "element-plus";
import { getUserOperationLogs } from "@/api/audit";
import type { UserOperationLog } from "@/types";
import AuditDetailDialog from "@/components/audit/AuditDetailDialog.vue";
import { useResponsive } from "@/composables/useResponsive";

const loading = ref(false);
const rows = ref<UserOperationLog[]>([]);
const total = ref(0);
const selectedRow = ref<UserOperationLog | null>(null);
const detailVisible = ref(false);
const { isMobile } = useResponsive();

const filters = reactive({
  keyword: "",
  operationType: "account_delete",
  status: "",
  phone: "",
  imUserId: "",
  ipAddress: "",
  dateRange: [] as string[],
});

const pagination = reactive({
  page: 1,
  pageSize: 20,
});
const pageSizeOptions = [10, 20, 50, 100];

function operationLabel(row: UserOperationLog) {
  return row.operationName || (row.operationType === "account_delete" ? "用户注销账号" : row.operationType || "未知操作");
}

function statusLabel(status: string) {
  if (status === "success") return "成功";
  if (status === "failed") return "失败";
  return status || "未知";
}

function statusType(status: string) {
  if (status === "success") return "success";
  if (status === "failed") return "danger";
  return "info";
}

async function fetchRows() {
  loading.value = true;
  try {
    const [startTime, endTime] = filters.dateRange || [];
    const res = await getUserOperationLogs({
      page: pagination.page,
      pageSize: pagination.pageSize,
      keyword: filters.keyword || undefined,
      operationType: filters.operationType || undefined,
      status: filters.status || undefined,
      phone: filters.phone || undefined,
      imUserId: filters.imUserId || undefined,
      ipAddress: filters.ipAddress || undefined,
      startTime: startTime || undefined,
      endTime: endTime || undefined,
    });
    rows.value = res.data.data?.list ?? [];
    total.value = res.data.data?.total ?? 0;
  } catch (error: any) {
    ElMessage.error(error.message || "加载用户操作日志失败");
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
  filters.operationType = "account_delete";
  filters.status = "";
  filters.phone = "";
  filters.imUserId = "";
  filters.ipAddress = "";
  filters.dateRange = [];
  handleSearch();
}

function openDetail(row: UserOperationLog) {
  selectedRow.value = row;
  detailVisible.value = true;
}

const detailItems = computed(() => {
  if (!selectedRow.value) return [];
  const row = selectedRow.value;
  return [
    { label: "时间", value: row.createdAt },
    { label: "操作", value: operationLabel(row) },
    { label: "状态", value: statusLabel(row.status) },
    { label: "操作说明", value: row.operationDetail || "—", multiline: true },
    { label: "失败原因", value: row.failureReason || "—", multiline: true },
    { label: "IM用户ID", value: row.imUserId || "—", code: true },
    { label: "手机号", value: row.phone || "—", code: true },
    { label: "昵称", value: row.nickname || "—" },
    { label: "归属人", value: row.ownerName || "—" },
    { label: "IP", value: row.ipAddress || "—", code: true },
    { label: "设备ID", value: row.deviceId || "—", code: true },
    { label: "设备", value: row.deviceName || "—" },
    { label: "平台", value: row.platform || "—" },
    { label: "App版本", value: row.appVersion || "—" },
    { label: "User-Agent", value: row.userAgent || "—", multiline: true },
  ];
});

onMounted(fetchRows);
</script>

<template>
  <div class="page">
    <div class="page-header">
      <div>
        <h1 class="page-title">用户操作日志</h1>
        <p class="page-subtitle">移动端用户关键操作记录</p>
      </div>
    </div>

    <div class="card">
      <div class="card-toolbar">
        <div class="audit-filters">
          <el-input
            v-model="filters.keyword"
            placeholder="手机号 / IM用户ID / 昵称 / 归属人"
            clearable
            class="filter-keyword"
            @keyup.enter="handleSearch"
          />
          <el-select v-model="filters.operationType" placeholder="操作类型" clearable class="filter-select">
            <el-option label="用户注销账号" value="account_delete" />
          </el-select>
          <el-select v-model="filters.status" placeholder="状态" clearable class="filter-select">
            <el-option label="成功" value="success" />
            <el-option label="失败" value="failed" />
          </el-select>
          <el-input
            v-model="filters.phone"
            placeholder="手机号精准查询"
            clearable
            class="filter-select"
            @keyup.enter="handleSearch"
          />
          <el-input
            v-model="filters.imUserId"
            placeholder="IM用户ID精准查询"
            clearable
            class="filter-select"
            @keyup.enter="handleSearch"
          />
          <el-input
            v-model="filters.ipAddress"
            placeholder="IP精准查询"
            clearable
            class="filter-select"
            @keyup.enter="handleSearch"
          />
          <el-date-picker
            v-model="filters.dateRange"
            type="datetimerange"
            range-separator="至"
            start-placeholder="开始时间"
            end-placeholder="结束时间"
            value-format="YYYY-MM-DD HH:mm:ss"
            class="filter-range"
          />
          <el-button type="primary" @click="handleSearch">筛选</el-button>
          <el-button @click="resetFilters">重置</el-button>
        </div>
      </div>

      <div v-loading="loading" class="audit-mobile-list" v-if="isMobile">
        <article v-for="row in rows" :key="row.id" class="audit-mobile-card" @click="openDetail(row)">
          <div class="audit-mobile-card-top">
            <span class="audit-mobile-time">{{ row.createdAt }}</span>
            <el-tag :type="statusType(row.status)" size="small">{{ statusLabel(row.status) }}</el-tag>
          </div>
          <div class="audit-mobile-title-row">
            <strong>{{ operationLabel(row) }}</strong>
            <code>{{ row.imUserId || "—" }}</code>
          </div>
          <dl class="audit-mobile-meta">
            <dt>手机号</dt>
            <dd>{{ row.phone || "—" }}</dd>
            <dt>IP</dt>
            <dd>{{ row.ipAddress || "—" }}</dd>
            <dt>设备</dt>
            <dd>{{ row.deviceName || row.platform || "—" }}</dd>
          </dl>
          <div class="audit-mobile-desc">{{ row.operationDetail || row.failureReason || "—" }}</div>
        </article>
      </div>

      <el-table v-else v-loading="loading" :data="rows" border>
        <el-table-column prop="createdAt" label="时间" width="170" />
        <el-table-column label="操作" min-width="130">
          <template #default="{ row }">{{ operationLabel(row) }}</template>
        </el-table-column>
        <el-table-column label="状态" width="90">
          <template #default="{ row }">
            <el-tag :type="statusType(row.status)" size="small">{{ statusLabel(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="imUserId" label="IM用户ID" min-width="130" />
        <el-table-column prop="phone" label="手机号" min-width="120" />
        <el-table-column prop="nickname" label="昵称" min-width="120" />
        <el-table-column prop="ownerName" label="归属人" min-width="110" />
        <el-table-column prop="ipAddress" label="IP" min-width="130" />
        <el-table-column label="设备" min-width="150">
          <template #default="{ row }">{{ row.deviceName || row.platform || "—" }}</template>
        </el-table-column>
        <el-table-column prop="appVersion" label="App版本" min-width="100" />
        <el-table-column prop="operationDetail" label="操作详情" min-width="260" show-overflow-tooltip />
        <el-table-column label="操作" width="90" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="openDetail(row)">详情</el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="table-footer">
        <el-pagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.pageSize"
          :page-sizes="pageSizeOptions"
          :total="total"
          layout="total, sizes, prev, pager, next"
          @current-change="fetchRows"
          @size-change="handleSearch"
        />
      </div>
    </div>

    <AuditDetailDialog
      v-model:visible="detailVisible"
      title="用户操作日志详情"
      :items="detailItems"
    />
  </div>
</template>

<style scoped>
.audit-filters {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  width: 100%;
}

.filter-keyword {
  width: 280px;
}

.filter-select {
  width: 170px;
}

.filter-range {
  width: 360px;
}

.table-footer {
  display: flex;
  justify-content: flex-end;
  padding-top: 16px;
}

.audit-mobile-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.audit-mobile-card {
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  padding: 14px;
  background: #fff;
}

.audit-mobile-card-top,
.audit-mobile-title-row,
.audit-mobile-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.audit-mobile-time,
.audit-mobile-meta dt {
  color: #8a97ab;
  font-size: 12px;
}

.audit-mobile-title-row {
  margin-top: 10px;
}

.audit-mobile-meta {
  flex-wrap: wrap;
  justify-content: flex-start;
  margin: 10px 0 0;
}

.audit-mobile-meta dt {
  width: 54px;
}

.audit-mobile-meta dd {
  margin: 0 12px 0 0;
  color: #4b5563;
}

.audit-mobile-desc {
  margin-top: 10px;
  color: #6b7280;
  font-size: 13px;
  line-height: 1.5;
}
</style>
