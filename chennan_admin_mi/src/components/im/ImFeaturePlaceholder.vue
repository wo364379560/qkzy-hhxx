<script setup lang="ts">
const props = defineProps<{
  title: string
  description: string
  nextStep: string
}>()
</script>

<template>
  <div class="page">
    <div class="card placeholder-card">
      <div class="placeholder-icon">
        <svg width="28" height="28" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.6"
            d="M8 10h.01M12 10h.01M16 10h.01M9 16H5l-2 2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-4l-3 3v-3H9z" />
        </svg>
      </div>

      <div class="placeholder-copy">
        <h1 class="placeholder-title">{{ props.title }}</h1>
        <p class="placeholder-desc">{{ props.description }}</p>
      </div>

      <div class="placeholder-state">
        <span class="state-badge">暂未开放</span>
        <p class="state-text">
          当前页面已移除 mock 数据交互，避免将未接真实后端的能力误判为可运营功能。
        </p>
      </div>

      <div class="placeholder-panel">
        <p class="panel-title">下一步接入要求</p>
        <p class="panel-text">{{ props.nextStep }}</p>
      </div>
    </div>
  </div>
</template>

<style scoped>
.page {
  padding: 24px 28px;
}

.placeholder-card {
  display: flex;
  flex-direction: column;
  gap: 20px;
  padding: 28px;
  max-width: 780px;
}

.placeholder-icon {
  width: 52px;
  height: 52px;
  border-radius: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #1677ff;
  background: #e6f4ff;
}

.placeholder-copy {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.placeholder-title {
  margin: 0;
  font-size: 24px;
  line-height: 1.2;
  font-weight: 700;
  color: #1e293b;
}

.placeholder-desc {
  margin: 0;
  font-size: 14px;
  line-height: 1.7;
  color: #64748b;
}

.placeholder-state {
  padding: 16px 18px;
  border-radius: 12px;
  border: 1px solid #ffe58f;
  background: #fffbe6;
}

.state-badge {
  display: inline-flex;
  align-items: center;
  min-height: 24px;
  padding: 0 10px;
  border-radius: 999px;
  background: #faad14;
  color: #fff;
  font-size: 12px;
  font-weight: 600;
}

.state-text {
  margin: 10px 0 0;
  font-size: 13px;
  line-height: 1.7;
  color: #8c6d1f;
}

.placeholder-panel {
  padding: 16px 18px;
  border-radius: 12px;
  border: 1px solid #e2e8f0;
  background: #f8fafc;
}

.panel-title {
  margin: 0 0 8px;
  font-size: 13px;
  font-weight: 600;
  color: #334155;
}

.panel-text {
  margin: 0;
  font-size: 13px;
  line-height: 1.7;
  color: #64748b;
}
</style>
