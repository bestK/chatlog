<script setup lang="ts">
import { computed, inject, nextTick, onMounted, onUnmounted, ref, watch } from 'vue';
import { chatlogKey } from '../composables/chatlogContext';
import { backend } from '../wailsbridge';

const injected = inject(chatlogKey);
if (!injected) throw new Error('chatlog not provided');
const chat = injected;

const logPath = ref('');
const content = ref('');
const loading = ref(false);
const keyword = ref('');
const maxLines = 200;

const logBox = ref<HTMLElement | null>(null);
let firstScroll = true;
let offLogChanged: (() => void) | undefined;

async function refresh() {
	loading.value = true;
	try {
		logPath.value = await backend.GetLogPath();
		content.value = await backend.ReadLogTail(maxLines);
	} catch (error) {
		chat.toast('读取失败', String(error));
	} finally {
		loading.value = false;
	}
}

const filtered = computed(() => {
	const kw = keyword.value.trim();
	if (!kw) return content.value;
	return content.value
		.split('\n')
		.filter((line) => line.includes(kw))
		.join('\n');
});

const filteredHtml = computed(() => {
	return filtered.value
		.replace(/&/g, '&amp;')
		.replace(/</g, '&lt;')
		.replace(/>/g, '&gt;')
		.replace(/^(\d{4}[\/-]\d{2}[\/-]\d{2}\s\d{2}:\d{2}:\d{2})/gm, '<span class="log-date">$1</span>')
		.replace(/\b(INF|INFO|SUCCESS)\b/g, '<span class="log-info">$1</span>')
		.replace(/\b(WRN|WARN|WARNING)\b/g, '<span class="log-warn">$1</span>')
		.replace(/\b(ERR|ERROR|FATAL|CRITICAL)\b/g, '<span class="log-error">$1</span>')
		.replace(/\b(DBG|DEBUG)\b/g, '<span class="log-debug">$1</span>');
});

async function copyText(text: string) {
	try {
		await navigator.clipboard.writeText(text);
		chat.toast('已复制', '已复制到剪贴板');
	} catch {
		chat.toast('复制失败', '当前环境不支持剪贴板');
	}
}

onMounted(async () => {
	await refresh();
	if (backend.isWails) {
		try {
			await backend.EnableLogEvents(true);
		} catch {
		}
	}
	offLogChanged = backend.EventsOn('log:changed', () => {
		void refresh();
	});
});

onUnmounted(() => {
	if (offLogChanged) offLogChanged();
	if (backend.isWails) {
		void backend.EnableLogEvents(false);
	}
});

function isNearBottom(el: HTMLElement) {
	return el.scrollTop + el.clientHeight >= el.scrollHeight - 40;
}

function scrollToBottom(el: HTMLElement) {
	el.scrollTop = el.scrollHeight;
}

watch(
	() => filtered.value,
	async () => {
		await nextTick();
		const el = logBox.value;
		if (!el) return;
		if (firstScroll) {
			firstScroll = false;
			scrollToBottom(el);
			return;
		}
		if (isNearBottom(el)) {
			scrollToBottom(el);
		}
	},
);
</script>

<template>
	<div class="logs-container">
		<div class="section-header">
			<span class="section-number">01</span>
			<span class="section-title">System Logs</span>
			<div class="section-dot"></div>
		</div>

		<div class="card cardWide flex-column grow">
			<div class="log-controls">
				<div class="field">
					<div class="label">Log File Path</div>
					<div class="input-group">
						<input class="input mono" :value="logPath" readonly />
						<button type="button" class="btn" @click="copyText(logPath)">Copy Path</button>
						<button type="button" class="btn" @click="refresh">Refresh</button>
					</div>
				</div>

				<div class="field">
					<div class="label">Filter Content</div>
					<div class="input-group">
						<input class="input" v-model="keyword" placeholder="e.g. ERROR / decrypt / webhook" />
						<button type="button" class="btn" @click="copyText(filtered)">Copy Log</button>
					</div>
				</div>
			</div>

			<div class="log-status-row">
				<div v-if="loading" class="log-pill loading">Reading logs...</div>
				<div v-else class="log-pill">Lines: {{ maxLines }}</div>
				<div v-if="keyword.trim()" class="log-pill filter">Filtering: {{ keyword.trim() }}</div>
			</div>

			<div class="log-viewport-wrapper grow">
				<pre ref="logBox" class="mono log-panel scrollbar" v-html="filteredHtml"></pre>
			</div>
		</div>
	</div>
</template>

<style scoped>
.logs-container {
	display: flex;
	flex-direction: column;
	height: 100%;
}

.section-header {
	display: flex;
	align-items: center;
	margin-bottom: 24px;
	border-bottom: 1px solid var(--border);
	padding-bottom: 12px;
	position: relative;
}

.section-number {
	font-size: 11px;
	color: var(--muted);
	margin-right: 12px;
	font-weight: 700;
}

.section-title {
	font-size: 13px;
	font-weight: 600;
	color: var(--text);
	text-transform: uppercase;
	letter-spacing: 0.05em;
}

.section-dot {
	width: 4px;
	height: 4px;
	background-color: var(--brand);
	border-radius: 50%;
	position: absolute;
	bottom: -2.5px;
	left: 0;
}

.flex-column {
	display: flex;
	flex-direction: column;
}

.log-controls {
	display: flex;
	flex-direction: column;
	gap: 16px;
	margin-bottom: 24px;
}

.input-group {
	display: flex;
	gap: 12px;
}

.input-group .input {
	flex: 1;
}

.log-status-row {
	display: flex;
	gap: 12px;
	margin-bottom: 16px;
}

.log-pill {
	padding: 2px 10px;
	font-size: 11px;
	font-weight: 700;
	background: var(--panel);
	border: 1px solid var(--border);
	border-radius: 4px;
	color: var(--muted);
	text-transform: uppercase;
}

.log-pill.loading {
	color: var(--brand);
	border-color: rgba(53, 215, 255, 0.3);
}

.log-pill.filter {
	color: var(--warn);
	border-color: rgba(255, 213, 106, 0.3);
}

.log-viewport-wrapper {
	position: relative;
	min-height: 0;
}

.log-panel {
	background: #000;
	padding: 20px;
	line-height: 1.6;
	font-size: 12px;
	color: rgba(255, 255, 255, 0.85);
	border: 1px solid var(--border);
	border-radius: var(--radius-sm);
	height: 100%;
	overflow-y: auto;
	box-shadow: inset 0 4px 20px rgba(0, 0, 0, 0.4);
}

:deep(.log-info) {
	color: var(--ok);
	font-weight: 600;
}

:deep(.log-warn) {
	color: var(--warn);
	font-weight: 600;
}

:deep(.log-error) {
	color: var(--bad);
	font-weight: 600;
}

:deep(.log-debug) {
	color: var(--brand);
	opacity: 0.9;
}

:deep(.log-date) {
	color: var(--subtle);
	margin-right: 12px;
}
</style>
