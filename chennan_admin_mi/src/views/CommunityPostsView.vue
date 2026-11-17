<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import CommunityCommentsDialog from '@/components/community/CommunityCommentsDialog.vue'
import {
  deleteCommunityPost,
  getCommunityPostList,
  updateCommunityPost,
} from '@/api/community'
import type { AdminCommunityPost } from '@/types'
import { useAuthStore } from '@/stores/auth'

const authStore = useAuthStore()
const loading = ref(false)
const saving = ref(false)
const rows = ref<AdminCommunityPost[]>([])
const total = ref(0)
const commentsVisible = ref(false)
const currentPost = ref<AdminCommunityPost | null>(null)
const editVisible = ref(false)

const filters = reactive({
  keyword: '',
  postType: '',
})

const pagination = reactive({
  page: 1,
  pageSize: 20,
})

const editForm = reactive({
  postType: '',
  title: '',
  content: '',
})

const pageSizeOptions = [10, 20, 50, 100]
const pageSubText = computed(() => {
  if (authStore.isSuperAdmin) return '管理全部省钱社区内容与评论'
  if (authStore.isManager) return '管理我归属范围内的省钱社区内容与评论'
  return '管理我名下用户的省钱社区内容与评论'
})

const postTypeOptions = [
  { value: '', label: '全部类型' },
  { value: 'checkin_share', label: '优惠分享' },
  { value: 'challenge_recruit', label: '分账讨论' },
  { value: 'experience_share', label: '省钱攻略' },
]

function postTypeLabel(postType: string) {
  return postTypeOptions.find(item => item.value === postType)?.label || postType
}

async function fetchPosts() {
  loading.value = true
  try {
    const res = await getCommunityPostList({
      page: pagination.page,
      pageSize: pagination.pageSize,
      keyword: filters.keyword || undefined,
      postType: filters.postType || undefined,
    })
    rows.value = res.data.data?.list ?? []
    total.value = res.data.data?.total ?? 0
  } catch (error: any) {
    ElMessage.error(error.message || '获取省钱社区内容失败')
  } finally {
    loading.value = false
  }
}

function handleSearch() {
  pagination.page = 1
  void fetchPosts()
}

function resetFilters() {
  filters.keyword = ''
  filters.postType = ''
  handleSearch()
}

function openComments(row: AdminCommunityPost) {
  currentPost.value = row
  commentsVisible.value = true
}

function openEdit(row: AdminCommunityPost) {
  currentPost.value = row
  editForm.postType = row.postType
  editForm.title = row.title
  editForm.content = row.content
  editVisible.value = true
}

async function handleSaveEdit() {
  if (!currentPost.value) return
  if (!editForm.content.trim()) {
    ElMessage.warning('内容不能为空')
    return
  }
  saving.value = true
  try {
    await updateCommunityPost(currentPost.value.id, {
      postType: editForm.postType,
      title: editForm.title,
      content: editForm.content,
    })
    editVisible.value = false
    ElMessage.success('内容已更新')
    await fetchPosts()
  } catch (error: any) {
    ElMessage.error(error.message || '更新内容失败')
  } finally {
    saving.value = false
  }
}

async function handleDelete(row: AdminCommunityPost) {
  try {
    await ElMessageBox.confirm('确认物理删除该内容及其评论、点赞记录吗？此操作不可恢复。', '删除内容', {
      type: 'warning',
      confirmButtonText: '删除',
      cancelButtonText: '取消',
    })
  } catch {
    return
  }

  try {
    await deleteCommunityPost(row.id)
    ElMessage.success('内容已删除')
    await fetchPosts()
  } catch (error: any) {
    ElMessage.error(error.message || '删除内容失败')
  }
}

onMounted(() => {
  void fetchPosts()
})
</script>

