<script setup lang="ts">
import { inject } from 'vue';
import { chatlogKey } from '../composables/chatlogContext';
import { maskKey } from '../composables/useChatlog';
import { backend } from '../wailsbridge';

const injected = inject(chatlogKey);
if (!injected) throw new Error('chatlog not provided');
const chat = injected;

const { dataKey, imgKey, dataDir, workDir, state, run } = chat;

async function toggleHTTP() {
	if (state.value?.httpEnabled) {
		const ok = await chat.confirm({
			title: '停止 HTTP 服务',
			message: '确认停止 HTTP 服务？停止后 API 与 MCP 接口将不可访问。',
			confirmText: '停止',
			cancelText: '取消',
			danger: true,
		});
		if (!ok) return;
		return run(() => backend.StopHTTP(), '已停止 HTTP 服务');
	}
	return run(() => backend.StartHTTP(), '已启动 HTTP 服务');
}

</script>

<template>
	<div class="overview-container">
		<div class="section-header">
			<span class="section-number">01</span>
			<span class="section-title">Key Information</span>
			<div class="section-dot"></div>
		</div>
		
		<div class="card" style="margin-bottom: 40px;">
			<div class="stats-grid">
				<div class="stat-item">
					<span class="stat-label">Data Key</span>
					<span class="stat-value">{{ maskKey(dataKey) }}</span>
				</div>
				<div class="stat-item">
					<span class="stat-label">Img Key</span>
					<span class="stat-value">{{ maskKey(imgKey) }}</span>
				</div>
				<div class="stat-item">
					<span class="stat-label">PID</span>
					<span class="stat-value">{{ state?.pid || '-' }}</span>
				</div>
			</div>
		</div>

		<div class="section-header">
			<span class="section-number">02</span>
			<span class="section-title">Directory Paths</span>
			<div class="section-dot"></div>
		</div>
		
		<div class="card" style="margin-bottom: 40px;">
			<div class="path-info">
				<div class="path-item">
					<span class="stat-label">Data Directory</span>
					<div class="path-value">{{ dataDir || '-' }}</div>
				</div>
				<div class="path-item" style="margin-top: 16px;">
					<span class="stat-label">Work Directory</span>
					<div class="path-value">{{ workDir || '-' }}</div>
				</div>
<style scoped>
.overview-container {
    display: flex;
    flex-direction: column;
}
.stats-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
}
.stat-item {
    padding: 0 24px;
    display: flex;
    flex-direction: column;
    gap: 6px;
    border-right: 1px solid var(--border-subtle);
}
.stat-item:first-child { padding-left: 0; }
.stat-item:last-child { border-right: none; }
.stat-label {
    font-size: 11px;
    color: var(--text-tertiary);
    text-transform: uppercase;
    letter-spacing: 0.5px;
}
.stat-value {
    font-family: var(--font-serif);
    font-size: 18px;
    color: var(--text-primary);
}
.path-value {
    font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
    font-size: 13px;
    color: var(--text-secondary);
    margin-top: 4px;
    word-break: break-all;
}
.actions-row {
    display: flex;
    gap: 12px;
    margin-bottom: 24px;
}
.api-links {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 24px;
    padding-top: 24px;
    border-top: 1px solid var(--border-subtle);
}
</style>
			</div>
		</div>

		<div class="section-header">
			<span class="section-number">03</span>
			<span class="section-title">Quick Actions & Services</span>
			<div class="section-dot"></div>
		</div>
		
		<div class="card">
			<div class="actions-row">
				<button type="button" :class="['btn', state?.httpEnabled ? 'btn-danger' : 'btn-primary']" @click="toggleHTTP">
					{{ state?.httpEnabled ? 'Stop HTTP Service' : 'Start HTTP Service' }}
				</button>
				<button type="button" class="btn btn-secondary" @click="run(() => backend.Refresh(), 'Refreshed')">Refresh Status</button>
			</div>
			<div v-if="state?.httpAddr" class="api-links">
				<div class="api-item">
					<span class="stat-label">API Endpoint</span>
					<div class="path-value">http://{{ state.httpAddr }}/api/v1/session</div>
				</div>
				<div class="api-item">
					<span class="stat-label">MCP Endpoint</span>
					<div class="path-value">http://{{ state.httpAddr }}/mcp</div>
				</div>
			</div>
		</div>
	</div>
</template>
