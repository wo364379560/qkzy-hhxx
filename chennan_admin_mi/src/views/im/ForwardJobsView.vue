<template>
  <section class="forward-jobs">
    <div class="toolbar"><h2>转发任务</h2><el-button :loading="loading" @click="refresh">刷新</el-button></div>
    <el-alert v-if="error" :title="error" type="error" :closable="false" />
    <p>显示当前权限范围内的服务端转发任务。结果未知的消息需先核对接收端。</p>
    <el-table :data="rows" v-loading="loading">
      <el-table-column prop="sender" label="发送账号" min-width="140" />
      <el-table-column prop="progress.request_id" label="任务编号" min-width="230" />
      <el-table-column prop="progress.state" label="状态" width="110" />
      <el-table-column prop="progress.total" label="总数" width="90" />
      <el-table-column prop="progress.success" label="成功" width="80" />
      <el-table-column prop="progress.failed" label="失败" width="80" />
      <el-table-column prop="progress.unknown" label="待确认" width="90" />
      <el-table-column prop="progress.pending" label="待处理" width="90" />
    </el-table>
    <el-button v-if="nextCursor" :loading="loading" @click="load(nextCursor)">下一页</el-button>
  </section>
</template>
<script setup lang="ts">
import { onMounted, ref } from 'vue'
import request from '@/api/request'
const rows = ref<any[]>([]), nextCursor = ref(''), loading = ref(false), error = ref('')
let revision = 0
async function load(cursor = '') {
  const epoch = ++revision
  loading.value = true
  error.value = ''
  try {
    const result = await request.get('/forward-jobs', { params: { cursor } })
    if (epoch !== revision) return
    rows.value = result.data.data.items
    nextCursor.value = result.data.data.next_cursor
  } catch (failure) { if (epoch === revision) error.value = failure instanceof Error ? failure.message : '任务查询失败' }
  finally { if (epoch === revision) loading.value = false }
}
function refresh() { void load() }
onMounted(refresh)
</script>
<style scoped>.forward-jobs{padding:24px}.toolbar{display:flex;justify-content:space-between;align-items:center}.forward-jobs p{color:#606266}</style>
