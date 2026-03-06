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
		.replace(/^(\d{4}-\d{2}-\d{2}\s\d{2}:\d{2}:\d{2})/gm, '<span class="log-date">$1</span>')
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
	<div class="grid logGrid">
		<div class="card cardWide cardFill logLayout">
			<div class="cardTitle">日志</div>
			<div class="cardSub">展示最近的日志内容，便于排障与反馈。</div>

			<div class="row">
				<div class="field">
					<div class="label">日志文件</div>
					<input class="input mono" :value="logPath" readonly />
				</div>
				<button type="button" class="btn" @click="copyText(logPath)">复制路径</button>
				<button type="button" class="btn" @click="refresh">刷新</button>
			</div>

			<div class="row">
				<div class="field">
					<div class="label">过滤关键词</div>
					<input class="input" v-model="keyword" placeholder="例如：ERROR / decrypt / webhook" />
				</div>
				<button type="button" class="btn" @click="copyText(filtered)">复制内容</button>
			</div>

			<div class="row">
				<div class="pill" v-if="loading">读取中…</div>
				<div class="pill" v-else>行数：{{ maxLines }}</div>
				<div class="pill" v-if="keyword.trim()">过滤：{{ keyword.trim() }}</div>
			</div>

			<div class="grow logViewport">
				<pre ref="logBox" class="mono panel scrollbar logPanel" v-html="filteredHtml"></pre>
			</div>
		</div>
	</div>
</template>

<style scoped>
.logPanel {
	background: #06080c;
	padding: 16px;
	line-height: 1.6;
	font-size: 13px;
	color: rgba(255, 255, 255, 0.8);
	border-color: rgba(255, 255, 255, 0.08);
	box-shadow: inset 0 2px 10px rgba(0, 0, 0, 0.2);
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
	opacity: 0.8;
}

:deep(.log-date) {
	color: var(--subtle);
	margin-right: 8px;
}

.logViewport {
	margin-top: 4px;
}

.cardTitle {
	display: flex;
	align-items: center;
	gap: 8px;
}

.cardTitle::before {
	content: '';
	display: block;
	width: 4px;
	height: 14px;
	background: var(--brand);
	border-radius: 2px;
}
</style>
