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
	padding: 32px 20px;
	box-sizing: border-box;
	background: #0a0a0a;
	border-right: 1px solid rgba(255, 255, 255, 0.03);
	position: relative;
}

/* Brand Section */
.brand {
	display: flex;
	align-items: center;
	gap: 14px;
	margin-bottom: 48px;
	padding: 0 4px;
}

.brandMark {
	width: 36px;
	height: 36px;
	border-radius: 10px;
	background: #1a1a1a;
	display: flex;
	align-items: center;
	justify-content: center;
	border: 1px solid rgba(255, 255, 255, 0.08);
	box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2);
}

.markInner {
	width: 14px;
	height: 14px;
	border-radius: 3px;
	background: linear-gradient(135deg, #00d2ff 0%, #3a7bd5 100%);
	box-shadow: 0 0 10px rgba(0, 210, 255, 0.4);
}

.brandText {
	display: flex;
	flex-direction: column;
}

.brandTitle {
	font-family: 'Playfair Display', serif; /* Fallback to serif if not available */
	font-size: 20px;
	font-weight: 700;
	color: #ffffff;
	letter-spacing: -0.02em;
	line-height: 1;
	margin: 0;
}

.brandSub {
	font-size: 10px;
	font-weight: 600;
	color: rgba(255, 255, 255, 0.3);
	text-transform: uppercase;
	letter-spacing: 0.12em;
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
	padding: 14px 16px;
	border-radius: 14px;
	background: transparent;
	border: none;
	cursor: pointer;
	transition: all 0.25s cubic-bezier(0.23, 1, 0.32, 1);
	width: 100%;
	text-align: left;
	position: relative;
	overflow: hidden;
}

.navItem:hover {
	background: rgba(255, 255, 255, 0.03);
}

.navItemActive {
	background: rgba(255, 255, 255, 0.05) !important;
}

.navContent {
	display: flex;
	flex-direction: column;
	gap: 2px;
	z-index: 1;
}

.navText {
	font-size: 14px;
	font-weight: 500;
	color: rgba(255, 255, 255, 0.5);
	transition: color 0.25s ease;
}

.navItem:hover .navText,
.navItemActive .navText {
	color: #ffffff;
}

.navHint {
	font-size: 11px;
	color: rgba(255, 255, 255, 0.2);
	transition: color 0.25s ease;
}

.navItemActive .navHint {
	color: rgba(255, 255, 255, 0.4);
}

.activeIndicator {
	position: absolute;
	left: 0;
	top: 50%;
	transform: translateY(-50%);
	width: 3px;
	height: 16px;
	background: #ffffff;
	border-radius: 0 2px 2px 0;
	box-shadow: 0 0 12px rgba(255, 255, 255, 0.3);
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
	border-radius: 18px;
	background: rgba(255, 255, 255, 0.02);
	border: 1px solid rgba(255, 255, 255, 0.04);
	transition: all 0.3s ease;
}

.userCard:hover {
	background: rgba(255, 255, 255, 0.04);
	border-color: rgba(255, 255, 255, 0.08);
}

.avatarWrapper {
	position: relative;
	flex-shrink: 0;
}

.userAvatar {
	width: 36px;
	height: 36px;
	border-radius: 12px;
	object-fit: cover;
	background: #1a1a1a;
}

.userAvatarPlaceholder {
	width: 36px;
	height: 36px;
	border-radius: 12px;
	background: #1a1a1a;
	display: flex;
	align-items: center;
	justify-content: center;
	font-size: 14px;
	font-weight: 600;
	color: rgba(255, 255, 255, 0.4);
	border: 1px solid rgba(255, 255, 255, 0.05);
}

.statusBadge {
	position: absolute;
	bottom: -2px;
	right: -2px;
	width: 10px;
	height: 10px;
	border-radius: 50%;
	border: 2px solid #0a0a0a;
}

.statusBadge.online {
	background: #22c55e;
	box-shadow: 0 0 8px rgba(34, 197, 94, 0.4);
}

.statusBadge.offline {
	background: #4b5563;
}

.userInfo {
	min-width: 0;
	flex: 1;
}

.userName {
	font-size: 13px;
	font-weight: 600;
	color: #ffffff;
	white-space: nowrap;
	overflow: hidden;
	text-overflow: ellipsis;
}

.userStatusLabel {
	font-size: 10px;
	font-weight: 500;
	color: rgba(255, 255, 255, 0.25);
	margin-top: 1px;
}
</style>
