<script setup lang="ts">
import { onMounted, reactive, ref } from "vue";
import { ElMessage } from "element-plus";
import { getSMSSendLogs } from "@/api/audit";
import type { SMSSendLog } from "@/types";
import { useResponsive } from "@/composables/useResponsive";

const loading = ref(false);
const rows = ref<SMSSendLog[]>([]);
const total = ref(0);
const { isMobile } = useResponsive();

const filters = reactive({
  keyword: "",
  scene: "",
  status: "",
});

const pagination = reactive({
  page: 1,
  pageSize: 20,
});

const pageSizeOptions = [10, 20, 50, 100];

async function fetchRows() {
  loading.value = true;
  try {
    const res = await getSMSSendLogs({
      page: pagination.page,
      pageSize: pagination.pageSize,
      keyword: filters.keyword || undefined,
      scene: filters.scene || undefined,
      status: filters.status || undefined,
    });
    rows.value = res.data.data?.list ?? [];
    total.value = res.data.data?.total ?? 0;
  } catch (error: any) {
    ElMessage.error(error.message || "加载短信发送日志失败");
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
  filters.scene = "";
  filters.status = "";
  handleSearch();
}

function sceneLabel(scene: string) {
  switch (scene) {
    case "login":
      return "登录";
    case "register":
      return "注册";
    case "reset_password":
      return "重置密码";
    default:
      return scene || "general";
  }
}

function statusLabel(status: string) {
  switch (status) {
    case "success":
      return "成功";
    case "failed":
      return "失败";
    case "rejected":
      return "拦截";
    default:
      return status || "未知";
  }
}

function statusClass(status: string) {
  switch (status) {
    case "success":
      return "status-success";
    case "failed":
      return "status-danger";
    case "rejected":
      return "status-warning";
    default:
      return "status-neutral";
  }
}

onMounted(fetchRows);
</script>

<template>
  <div class="page">
    <div class="card">
      <div class="card-header">
        <div>
          <h1 class="page-title">短信发送日志</h1>
          <p class="page-sub">查看验证码发送成功、失败与频控拦截记录，用于排查刷短信和供应商异常。</p>
        </div>
      </div>

      <div class="card-toolbar">
        <div class="audit-filters">
          <el-input
            v-model="filters.keyword"
            placeholder="手机号 / IP / 设备 ID / 拦截原因"
            clearable
            class="filter-keyword"
            @keyup.enter="handleSearch"
          />
          <el-select v-model="filters.scene" placeholder="场景" clearable class="filter-select">
            <el-option label="登录" value="login" />
            <el-option label="注册" value="register" />
            <el-option label="重置密码" value="reset_password" />
            <el-option label="通用" value="general" />
          </el-select>
          <el-select v-model="filters.status" placeholder="状态" clearable class="filter-select">
            <el-option label="成功" value="success" />
            <el-option label="失败" value="failed" />
            <el-option label="拦截" value="rejected" />
          </el-select>
          <el-button type="primary" @click="handleSearch">筛选</el-button>
          <el-button @click="resetFilters">重置</el-button>
        </div>
        <div class="toolbar-right">
          <span class="total-count">共 {{ total }} 条</span>
        </div>
      </div>

      <template v-if="isMobile">
        <div v-loading="loading" class="audit-mobile-list">
          <article v-for="row in rows" :key="row.id" class="audit-mobile-card">
            <div class="audit-mobile-card-top">
              <span class="audit-mobile-time">{{ row.createdAt }}</span>
              <div class="audit-mobile-badges">
                <span class="status-badge" :class="statusClass(row.status)">{{ statusLabel(row.status) }}</span>
              </div>
            </div>
            <div class="audit-mobile-title-row">
              <strong>{{ row.phone || "—" }}</strong>
              <span>{{ sceneLabel(row.scene) }}</span>
            </div>
            <div class="audit-mobile-meta">
              <div><dt>IP</dt><dd>{{ row.ipAddress || "—" }}</dd></div>
              <div><dt>设备</dt><dd>{{ row.deviceID || "—" }}</dd></div>
              <div><dt>通道</dt><dd>{{ row.provider || "—" }}</dd></div>
            </div>
            <div class="audit-mobile-desc">{{ row.rejectReason || "—" }}</div>
          </article>
        </div>
      </template>

      <div v-else class="table-scroll-x page-table-wrap">
        <el-table :data="rows" v-loading="loading" stripe size="small" style="width: 100%">
          <el-table-column prop="createdAt" label="时间" min-width="160" />
          <el-table-column label="手机号" min-width="130">
            <template #default="{ row }"><span class="cell-code">{{ row.phone || "—" }}</span></template>
          </el-table-column>
          <el-table-column label="场景" width="110">
            <template #default="{ row }">{{ sceneLabel(row.scene) }}</template>
          </el-table-column>
          <el-table-column label="状态" width="100">
            <template #default="{ row }">
              <span class="status-badge" :class="statusClass(row.status)">{{ statusLabel(row.status) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="IP" min-width="130">
            <template #default="{ row }"><span class="cell-code">{{ row.ipAddress || "—" }}</span></template>
          </el-table-column>
          <el-table-column label="设备 ID" min-width="180">
            <template #default="{ row }"><span class="cell-code">{{ row.deviceID || "—" }}</span></template>
          </el-table-column>
          <el-table-column label="通道" width="100">
            <template #default="{ row }">{{ row.provider || "—" }}</template>
          </el-table-column>
          <el-table-column label="原因" min-width="220">
            <template #default="{ row }">{{ row.rejectReason || "—" }}</template>
          </el-table-column>
        </el-table>
      </div>

      <div class="table-pagination">
        <el-pagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.pageSize"
          background
          layout="total, sizes, prev, pager, next"
          :page-sizes="pageSizeOptions"
          :total="total"
          @size-change="fetchRows"
          @current-change="fetchRows"
        />
      </div>
    </div>
  </div>
</template>

<style scoped>
.audit-filters {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
}

.filter-keyword {
  width: 320px;
}

.filter-select {
  width: 160px;
}

.status-success {
  background: #ecfdf3;
  color: #027a48;
}

.status-danger {
  background: #fef3f2;
  color: #b42318;
}

.status-warning {
  background: #fffaeb;
  color: #b54708;
}

.status-neutral {
  background: #f3f4f6;
  color: #475467;
}

.audit-mobile-list {
  display: grid;
  gap: 12px;
}

.audit-mobile-card {
  border: 1px solid #eef2f7;
  border-radius: 12px;
  padding: 14px;
  background: #fff;
}

.audit-mobile-card-top,
.audit-mobile-title-row,
.audit-mobile-meta {
  display: flex;
  justify-content: space-between;
  gap: 12px;
}

.audit-mobile-title-row {
  margin-top: 8px;
  align-items: center;
}

.audit-mobile-meta {
  margin-top: 10px;
  flex-wrap: wrap;
}

.audit-mobile-meta dt {
  font-size: 12px;
  color: #98a2b3;
}

.audit-mobile-meta dd {
  margin: 4px 0 0;
  color: #101828;
}

.audit-mobile-desc {
  margin-top: 10px;
  color: #475467;
}

.table-pagination {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}

@media (max-width: 768px) {
  .filter-keyword,
  .filter-select {
    width: 100%;
  }
}
</style>
