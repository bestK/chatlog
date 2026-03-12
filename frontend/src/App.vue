<script setup lang="ts">
import { computed, provide } from 'vue';
import { backend } from './wailsbridge';
import { appContextKey } from './app/context';
import { createAppState, type Page } from './app/state';
import { createFeedbackService } from './app/feedback';
import { Toaster } from '@/components/ui/sonner';
import {
	Dialog,
	DialogContent,
	DialogDescription,
	DialogFooter,
	DialogHeader,
	DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import SidebarNav from './components/SidebarNav.vue';
import Topbar from './components/Topbar.vue';
import PageOverview from './pages/PageOverview.vue';
import PageAccounts from './pages/PageAccounts.vue';
import PageService from './pages/PageService.vue';
import PageWebhook from './pages/PageWebhook.vue';
import PageSettings from './pages/PageSettings.vue';
import PageLogs from './pages/PageLogs.vue';
import GuideWizard from './components/GuideWizard.vue';


const feedback = createFeedbackService();
const appState = createAppState(feedback);
const appContext = {
	...appState,
	feedback,
};
provide(appContextKey, appContext);

const { page, nav, state, statusPill, previewBanner, run } = appContext;
const hostConfirmState = computed(() => feedback.confirmState.value);
const hostStatusState = computed(() => feedback.statusState.value);
const confirmOpen = computed({
	get: () => Boolean(feedback.confirmState.value),
	set: (open: boolean) => {
		if (!open && feedback.confirmState.value) {
			feedback.cancelConfirm();
		}
	},
});
const statusOpen = computed({
	get: () => Boolean(feedback.statusState.value),
	set: (open: boolean) => {
		if (!open) {
			feedback.status.close();
		}
	},
});

function setPage(p: Page) {
	page.value = p;
}

</script>

<template>
	<div class="h-full w-full">
		<Toaster richColors closeButton position="top-right" :expand="false" :visibleToasts="5" />
		<GuideWizard />


		<div class="grid h-full w-full min-h-0 lg:grid-cols-[280px_minmax(0,1fr)]">
			<div class="h-full min-h-0 overflow-hidden">
				<SidebarNav :nav="nav" :page="page" :state="state" @update:page="setPage" />
			</div>

			<div class="flex h-full min-h-0 min-w-0 flex-col gap-6 overflow-hidden px-4 py-4 lg:px-8 lg:py-8">
				<div
					v-if="previewBanner"
					class="inline-flex w-fit rounded-md border border-amber-500/30 bg-amber-500/10 px-3 py-1.5 text-xs font-medium text-amber-200"
				>
					{{ previewBanner }}
				</div>

			<Topbar
				:page="page"
				:state="state"
				:statusPill="statusPill"
				@refresh="run(() => backend.Refresh(), '已刷新')"
			/>

			<div :class="['page min-h-0 min-w-0 flex-1 flex flex-col', (page !== '概览' && page !== '账号' && page !== '服务' && page !== 'Webhook' && page !== '设置') ? 'overflow-hidden' : 'overflow-auto']">
				<PageOverview v-if="page === '概览'" />
				<PageAccounts v-else-if="page === '账号'" />
				<PageService v-else-if="page === '服务'" />
				<PageWebhook v-else-if="page === 'Webhook'" />
				<PageSettings v-else-if="page === '设置'" />
				<PageLogs v-else />
			</div>

				<Dialog v-model:open="confirmOpen">
					<DialogContent>
						<DialogHeader>
							<DialogTitle>{{ hostConfirmState?.title }}</DialogTitle>
							<DialogDescription>
								{{ hostConfirmState?.message }}
							</DialogDescription>
						</DialogHeader>
						<DialogFooter>
							<Button variant="outline" @click="feedback.cancelConfirm()">
								{{ hostConfirmState?.cancelText || '取消' }}
							</Button>
							<Button
								:variant="hostConfirmState?.danger ? 'destructive' : 'default'"
								@click="feedback.acceptConfirm()"
							>
								{{ hostConfirmState?.confirmText || '确认' }}
							</Button>
						</DialogFooter>
					</DialogContent>
				</Dialog>

				<Dialog v-model:open="statusOpen">
					<DialogContent :show-close-button="hostStatusState?.mode !== 'loading'">
						<DialogHeader>
							<DialogTitle>{{ hostStatusState?.title }}</DialogTitle>
							<DialogDescription>
								{{ hostStatusState?.message }}
							</DialogDescription>
						</DialogHeader>
						<div
							v-if="hostStatusState?.detail"
							class="max-h-56 overflow-auto rounded-md border bg-muted/30 p-4 text-xs leading-6 text-muted-foreground whitespace-pre-wrap break-all"
						>
							{{ hostStatusState.detail }}
						</div>
						<DialogFooter>
							<Button v-if="hostStatusState?.mode === 'loading'" variant="outline" disabled>
								处理中...
							</Button>
							<Button v-else @click="feedback.status.close()">
								关闭
							</Button>
						</DialogFooter>
					</DialogContent>
				</Dialog>
			</div>
		</div>
	</div>
</template>
