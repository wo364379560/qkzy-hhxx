<script setup lang="ts">
import LoginAuditPanel from '@/components/audit/LoginAuditPanel.vue'
</script>

<template>
  <div class="page">
    <div class="card">
      <div class="card-header">
        <div>
          <h1 class="page-title">登录审计</h1>
          <p class="page-sub">查看管理员与助理的后台登录尝试记录，包含人机验证、密码和 2FA 阶段。</p>
        </div>
      </div>

      <LoginAuditPanel />
    </div>
  </div>
</template>
