<script setup lang="ts">
import { reactive, ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { NAlert, NButton, NForm, NFormItem, NInput, useMessage } from 'naive-ui'
import { ApiError } from '@/api/client'
import { useUserStore } from '@/stores/user'

const router = useRouter()
const userStore = useUserStore()
const message = useMessage()

const form = reactive({ username: '', email: '', password: '' })
const submitting = ref(false)
const errorMessage = ref('')
/** 后端返回的字段级错误，如 { username: '该用户名已被占用' } */
const fieldErrors = ref<Record<string, string>>({})

function feedback(field: string): string | undefined {
  return fieldErrors.value[field]
}

async function handleSubmit(): Promise<void> {
  submitting.value = true
  errorMessage.value = ''
  fieldErrors.value = {}

  try {
    await userStore.register({
      username: form.username,
      email: form.email,
      password: form.password,
    })
    message.success('注册成功，已自动登录')
    await router.replace({ name: 'dashboard' })
  } catch (error) {
    if (error instanceof ApiError) {
      errorMessage.value = error.message
      // 422 校验失败时把每个字段的错误挂到对应输入框上
      fieldErrors.value = error.details ?? {}
    } else {
      errorMessage.value = '注册失败，请稍后重试'
    }
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="auth-page">
    <div class="auth-card">
      <h1 class="auth-card__title">创建账号</h1>
      <p class="auth-card__subtitle">开始积累你的算法笔记</p>

      <n-alert v-if="errorMessage" type="error" :show-icon="false" style="margin-bottom: 16px">
        {{ errorMessage }}
      </n-alert>

      <n-form :show-label="false" @submit.prevent="handleSubmit">
        <n-form-item :feedback="feedback('username')" :validation-status="feedback('username') ? 'error' : undefined">
          <n-input
            v-model:value="form.username"
            placeholder="用户名（3-50 位字母、数字、下划线）"
            :disabled="submitting"
            autocomplete="username"
          />
        </n-form-item>

        <n-form-item :feedback="feedback('email')" :validation-status="feedback('email') ? 'error' : undefined">
          <n-input
            v-model:value="form.email"
            placeholder="邮箱"
            :disabled="submitting"
            autocomplete="email"
          />
        </n-form-item>

        <n-form-item :feedback="feedback('password')" :validation-status="feedback('password') ? 'error' : undefined">
          <n-input
            v-model:value="form.password"
            type="password"
            show-password-on="click"
            placeholder="密码（至少 8 位）"
            :disabled="submitting"
            autocomplete="new-password"
            @keyup.enter="handleSubmit"
          />
        </n-form-item>

        <n-button type="primary" block :loading="submitting" @click="handleSubmit">
          注册
        </n-button>
      </n-form>

      <p class="auth-card__footer">
        已有账号？
        <RouterLink :to="{ name: 'login' }">去登录</RouterLink>
      </p>
    </div>
  </div>
</template>
