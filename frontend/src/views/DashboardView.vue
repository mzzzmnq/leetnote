<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { NAlert, NButton, NSpin, useMessage } from 'naive-ui'
import { ApiError } from '@/api/client'
import { fetchMe } from '@/api/users'
import { useUserStore } from '@/stores/user'

const userStore = useUserStore()
const message = useMessage()

const refreshing = ref(false)
const errorMessage = ref('')

const greeting = computed(() => {
  const hour = new Date().getHours()
  if (hour < 6) return '夜深了'
  if (hour < 12) return '早上好'
  if (hour < 18) return '下午好'
  return '晚上好'
})

/** 调一次 /users/me，顺便验证「带 Token 的受保护请求」整条链路是通的 */
async function refresh(): Promise<void> {
  refreshing.value = true
  errorMessage.value = ''
  try {
    await fetchMe()
    message.success('已刷新用户信息')
  } catch (error) {
    errorMessage.value = error instanceof ApiError ? error.message : '请求失败'
  } finally {
    refreshing.value = false
  }
}

onMounted(() => {
  void refresh()
})
</script>

<template>
  <div class="page">
    <h1 class="page__title">{{ greeting }}，{{ userStore.user?.username }}</h1>
    <p class="page__subtitle">后端连通性检查</p>

    <n-alert v-if="errorMessage" type="error" style="margin-bottom: 16px">
      {{ errorMessage }}
    </n-alert>

    <div class="placeholder-card">
      <p style="margin: 0 0 8px">✅ 认证链路已打通</p>
      <p style="margin: 0 0 20px; font-size: 13px">
        access_token 存在内存、refresh_token 在 httpOnly Cookie，
        刷新页面会自动恢复登录态
      </p>

      <n-spin v-if="refreshing" size="small" />
      <n-button v-else size="small" @click="refresh">重新请求 /users/me</n-button>
    </div>
  </div>
</template>
