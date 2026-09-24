<script setup lang="ts">
import { reactive, ref } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { NAlert, NButton, NForm, NFormItem, NInput, useMessage } from 'naive-ui'
import { fetchGitHubAuthorizeURL } from '@/api/auth'
import { ApiError } from '@/api/client'
import { useUserStore } from '@/stores/user'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()
const message = useMessage()

const form = reactive({ login: '', password: '' })
const submitting = ref(false)
const errorMessage = ref('')

/** 登录成功后要跳回的原页面（由路由守卫带上） */
function redirectTarget(): string {
  const raw = route.query.redirect
  return typeof raw === 'string' ? raw : '/'
}

async function handleSubmit(): Promise<void> {
  if (!form.login || !form.password) {
    errorMessage.value = '请填写用户名和密码'
    return
  }

  submitting.value = true
  errorMessage.value = ''

  try {
    await userStore.login({ login: form.login, password: form.password })
    message.success('登录成功')
    await router.replace(redirectTarget())
  } catch (error) {
    errorMessage.value = error instanceof ApiError ? error.message : '登录失败，请稍后重试'
  } finally {
    submitting.value = false
  }
}

async function handleGitHubLogin(): Promise<void> {
  submitting.value = true
  errorMessage.value = ''

  try {
    const raw = route.query.redirect
    const url = await fetchGitHubAuthorizeURL(typeof raw === 'string' ? raw : undefined)
    // 整页跳转到 GitHub，不再需要恢复按钮状态
    window.location.href = url
  } catch (error) {
    errorMessage.value = error instanceof ApiError ? error.message : '无法发起 GitHub 登录'
    submitting.value = false
  }
}
</script>

<template>
  <div class="auth-page">
    <div class="auth-card">
      <h1 class="auth-card__title">登录 LeetNote</h1>
      <p class="auth-card__subtitle">记录你的每一道题</p>

      <n-alert v-if="errorMessage" type="error" :show-icon="false" style="margin-bottom: 16px">
        {{ errorMessage }}
      </n-alert>

      <n-form :show-label="false" @submit.prevent="handleSubmit">
        <n-form-item>
          <n-input
            v-model:value="form.login"
            placeholder="用户名或邮箱"
            :disabled="submitting"
            autocomplete="username"
            @keyup.enter="handleSubmit"
          />
        </n-form-item>

        <n-form-item>
          <n-input
            v-model:value="form.password"
            type="password"
            show-password-on="click"
            placeholder="密码"
            :disabled="submitting"
            autocomplete="current-password"
            @keyup.enter="handleSubmit"
          />
        </n-form-item>

        <n-button type="primary" block :loading="submitting" @click="handleSubmit">
          登录
        </n-button>
      </n-form>

      <div class="divider">或</div>

      <n-button block :disabled="submitting" @click="handleGitHubLogin">
        使用 GitHub 登录
      </n-button>

      <p class="auth-card__footer">
        还没有账号？
        <RouterLink :to="{ name: 'register', query: route.query }">立即注册</RouterLink>
      </p>
    </div>
  </div>
</template>
