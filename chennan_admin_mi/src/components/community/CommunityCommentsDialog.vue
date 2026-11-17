<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  deleteCommunityComment,
  getCommunityComments,
  updateCommunityUserRestriction,
} from '@/api/community'
import type { AdminCommunityComment, AdminCommunityPost } from '@/types'

const props = defineProps<{
  visible: boolean
  post: AdminCommunityPost | null
}>()

const emit = defineEmits<{
  (event: 'update:visible', value: boolean): void
  (event: 'updated'): void
}>()

const loading = ref(false)
const actionLoading = ref<number | null>(null)
const comments = ref<AdminCommunityComment[]>([])

const dialogTitle = computed(() => props.post ? `评论管理 · ${props.post.title || '未命名内容'}` : '评论管理')

watch(
  () => [props.visible, props.post?.id] as const,
  async ([visible, postID]) => {
    if (!visible || !postID) return
    await fetchComments(postID)
  },
  { immediate: true },
)

async function fetchComments(postID: number) {
  loading.value = true
  try {
    const res = await getCommunityComments(postID)
    comments.value = res.data.data ?? []
  } catch (error: any) {
    ElMessage.error(error.message || '获取评论失败')
  } finally {
    loading.value = false
  }
}

function closeDialog() {
  emit('update:visible', false)
}

async function handleDelete(comment: AdminCommunityComment) {
  try {
    await ElMessageBox.confirm('确认删除这条评论？删除后不可恢复。', '删除评论', {
      type: 'warning',
      confirmButtonText: '删除',
      cancelButtonText: '取消',
    })
  } catch {
    return
  }

  actionLoading.value = comment.id
  try {
    await deleteCommunityComment(comment.id)
    comments.value = comments.value.filter(item => item.id !== comment.id)
    ElMessage.success('评论已删除')
    emit('updated')
  } catch (error: any) {
    ElMessage.error(error.message || '删除评论失败')
  } finally {
    actionLoading.value = null
  }
}

async function updateRestriction(comment: AdminCommunityComment, payload: { communityPostDisabled?: boolean; communityCommentDisabled?: boolean }, successText: string) {
  actionLoading.value = comment.id
  try {
    await updateCommunityUserRestriction(comment.author.userID, payload)
    comments.value = comments.value.map((item) => {
      if (item.author.userID !== comment.author.userID) {
        return item
      }
      return {
        ...item,
        author: {
          ...item.author,
          communityPostDisabled: payload.communityPostDisabled ?? item.author.communityPostDisabled,
          communityCommentDisabled: payload.communityCommentDisabled ?? item.author.communityCommentDisabled,
        },
      }
    })
    ElMessage.success(successText)
  } catch (error: any) {
    ElMessage.error(error.message || '更新限制失败')
  } finally {
    actionLoading.value = null
  }
}

function restrictionActions(comment: AdminCommunityComment) {
  return [
    comment.author.communityPostDisabled
      ? { label: '允许发帖', action: () => updateRestriction(comment, { communityPostDisabled: false }, '已恢复发帖权限') }
      : { label: '禁止发帖', action: () => updateRestriction(comment, { communityPostDisabled: true }, '已禁止发帖') },
    comment.author.communityCommentDisabled
      ? { label: '允许评论', action: () => updateRestriction(comment, { communityCommentDisabled: false }, '已恢复评论权限') }
      : { label: '禁止评论', action: () => updateRestriction(comment, { communityCommentDisabled: true }, '已禁止评论') },
  ]
}
</script>

<template>
  <el-dialog
    :model-value="visible"
    :title="dialogTitle"
    width="920px"
    destroy-on-close
    @close="closeDialog"
  >
    <div class="dialog-meta" v-if="post">
      <span class="meta-chip">评论数 {{ post.commentCount }}</span>
      <span class="meta-chip">点赞数 {{ post.likeCount }}</span>
      <span class="meta-chip">作者 {{ post.author.nickname || post.author.imUserID }}</span>
    </div>

    <el-table :data="comments" v-loading="loading" stripe size="small" style="width: 100%">
      <el-table-column label="评论人" min-width="180">
        <template #default="{ row }">
          <div class="author-cell">
            <strong>{{ row.author.nickname || row.author.imUserID }}</strong>
            <span class="author-meta">信号：{{ row.author.userNo || row.author.wananID || '—' }}</span>
            <span class="author-meta">归属：{{ row.author.ownerName || '—' }}</span>
          </div>
        </template>
      </el-table-column>
      <el-table-column label="评论内容" min-width="260">
        <template #default="{ row }">
          <div class="comment-content">{{ row.content }}</div>
        </template>
      </el-table-column>
      <el-table-column label="权限状态" width="180">
        <template #default="{ row }">
          <div class="restriction-tags">
            <el-tag :type="row.author.communityPostDisabled ? 'danger' : 'success'" size="small" effect="light">
              {{ row.author.communityPostDisabled ? '已禁发帖' : '可发帖' }}
            </el-tag>
            <el-tag :type="row.author.communityCommentDisabled ? 'danger' : 'success'" size="small" effect="light">
              {{ row.author.communityCommentDisabled ? '已禁评论' : '可评论' }}
            </el-tag>
          </div>
        </template>
      </el-table-column>
      <el-table-column label="发布时间" width="160">
        <template #default="{ row }">
          <span class="time-text">{{ row.createdAt }}</span>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="220" fixed="right" align="center">
        <template #default="{ row }">
          <div class="comment-actions">
            <el-dropdown trigger="click">
              <el-button size="small" plain :loading="actionLoading === row.id">限制操作</el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item
                    v-for="item in restrictionActions(row)"
                    :key="item.label"
                    @click="item.action"
                  >
                    {{ item.label }}
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
            <el-button size="small" type="danger" plain :loading="actionLoading === row.id" @click="handleDelete(row)">删除</el-button>
          </div>
        </template>
      </el-table-column>
    </el-table>

    <div v-if="!loading && comments.length === 0" class="empty-state">
      当前内容暂无评论
    </div>
  </el-dialog>
</template>

<style scoped>
.dialog-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 14px;
}

.meta-chip {
  padding: 4px 10px;
  border-radius: 999px;
  font-size: 12px;
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  color: #64748b;
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

.comment-content {
  white-space: pre-wrap;
  line-height: 1.6;
  color: #334155;
}

.restriction-tags {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.time-text {
  font-size: 12px;
  color: #64748b;
}

.comment-actions {
  display: inline-flex;
  gap: 8px;
  align-items: center;
}
</style>
