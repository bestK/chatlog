<script setup lang="ts">
import type { Page } from '../composables/useChatlog';
import type { State } from '../wailsbridge';

const props = defineProps<{
	nav: Array<{ name: Page; hint: string }>;
	page: Page;
	state: State | null;
}>();
const emit = defineEmits<{ (e: 'update:page', value: Page): void }>();

function setPage(p: Page) {
	emit('update:page', p);
}
</script>

<template>
	<div class="sidebar">
		<div class="logo">
			<span></span>
			Chatlog.
		</div>

		<div class="nav-group">
			<div class="nav-label">Main</div>
			<ul class="nav-list">
				<li v-for="n in props.nav" :key="n.name">
					<a href="#" :class="['nav-item', props.page === n.name ? 'active' : '']" @click.prevent="setPage(n.name)">
						<span class="nav-icon">{{ props.page === n.name ? '▪' : '▫' }}</span> {{ n.name }}
					</a>
				</li>
			</ul>
		</div>

		<footer v-if="props.state?.account" class="user-profile">
			<img v-if="props.state.smallHeadImgUrl" :src="props.state.smallHeadImgUrl" class="avatar" alt="Avatar" referrerpolicy="no-referrer" />
			<div v-else class="avatar"></div>
			<div class="user-info">
				<div class="user-name">{{ props.state.nickname || props.state.account }}</div>
				<div class="user-role">{{ props.state.status === 'online' ? 'Connected' : 'Disconnected' }}</div>
			</div>
		</footer>
	</div>
</template>

<style scoped>
.logo {
    font-family: var(--font-serif);
    font-size: 20px;
    font-weight: 600;
    margin-bottom: 40px;
    color: var(--text-primary);
    display: flex;
    align-items: center;
    gap: 12px;
}
.logo span {
    width: 8px;
    height: 8px;
    background-color: var(--accent);
    border-radius: 50%;
    display: inline-block;
}
.nav-group {
    margin-bottom: 32px;
}
.nav-label {
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 1px;
    color: var(--text-tertiary);
    margin-bottom: 12px;
    font-weight: 600;
}
.nav-list {
    list-style: none;
    display: flex;
    flex-direction: column;
    gap: 4px;
}
.nav-item {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 10px 12px;
    border-radius: var(--radius-sm);
    color: var(--text-secondary);
    text-decoration: none;
    font-size: 14px;
    transition: all 0.2s ease;
}
.nav-item:hover {
    background-color: var(--bg-input);
    color: var(--text-primary);
}
.nav-item.active {
    background-color: var(--bg-card);
    color: var(--text-primary);
    border: 1px solid var(--border-subtle);
}
.nav-icon {
    width: 18px;
    height: 18px;
    opacity: 0.7;
}
.user-profile {
    margin-top: auto;
    display: flex;
    align-items: center;
    gap: 12px;
    padding-top: 20px;
    border-top: 1px solid var(--border-subtle);
}
.avatar {
    width: 32px;
    height: 32px;
    border-radius: 50%;
    background-color: var(--bg-input);
    border: 1px solid var(--border-subtle);
    object-fit: cover;
}
.user-info {
    display: flex;
    flex-direction: column;
    min-width: 0;
}
.user-name {
    font-size: 13px;
    font-weight: 500;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
}
.user-role {
    font-size: 11px;
    color: var(--text-tertiary);
}
