<script setup lang="ts">
import { onMounted, reactive, ref } from "vue";
import { ElMessage } from "element-plus";
import { getChatQuietHoursHitLogs } from "@/api/audit";
import type { ChatQuietHoursHitLog } from "@/types";
import { useResponsive } from "@/composables/useResponsive";

const loading = ref(false);
const rows = ref<ChatQuietHoursHitLog[]>([]);
const total = ref(0);
const { isMobile } = useResponsive();

const filters = reactive({
  keyword: "",
  action: "",
  source: "",
});

const pagination = reactive({
  page: 1,
  pageSize: 20,
});

const pageSizeOptions = [10, 20, 50, 100];

async function fetchRows() {
  loading.value = true;
  try {
    const res = await getChatQuietHoursHitLogs({
      page: pagination.page,
      pageSize: pagination.pageSize,
      keyword: filters.keyword || undefined,
      action: filters.action || undefined,
      source: filters.source || undefined,
    });
    rows.value = res.data.data?.list ?? [];
    total.value = res.data.data?.total ?? 0;
  } catch (error: any) {
    ElMessage.error(error.message || "加载静默时段命中日志失败");
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
  filters.action = "";
  filters.source = "";
  handleSearch();
}

function actionLabel(action: string) {
  switch (action) {
    case "register":
      return "注册";
    case "send_message":
      return "发消息";
    case "create_group":
      return "创建群组";
    case "auto_dismiss_group":
      return "自动解散群";
    case "add_friend":
      return "加好友";
    default:
      return action || "未知";
  }
}

function sourceLabel(source: string) {
  switch (source) {
    case "chat_register":
      return "注册接口";
    case "im_before_send":
      return "IM 发消息回调";
    case "im_prev_friend_add":
      return "IM 加好友回调";
    case "im_prev_create_group":
      return "IM 建群回调";
    case "im_after_create_group":
      return "IM 建群后回调";
    case "im_before_send_group_msg":
      return "IM 群发消息前兜底";
    case "challenge_create":
      return "分账自动建群";
    default:
      return source || "未知";
  }
}

onMounted(fetchRows);
</script>

<template>
  <div class="page">
    <div class="card">
      <div class="card-header">
        <div>
          <h1 class="page-title">静默时段命中日志</h1>
          <p class="page-sub">记录用户在静默时段触发限制动作的命中详情，便于排查和回溯。</p>
        </div>
      </div>

      <div class="card-toolbar">
        <div class="audit-filters">
          <el-input
            v-model="filters.keyword"
            placeholder="手机号 / IM 用户 ID / 目标账号 / 群组 / 原因"
            clearable
            class="filter-keyword"
            @keyup.enter="handleSearch"
          />
          <el-select
            v-model="filters.action"
            placeholder="动作"
            clearable
            class="filter-select"
          >
            <el-option label="注册" value="register" />
            <el-option label="发消息" value="send_message" />
            <el-option label="创建群组" value="create_group" />
            <el-option label="自动解散群" value="auto_dismiss_group" />
            <el-option label="加好友" value="add_friend" />
          </el-select>
          <el-select
            v-model="filters.source"
            placeholder="来源"
            clearable
            class="filter-select"
          >
            <el-option label="注册接口" value="chat_register" />
            <el-option label="IM 发消息回调" value="im_before_send" />
            <el-option label="IM 加好友回调" value="im_prev_friend_add" />
            <el-option label="IM 建群回调" value="im_prev_create_group" />
            <el-option label="IM 建群后回调" value="im_after_create_group" />
            <el-option label="IM 群发消息前兜底" value="im_before_send_group_msg" />
            <el-option label="分账自动建群" value="challenge_create" />
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
          <article
            v-for="row in rows"
            :key="row.id"
            class="audit-mobile-card"
          >
            <div class="audit-mobile-card-top">
              <span class="audit-mobile-time">{{ row.createdAt }}</span>
              <div class="audit-mobile-badges">
                <span class="status-badge status-danger">{{ actionLabel(row.action) }}</span>
              </div>
            </div>
            <div class="audit-mobile-title">{{ sourceLabel(row.source) }}</div>
            <div class="audit-mobile-desc">{{ row.reason }}</div>
            <dl class="audit-mobile-meta">
              <div>
                <dt>IM 用户</dt>
                <dd>{{ row.imUserId || "—" }}</dd>
              </div>
              <div>
                <dt>手机号</dt>
                <dd>{{ row.phone || "—" }}</dd>
              </div>
              <div>
                <dt>目标账号</dt>
                <dd>{{ row.targetAccount || "—" }}</dd>
              </div>
              <div>
                <dt>目标群组</dt>
                <dd>{{ row.targetGroupId || "—" }}</dd>
              </div>
            </dl>
          </article>
        </div>
      </template>

      <div v-else class="table-scroll-x page-table-wrap">
        <el-table :data="rows" v-loading="loading" stripe size="small" style="width: 100%">
          <el-table-column label="时间" min-width="160">
            <template #default="{ row }">
              <span class="cell-code cell-xs">{{ row.createdAt }}</span>
            </template>
          </el-table-column>
          <el-table-column label="动作" width="110">
            <template #default="{ row }">
              <span class="status-badge status-danger">{{ actionLabel(row.action) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="来源" min-width="140">
            <template #default="{ row }">
              <span class="cell-username">{{ sourceLabel(row.source) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="IM 用户 ID" min-width="140">
            <template #default="{ row }">
              <span class="cell-code">{{ row.imUserId || "—" }}</span>
            </template>
          </el-table-column>
          <el-table-column label="手机号" min-width="120">
            <template #default="{ row }">
              <span class="cell-code">{{ row.phone || "—" }}</span>
            </template>
          </el-table-column>
          <el-table-column label="目标账号" min-width="140">
            <template #default="{ row }">
              <span class="cell-code">{{ row.targetAccount || "—" }}</span>
            </template>
          </el-table-column>
          <el-table-column label="目标群组" min-width="140">
            <template #default="{ row }">
              <span class="cell-code">{{ row.targetGroupId || "—" }}</span>
            </template>
          </el-table-column>
          <el-table-column label="IP" min-width="120">
            <template #default="{ row }">
              <span class="cell-code">{{ row.ipAddress || "—" }}</span>
            </template>
          </el-table-column>
          <el-table-column label="原因" min-width="280">
            <template #default="{ row }">
              <span class="cell-muted">{{ row.reason || "—" }}</span>
            </template>
          </el-table-column>
        </el-table>
      </div>

      <div v-if="!loading && rows.length === 0" class="empty-state">
        当前暂无静默时段命中日志
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
          @size-change="(size: number) => { pagination.pageSize = size; pagination.page = 1; fetchRows() }"
          @current-change="fetchRows"
        />
      </div>
    </div>
  </div>
</template>

<style scoped>
.card-toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  border-bottom: 1px solid #f0f0f0;
}

.page-table-wrap {
  border-bottom: 1px solid #f0f0f0;
}

.audit-filters {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
  flex: 1;
}

.toolbar-right {
  display: flex;
  align-items: center;
  gap: 12px;
  white-space: nowrap;
}

.filter-keyword {
  width: 280px;
}

.filter-select {
  width: 160px;
}

.total-count {
  font-size: 12px;
  color: #94a3b8;
}

.cell-username {
  font-size: 13px;
  font-weight: 500;
  color: #1e293b;
}

.cell-code,
.cell-xs,
.cell-muted {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
}

.cell-xs {
  color: #94a3b8;
  font-size: 11px;
}

.cell-muted {
  color: #64748b;
}

.status-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 68px;
  height: 24px;
  padding: 0 10px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 500;
  border: 1px solid transparent;
}

.status-danger {
  color: #c2410c;
  border-color: #fdba74;
  background: #fff7ed;
}

.audit-mobile-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 12px;
}

.audit-mobile-card {
  padding: 14px;
  border-radius: 14px;
  border: 1px solid #edf2f7;
  background: #fff;
  box-shadow: 0 8px 18px rgba(15, 23, 42, 0.04);
}

.audit-mobile-card-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.audit-mobile-time {
  font-size: 12px;
  color: #94a3b8;
}

.audit-mobile-badges {
  display: flex;
  gap: 8px;
}

.audit-mobile-title {
  margin-top: 10px;
  font-size: 15px;
  font-weight: 700;
  color: #0f172a;
}

.audit-mobile-desc {
  margin-top: 8px;
  color: #64748b;
  font-size: 13px;
  line-height: 1.6;
}

.audit-mobile-meta {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px 12px;
  margin-top: 12px;
}

.audit-mobile-meta dt {
  font-size: 11px;
  color: #94a3b8;
}

.audit-mobile-meta dd {
  margin: 4px 0 0;
  font-size: 12px;
  color: #0f172a;
  word-break: break-all;
}

.empty-state {
  padding: 24px;
  text-align: center;
  color: #94a3b8;
}

.pagination-bar {
  padding: 14px 18px;
  display: flex;
  justify-content: flex-end;
}

@media (max-width: 767px) {
  .card-toolbar,
  .pagination-bar {
    flex-direction: column;
    align-items: stretch;
  }

  .filter-keyword,
  .filter-select {
    width: 100%;
  }

  .total-count {
    margin-left: 0;
  }
}
</style>
