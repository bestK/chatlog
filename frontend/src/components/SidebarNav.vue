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
	<div class="sidebarContainer">
		<header class="brand">
			<div class="brandMark">
				<div class="markInner" />
			</div>
			<div class="brandText">
				<h1 class="brandTitle">Chatlog</h1>
				<span class="brandSub">Wails Desktop</span>
			</div>
		</header>

		<nav class="nav">
			<div class="navGroup">
				<button
					v-for="n in props.nav"
					:key="n.name"
					type="button"
					:class="['navItem', props.page === n.name ? 'navItemActive' : '']"
					@click="setPage(n.name)"
				>
					<div class="navContent">
						<span class="navText">{{ n.name }}</span>
						<span class="navHint">{{ n.hint }}</span>
					</div>
					<div v-if="props.page === n.name" class="activeIndicator" />
				</button>
			</div>
		</nav>

		<footer v-if="props.state?.account" class="sidebarFooter">
			<div class="userCard">
				<div class="avatarWrapper">
					<img
						v-if="props.state.smallHeadImgUrl"
						:src="props.state.smallHeadImgUrl"
						class="userAvatar"
						alt="Avatar"
						referrerpolicy="no-referrer"
					/>
					<div v-else class="userAvatarPlaceholder">
						{{ props.state.nickname?.slice(0, 1)?.toUpperCase() || props.state.account?.slice(0, 1)?.toUpperCase() || '?' }}
					</div>
					<div :class="['statusBadge', props.state.status === 'online' ? 'online' : 'offline']" />
				</div>
				<div class="userInfo">
					<div class="userName">{{ props.state.nickname || props.state.account }}</div>
					<div class="userStatusLabel">{{ props.state.status === 'online' ? 'Connected' : 'Disconnected' }}</div>
				</div>
			</div>
		</footer>
	</div>
</template>

<style scoped>
.sidebarContainer {
	display: flex;
	flex-direction: column;
	height: 100%;
	padding: 40px 24px;
	box-sizing: border-box;
	background: var(--bg-sidebar);
	border-right: 1px solid var(--border);
	position: relative;
}

/* Brand Section */
.brand {
	display: flex;
	align-items: center;
	gap: 14px;
	margin-bottom: 60px;
}

.brandMark {
	width: 36px;
	height: 36px;
	border-radius: 10px;
	background: var(--panel);
	display: flex;
	align-items: center;
	justify-content: center;
	border: 1px solid var(--border);
}

.markInner {
	width: 14px;
	height: 14px;
	border-radius: 3px;
	background: var(--brand);
	box-shadow: 0 0 12px rgba(53, 215, 255, 0.4);
}

.brandText {
	display: flex;
	flex-direction: column;
}

.brandTitle {
	font-size: 20px;
	font-weight: 700;
	color: var(--text);
	letter-spacing: -0.02em;
	line-height: 1;
	margin: 0;
}

.brandSub {
	font-size: 10px;
	font-weight: 700;
	color: var(--muted);
	text-transform: uppercase;
	letter-spacing: 0.1em;
	margin-top: 4px;
}

/* Navigation */
.nav {
	flex: 1;
}

.navGroup {
	display: flex;
	flex-direction: column;
	gap: 4px;
}

.navItem {
	display: flex;
	align-items: center;
	justify-content: space-between;
	padding: 12px 16px;
	border-radius: 12px;
	background: transparent;
	border: 1px solid transparent;
	cursor: pointer;
	transition: all 0.2s ease;
	width: 100%;
	text-align: left;
	position: relative;
}

.navItem:hover {
	background: var(--panel);
}

.navItemActive {
	background: var(--panel-2) !important;
	border-color: var(--border);
}

.navContent {
	display: flex;
	flex-direction: column;
	gap: 2px;
	z-index: 1;
}

.navText {
	font-size: 14px;
	font-weight: 600;
	color: var(--muted);
	transition: color 0.2s ease;
}

.navItem:hover .navText,
.navItemActive .navText {
	color: var(--text);
}

.navHint {
	font-size: 10px;
	color: var(--subtle);
}

.activeIndicator {
	position: absolute;
	left: 0;
	top: 50%;
	transform: translateY(-50%);
	width: 3px;
	height: 16px;
	background: var(--brand);
	border-radius: 0 2px 2px 0;
	box-shadow: 0 0 12px var(--brand);
}

/* User Footer */
.sidebarFooter {
	margin-top: auto;
	padding-top: 24px;
}

.userCard {
	display: flex;
	align-items: center;
	gap: 12px;
	padding: 12px;
	border-radius: 16px;
	background: rgba(255, 255, 255, 0.02);
	border: 1px solid var(--border);
	transition: all 0.2s ease;
}

.userCard:hover {
	background: var(--panel);
}

.avatarWrapper {
	position: relative;
	flex-shrink: 0;
}

.userAvatar {
	width: 36px;
	height: 36px;
	border-radius: 10px;
	object-fit: cover;
	background: var(--bg-sidebar);
	border: 1px solid var(--border);
}

.userAvatarPlaceholder {
	width: 36px;
	height: 36px;
	border-radius: 10px;
	background: var(--panel);
	display: flex;
	align-items: center;
	justify-content: center;
	font-size: 14px;
	font-weight: 700;
	color: var(--muted);
	border: 1px solid var(--border);
}

.statusBadge {
	position: absolute;
	bottom: -2px;
	right: -2px;
	width: 10px;
	height: 10px;
	border-radius: 50%;
	border: 2px solid var(--bg-sidebar);
}

.statusBadge.online {
	background: var(--ok);
	box-shadow: 0 0 8px var(--ok);
}

.statusBadge.offline {
	background: var(--subtle);
}

.userInfo {
	min-width: 0;
	flex: 1;
}

.userName {
	font-size: 13px;
	font-weight: 700;
	color: var(--text);
	white-space: nowrap;
	overflow: hidden;
	text-overflow: ellipsis;
}

.userStatusLabel {
	font-size: 10px;
	font-weight: 600;
	color: var(--muted);
	margin-top: 1px;
}
</style>
