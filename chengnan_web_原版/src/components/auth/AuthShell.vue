<script setup lang="ts">
defineProps<{
  title: string;
  subtitle?: string;
}>();
</script>

<template>
  <div class="auth-shell">
    <div class="auth-shell-panel">
      <header class="auth-shell-header">
        <p class="auth-shell-brand">睿之禾</p>
        <h1 class="auth-shell-title">{{ title }}</h1>
        <p
          v-if="subtitle"
          class="auth-shell-subtitle"
        >
          {{ subtitle }}
        </p>
      </header>
      <main class="auth-shell-body">
        <slot />
      </main>
      <footer
        v-if="$slots.footer"
        class="auth-shell-footer"
      >
        <slot name="footer" />
      </footer>
    </div>
  </div>
</template>

<style scoped lang="scss">
.auth-shell {
  min-height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 32px 16px;
  background:
    radial-gradient(circle at top left, rgba(67, 97, 238, 0.12), transparent 36%),
    radial-gradient(circle at bottom right, rgba(37, 99, 235, 0.12), transparent 30%),
    #f6f8fc;
}

.auth-shell-panel {
  width: min(100%, 460px);
  padding: 32px 28px;
  border-radius: 24px;
  background: #ffffff;
  box-shadow: 0 20px 60px rgba(15, 23, 42, 0.08);
}

.auth-shell-header {
  margin-bottom: 24px;
}

.auth-shell-brand {
  margin: 0 0 12px;
  color: #4361ee;
  font-size: 14px;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.auth-shell-title {
  margin: 0;
  color: #0f172a;
  font-size: 28px;
  font-weight: 700;
  line-height: 1.15;
}

.auth-shell-subtitle {
  margin: 10px 0 0;
  color: #64748b;
  font-size: 14px;
  line-height: 1.6;
}

.auth-shell-body {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.auth-shell-footer {
  margin-top: 24px;
}
</style>
