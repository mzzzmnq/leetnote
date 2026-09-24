<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { NAlert, NButton, NForm, NFormItem, NInput, NTag, useDialog, useMessage } from 'naive-ui'
import { ApiError } from '@/api/client'
import { changePassword, fetchOAuthAccounts, unlinkGitHub, updateProfile } from '@/api/users'
import type { OAuthAccount } from '@/api/types'
import { useUserStore } from '@/stores/user'

const userStore = useUserStore()
const message = useMessage()
const dialog = useDialog()

const accounts = ref<OAuthAccount[]>([])
const savingProfile = ref(false)
const savingPassword = ref(false)
const errorMessage = ref('')

const profileForm = reactive({
  bio: userStore.user?.bio ?? '',
})

const passwordForm = reactive({
  old_password: '',
  new_password: '',
})

async function loadAccounts(): Promise<void> {
  try {
    accounts.value = await fetchOAuthAccounts()
  } catch {
    // 未绑定任何第三方账号时后端返回空数组，正常情况不会走到这
    accounts.value = []
  }
}

async function handleSaveProfile(): Promise<void> {
  savingProfile.value = true
  errorMessage.value = ''
  try {
    await updateProfile({ bio: profileForm.bio })
    await userStore.fetchMe()
    message.success('已保存')
  } catch (error) {
    errorMessage.value = error instanceof ApiError ? error.message : '保存失败'
  } finally {
    savingProfile.value = false
  }
}

async function handleChangePassword(): Promise<void> {
  savingPassword.value = true
  errorMessage.value = ''
  try {
    // 注意：OAuth 用户没有密码，后端允许不传旧密码直接设置
    await changePassword({
      old_password: passwordForm.old_password,
      new_password: passwordForm.new_password,
    })
    message.success('密码已更新')
    passwordForm.old_password = ''
    passwordForm.new_password = ''
  } catch (error) {
    errorMessage.value = error instanceof ApiError ? error.message : '修改密码失败'
  } finally {
    savingPassword.value = false
  }
}

function handleUnlinkGitHub(): void {
  dialog.warning({
    title: '解绑 GitHub',
    content: '解绑后将无法用 GitHub 登录。若账号没有密码，需先设置密码。',
    positiveText: '确认解绑',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await unlinkGitHub()
        message.success('已解绑')
        await loadAccounts()
      } catch (error) {
        message.error(error instanceof ApiError ? error.message : '解绑失败')
      }
    },
  })
}

onMounted(() => {
  void loadAccounts()
})
</script>

<template>
  <div class="page">
    <h1 class="page__title">账号设置</h1>
    <p class="page__subtitle">个人资料与第三方账号绑定</p>

    <n-alert v-if="errorMessage" type="error" style="margin-bottom: 16px">
      {{ errorMessage }}
    </n-alert>

    <section class="settings-block">
      <h2 class="settings-block__title">个人资料</h2>

      <n-form label-placement="top" :show-feedback="false">
        <n-form-item label="用户名">
          <n-input :value="userStore.user?.username" disabled />
        </n-form-item>
        <n-form-item label="邮箱">
          <n-input :value="userStore.user?.email" disabled />
        </n-form-item>
        <n-form-item label="简介">
          <n-input
            v-model:value="profileForm.bio"
            type="textarea"
            :rows="2"
            maxlength="200"
            show-count
            placeholder="一句话介绍自己"
          />
        </n-form-item>
        <n-button type="primary" :loading="savingProfile" @click="handleSaveProfile">
          保存资料
        </n-button>
      </n-form>
    </section>

    <section class="settings-block">
      <h2 class="settings-block__title">修改密码</h2>
      <p class="settings-block__hint">
        通过 GitHub 注册的账号没有密码，此时留空「当前密码」即可直接设置。
      </p>

      <n-form label-placement="top" :show-feedback="false">
        <n-form-item label="当前密码">
          <n-input
            v-model:value="passwordForm.old_password"
            type="password"
            show-password-on="click"
            placeholder="没有密码可留空"
          />
        </n-form-item>
        <n-form-item label="新密码">
          <n-input
            v-model:value="passwordForm.new_password"
            type="password"
            show-password-on="click"
            placeholder="至少 8 位"
          />
        </n-form-item>
        <n-button :loading="savingPassword" @click="handleChangePassword">更新密码</n-button>
      </n-form>
    </section>

    <section class="settings-block">
      <h2 class="settings-block__title">第三方账号</h2>

      <div v-if="accounts.length === 0" class="settings-block__hint">
        尚未绑定任何第三方账号
      </div>

      <div v-for="account in accounts" :key="account.provider" class="oauth-row">
        <div class="oauth-row__info">
          <n-tag size="small" type="info">{{ account.provider }}</n-tag>
          <span>{{ account.provider_login }}</span>
        </div>
        <n-button size="small" quaternary type="error" @click="handleUnlinkGitHub">
          解绑
        </n-button>
      </div>
    </section>
  </div>
</template>

<style scoped>
.settings-block {
  max-width: 520px;
  padding: 24px;
  margin-bottom: 20px;
  background: #fff;
  border: 1px solid var(--ln-border);
  border-radius: 12px;
}

.settings-block__title {
  margin: 0 0 4px;
  font-size: 16px;
}

.settings-block__hint {
  margin: 0 0 16px;
  font-size: 13px;
  color: var(--ln-text-muted);
}

.oauth-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 0;
  border-top: 1px solid var(--ln-border);
}

.oauth-row__info {
  display: flex;
  gap: 10px;
  align-items: center;
  font-size: 14px;
}
</style>
