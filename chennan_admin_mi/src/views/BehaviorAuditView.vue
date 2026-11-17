<script setup lang="ts">
import AccessAuditPanel from '@/components/audit/AccessAuditPanel.vue'
</script>

<template>
  <div class="page">
    <div class="card">
      <div class="card-header">
        <div>
          <h1 class="page-title">行为审计</h1>
          <p class="page-sub">查看后台系统用户访问记录，并支持筛选系统用户访问与非法访问。</p>
        </div>
      </div>

      <AccessAuditPanel />
    </div>
  </div>
</template>
