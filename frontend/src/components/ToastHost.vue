<script setup lang="ts">
import type { Toast } from '../composables/useChatlog';

defineProps<{ toasts: Toast[] }>();
</script>

<template>
	<div class="toastHost">
		<div v-for="t in toasts" :key="t.id" class="toast">
			<div class="toastTitle">{{ t.title }}</div>
			<div class="toastMsg">{{ t.message }}</div>
		</div>
	</div>
</template>
