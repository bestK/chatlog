<script setup lang="ts">
import type { StatusPill } from '../composables/useChatlog';
import type { State } from '../wailsbridge';

withDefaults(
	defineProps<{
		page: string;
		state: State | null;
		statusPill: StatusPill;
	}>(),
	{},
);
defineEmits<{
	(e: 'refresh'): void;
}>();
</script>

<template>
	<div class="header-row">
		<div class="header-left">
			<div class="breadcrumbs">Chatlog / {{ page }}</div>
			<div class="title-row">
				<h1>{{ page }}</h1>
				<span v-if="state?.status === 'online'" class="status-badge">Connected</span>
				<span v-else class="status-badge" style="background: rgba(110, 111, 115, 0.1); color: var(--text-tertiary); border-color: rgba(110, 111, 115, 0.2);">Disconnected</span>
			</div>
		</div>
		<div class="actions">
			<button type="button" class="btn btn-secondary" @click="$emit('refresh')">Refresh Status</button>
			<button v-if="state" type="button" class="btn btn-primary" :title="state.account">{{ state.nickname || state.account || 'No Account' }}</button>
		</div>
	</div>
</template>

<style scoped>
.header-row {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    margin-bottom: 40px;
}
.header-left {
    max-width: 600px;
}
.breadcrumbs {
    display: flex;
    gap: 8px;
    font-size: 12px;
    color: var(--text-tertiary);
    margin-bottom: 16px;
}
.title-row {
    display: flex;
    align-items: center;
    gap: 20px;
}
h1 {
    font-family: var(--font-serif);
    font-size: 48px;
    font-weight: 400;
    letter-spacing: -0.02em;
    margin: 0;
}
.actions {
    display: flex;
    gap: 12px;
}
</style>