<template>
  <div class="page">
    <div class="card">
      <div class="card-header">
        <div>
          <h1 class="page-title">省钱社区内容</h1>
          <p class="page-sub">{{ pageSubText }}</p>
        </div>
      </div>

      <div class="card-toolbar">
        <div class="toolbar-left">
          <el-input
            v-model="filters.keyword"
            placeholder="搜索作者 / 信号 / 标题 / 内容"
            clearable
            class="filter-keyword"
            @keyup.enter="handleSearch"
          />
          <el-select v-model="filters.postType" class="filter-select">
            <el-option v-for="item in postTypeOptions" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
          <el-button type="primary" @click="handleSearch">筛选</el-button>
          <el-button @click="resetFilters">重置</el-button>
        </div>
        <div class="toolbar-right">
          <span class="total-count">共 {{ total }} 条</span>
        </div>
      </div>

      <el-table :data="rows" v-loading="loading" stripe size="small" style="width: 100%">
        <el-table-column label="序号" width="70" align="center">
          <template #default="{ $index }">
            <span class="cell-xs">{{ (pagination.page - 1) * pagination.pageSize + $index + 1 }}</span>
          </template>
        </el-table-column>
        <el-table-column label="作者" min-width="180">
          <template #default="{ row }">
            <div class="author-cell">
              <strong>{{ row.author.nickname || row.author.imUserID }}</strong>
              <span class="author-meta">信号：{{ row.author.userNo || row.author.wananID || '—' }}</span>
              <span class="author-meta">归属：{{ row.author.ownerName || '—' }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="类型" width="110">
          <template #default="{ row }">
            <span class="category-badge">{{ postTypeLabel(row.contentType || row.postType) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="标题" min-width="180">
          <template #default="{ row }">
            <div class="post-title">{{ row.title || '—' }}</div>
          </template>
        </el-table-column>
        <el-table-column label="内容" min-width="260">
          <template #default="{ row }">
            <div class="post-content">{{ row.content }}</div>
          </template>
        </el-table-column>
        <el-table-column label="互动" width="110">
          <template #default="{ row }">
            <div class="metric-stack">
              <span>评论 {{ row.commentCount }}</span>
              <span>点赞 {{ row.likeCount }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="发布时间" width="160">
          <template #default="{ row }">
            <span class="cell-xs">{{ row.createdAt }}</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="210" fixed="right" align="center">
          <template #default="{ row }">
            <div class="table-actions">
              <el-button size="small" plain @click="openComments(row)">评论</el-button>
              <el-button size="small" type="primary" plain @click="openEdit(row)">编辑</el-button>
              <el-button size="small" type="danger" plain @click="handleDelete(row)">删除</el-button>
            </div>
          </template>
        </el-table-column>
      </el-table>

      <div v-if="!loading && rows.length === 0" class="empty-state">
        当前暂无省钱社区内容
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
          @size-change="(size: number) => { pagination.pageSize = size; pagination.page = 1; fetchPosts() }"
          @current-change="fetchPosts"
        />
      </div>
    </div>

    <CommunityCommentsDialog
      v-model:visible="commentsVisible"
      :post="currentPost"
      @updated="fetchPosts"
    />

    <el-dialog
      v-model="editVisible"
      title="编辑内容"
      width="620px"
      destroy-on-close
    >
      <el-form label-position="top">
        <el-form-item label="内容类型">
          <el-select v-model="editForm.postType">
            <el-option
              v-for="item in postTypeOptions.filter(option => option.value)"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="标题">
          <el-input v-model="editForm.title" maxlength="64" show-word-limit />
        </el-form-item>
        <el-form-item label="内容">
          <el-input v-model="editForm.content" type="textarea" :rows="8" maxlength="2000" show-word-limit />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="handleSaveEdit">保存</el-button>
      </template>
    </el-dialog>
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

.toolbar-left {
  display: flex;
  align-items: center;
  gap: 12px;
  flex: 1;
  flex-wrap: wrap;
}

.toolbar-right {
  white-space: nowrap;
}

.filter-keyword {
  width: 300px;
}

.filter-select {
  width: 160px;
}

.total-count,
.cell-xs {
  font-size: 12px;
  color: #94a3b8;
}

.author-cell {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.author-meta {
  font-size: 12px;
  color: #94a3b8;
}

.category-badge {
  background: #f1f5f9;
  color: #475569;
  font-size: 12px;
  padding: 2px 8px;
  border-radius: 4px;
  font-weight: 500;
}

.post-title {
  font-size: 13px;
  font-weight: 600;
  color: #1e293b;
}

.post-content {
  line-height: 1.6;
  color: #475569;
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.metric-stack {
  display: flex;
  flex-direction: column;
  gap: 4px;
  color: #64748b;
  font-size: 12px;
}

.table-actions {
  display: inline-flex;
  gap: 8px;
}
</style>
