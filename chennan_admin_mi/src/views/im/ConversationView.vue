<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import ImConversationManager from '@/components/im/ImConversationManager.vue'

const route = useRoute()
const router = useRouter()

const usersReturnPath = computed(() => {
  const from = Array.isArray(route.query.from)
    ? String(route.query.from[0] || '')
    : String(route.query.from || '')
  return from === '/users' || from.startsWith('/users?') ? from : ''
})

function returnToUsers() {
  if (usersReturnPath.value) {
    void router.push(usersReturnPath.value)
  }
}
</script>

<template>
  <div class="conversation-view">
    <button
      v-if="usersReturnPath"
      type="button"
      class="users-return-link"
      @click="returnToUsers"
    >
      ← 返回用户列表
    </button>
    <ImConversationManager />
  </div>
</template>

<style scoped>
.conversation-view {
  min-width: 0;
}

.users-return-link {
  display: inline-flex;
  align-items: center;
  margin: 20px 20px 0;
  padding: 0;
  border: 0;
  background: transparent;
  color: #1677ff;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
}

.users-return-link:hover {
  color: #0958d9;
}
</style>
