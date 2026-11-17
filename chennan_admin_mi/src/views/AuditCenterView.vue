<script setup lang="ts">
import { ref } from 'vue'
import AccessAuditPanel from '@/components/audit/AccessAuditPanel.vue'
import LoginAuditPanel from '@/components/audit/LoginAuditPanel.vue'

const activeTab = ref<'access' | 'login'>('access')
</script>

<template>
  <div class="page">
    <div class="card">
      <div class="card-header">
        <div>
          <h1 class="page-title">审计中心</h1>
          <p class="page-sub">查看后台系统用户访问、非法访问以及管理员与助理的登录尝试记录。</p>
        </div>
      </div>

      <el-tabs v-model="activeTab" class="audit-tabs">
        <el-tab-pane label="访问审计" name="access">
          <AccessAuditPanel />
        </el-tab-pane>
        <el-tab-pane label="登录审计" name="login">
          <LoginAuditPanel />
        </el-tab-pane>
      </el-tabs>
    </div>
  </div>
</template>

<style scoped>
.audit-tabs :deep(.el-tabs__header) {
  margin-bottom: 18px;
}
</style>
