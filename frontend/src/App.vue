<script setup lang="ts">
import { provide } from 'vue';
import { backend } from './wailsbridge';
import { chatlogKey } from './composables/chatlogContext';
import { useChatlog, type Page } from './composables/useChatlog';
import { useConfirm } from './composables/useConfirm';
import SidebarNav from './components/SidebarNav.vue';
import Topbar from './components/Topbar.vue';
import ToastHost from './components/ToastHost.vue';
import ConfirmDialog from './components/ConfirmDialog.vue';
import PageOverview from './pages/PageOverview.vue';
import PageAccounts from './pages/PageAccounts.vue';
import PageKeys from './pages/PageKeys.vue';
import PageDecrypt from './pages/PageDecrypt.vue';
import PageService from './pages/PageService.vue';
import PageWebhook from './pages/PageWebhook.vue';
import PageSettings from './pages/PageSettings.vue';
import PageLogs from './pages/PageLogs.vue';

const chatBase = useChatlog();
const confirmApi = useConfirm();
const chat = { ...chatBase, confirm: confirmApi.confirm };
provide(chatlogKey, chat);

const { state: confirmState, accept: confirmAccept, cancel: confirmCancel } = confirmApi;

const { page, nav, state, statusPill, previewBanner, toasts, run } = chat;

function setPage(p: Page) {
	page.value = p;
}
</script>

<template>
	<div class="shell">
		<div class="sidebar">
			<SidebarNav :nav="nav" :page="page" @update:page="setPage" />
		</div>

		<div class="main">
			<div v-if="previewBanner" class="topbar" style="margin-bottom: 14px">
				<div class="subtitle">{{ previewBanner }}</div>
			</div>

			<Topbar
				:page="page"
				:state="state"
				:statusPill="statusPill"
				:showGetKeys="page === '概览' || page === '密钥'"
				:showDecrypt="page === '概览' || page === '解密'"
				@refresh="run(() => backend.Refresh(), '已刷新')"
				@getkeys="run(() => backend.GetKeys(), '已获取密钥')"
				@decrypt="run(() => backend.Decrypt(), '解密完成')"
			/>

			<div class="page">
				<PageOverview v-if="page === '概览'" />
				<PageAccounts v-else-if="page === '账号'" />
				<PageKeys v-else-if="page === '密钥'" />
				<PageDecrypt v-else-if="page === '解密'" />
				<PageService v-else-if="page === '服务'" />
				<PageWebhook v-else-if="page === 'Webhook'" />
				<PageSettings v-else-if="page === '设置'" />
				<PageLogs v-else />
			</div>

			<ToastHost :toasts="toasts" />
			<ConfirmDialog
				v-if="confirmState"
				:title="confirmState.title"
				:message="confirmState.message"
				:confirmText="confirmState.confirmText"
				:cancelText="confirmState.cancelText"
				:danger="confirmState.danger"
				@confirm="confirmAccept"
				@cancel="confirmCancel"
			/>
		</div>
	</div>
</template>
