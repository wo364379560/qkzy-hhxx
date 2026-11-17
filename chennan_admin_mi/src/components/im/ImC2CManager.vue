<script setup lang="ts">
import { ref } from 'vue'
import ImC2CSendPanel from '@/components/im/ImC2CSendPanel.vue'
import ImC2CHistoryPanel from '@/components/im/ImC2CHistoryPanel.vue'
import { useResponsive } from '@/composables/useResponsive'

const activeTab = ref<'send' | 'history'>('send')
const { isMobile } = useResponsive()
</script>

<template>
  <div class="page">
    <el-tabs v-model="activeTab" :stretch="isMobile">
      <el-tab-pane label="批量发送" name="send">
        <ImC2CSendPanel />
      </el-tab-pane>
      <el-tab-pane label="消息记录" name="history">
        <ImC2CHistoryPanel />
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<style scoped>
.page {
  padding: var(--app-page-padding-y) var(--app-page-padding-x);
}

@media (max-width: 767px) {
  .page {
    padding: 0;
  }
}
</style>
