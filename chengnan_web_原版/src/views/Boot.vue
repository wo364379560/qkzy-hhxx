<script setup lang="ts">
</script>

<template>
  <div class="boot-view">
    <div class="boot-view-card">
      <div class="boot-view-spinner" />
      <p class="boot-view-text">正在恢复会话...</p>
    </div>
  </div>
</template>

<style scoped lang="scss">
.boot-view {
  min-height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  background:
    radial-gradient(circle at top left, rgba(67, 97, 238, 0.12), transparent 32%),
    #f6f8fc;
}

.boot-view-card {
  padding: 28px 32px;
  border-radius: 20px;
  background: #ffffff;
  box-shadow: 0 16px 48px rgba(15, 23, 42, 0.08);
  text-align: center;
}

.boot-view-spinner {
  width: 32px;
  height: 32px;
  margin: 0 auto 12px;
  border: 3px solid rgba(67, 97, 238, 0.16);
  border-top-color: #4361ee;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

.boot-view-text {
  margin: 0;
  color: #475569;
  font-size: 14px;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
