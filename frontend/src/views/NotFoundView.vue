<script setup lang="ts">
import { RouterLink } from 'vue-router'
</script>

<template>
  <div class="auth-page">
    <div class="auth-card" style="text-align: center">
      <h1 class="auth-card__title">404</h1>
      <p class="auth-card__subtitle">这个页面不存在</p>
      <RouterLink :to="{ name: 'dashboard' }">返回首页</RouterLink>
    </div>
  </div>
</template>
