<script setup lang="ts">
import { inject, reactive, ref } from 'vue';
import { backend } from '../wailsbridge';
import { chatlogKey } from '../composables/chatlogContext';

const injected = inject(chatlogKey);
if (!injected) throw new Error('chatlog not provided');
const chat = injected;

const { httpAddr, state, run } = chat;

function saveAddr() {
	return chat
		.confirm({
			title: '保存 HTTP 地址',
			message: '确认保存并写入配置？',
			confirmText: '保存',
			cancelText: '取消',
		})
		.then((ok) => (ok ? run(() => backend.SetHTTPAddr(httpAddr.value), '已保存') : undefined));
}

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
		return run(() => backend.StopHTTP(), '已停止');
	}
	return run(() => backend.StartHTTP(), '已启动');
}

function baseUrl() {
	const addr = state.value?.httpAddr || httpAddr.value || '127.0.0.1:5030';
	return `http://${addr}`;
}

interface Endpoint {
	name: string;
	method: string;
	path: string;
	desc: string;
	params?: Array<{ key: string; placeholder: string; desc: string }>;
}

const endpoints: Endpoint[] = [
	{
		name: '会话列表',
		method: 'GET',
		path: '/api/v1/session',
		desc: '获取最近会话列表',
		params: [
			{ key: 'keyword', placeholder: '关键词', desc: '内容过滤' },
			{ key: 'limit', placeholder: '50', desc: '返回数量' },
			{ key: 'offset', placeholder: '0', desc: '分页偏移' },
			{ key: 'format', placeholder: 'json', desc: '输出格式' },
		],
	},
	{
		name: '联系人列表',
		method: 'GET',
		path: '/api/v1/contact',
		desc: '获取联系人列表',
		params: [
			{ key: 'keyword', placeholder: '关键词', desc: '昵称/备注过滤' },
			{ key: 'limit', placeholder: '20', desc: '返回数量' },
			{ key: 'offset', placeholder: '0', desc: '分页偏移' },
			{ key: 'format', placeholder: 'json', desc: '输出格式' },
		],
	},
	{
		name: '群聊列表',
		method: 'GET',
		path: '/api/v1/chatroom',
		desc: '获取群聊列表',
		params: [
			{ key: 'keyword', placeholder: '关键词', desc: '群名过滤' },
			{ key: 'limit', placeholder: '20', desc: '返回数量' },
			{ key: 'offset', placeholder: '0', desc: '分页偏移' },
			{ key: 'format', placeholder: 'json', desc: '输出格式' },
		],
	},
	{
		name: '聊天记录',
		method: 'GET',
		path: '/api/v1/chatlog',
		desc: '按条件查询聊天记录',
		params: [
			{ key: 'time', placeholder: '2025-01-01', desc: '时间范围' },
			{ key: 'talker', placeholder: 'wxid_xxx', desc: '聊天对象' },
			{ key: 'sender', placeholder: 'wxid_xxx', desc: '发送者' },
			{ key: 'keyword', placeholder: '关键词', desc: '内容搜索' },
			{ key: 'limit', placeholder: '20', desc: '返回数量' },
			{ key: 'offset', placeholder: '0', desc: '分页偏移' },
			{ key: 'format', placeholder: 'json', desc: '输出格式' },
		],
	},
	{
		name: 'MCP',
		method: 'GET',
		path: '/mcp',
		desc: 'MCP Streamable HTTP 端点',
	},
];

const epParams = reactive<Record<string, Record<string, string>>>({
	'/api/v1/session': { keyword: '', limit: '', offset: '', format: 'json' },
	'/api/v1/contact': { keyword: '', limit: '', offset: '', format: 'json' },
	'/api/v1/chatroom': { keyword: '', limit: '', offset: '', format: 'json' },
	'/api/v1/chatlog': { time: '', talker: '', sender: '', keyword: '', limit: '', offset: '', format: 'json' },
});

function fullUrl(ep: Endpoint): string {
	const base = `${baseUrl()}${ep.path}`;
	const params = epParams[ep.path];
	if (!params) return base;

	const qs = Object.entries(params)
		.map(([key, value]) => (value.trim() ? `${encodeURIComponent(key)}=${encodeURIComponent(value.trim())}` : ''))
		.filter(Boolean)
		.join('&');

	return qs ? `${base}?${qs}` : base;
}

function curlCmd(ep: Endpoint): string {
	return `curl -X ${ep.method} "${fullUrl(ep)}"`;
}

const copiedId = ref('');

async function copyCmd(ep: Endpoint) {
	const cmd = curlCmd(ep);
	try {
		await navigator.clipboard.writeText(cmd);
		chat.toast('已复制', cmd);
		copiedId.value = ep.path;
		setTimeout(() => {
			if (copiedId.value === ep.path) copiedId.value = '';
		}, 1600);
	} catch {
		chat.toast('复制失败', '浏览器不支持剪贴板操作');
	}
}

const responses = reactive<Record<string, any>>({});

async function tryApi(ep: Endpoint) {
	const url = fullUrl(ep);
	try {
		const res = await fetch(url);
		const data = await res.json();
		responses[ep.path] = JSON.stringify(data, null, 2);
	} catch (error) {
		responses[ep.path] = String(error);
	}
}
</script>

<template>
	<div class="grid">
		<div class="card cardWide">
			<div class="cardTitle">HTTP & MCP 服务</div>
			<div class="cardSub">启用后可通过 API 或 MCP 协议访问本地聊天数据。</div>

			<div class="service-container">
				<!-- 状态展示区 -->
				<div class="service-dashboard">
					<div class="dashboard-left">
						<div class="label">服务状态</div>
						<div :class="['status-indicator', state?.httpEnabled ? 'is-active' : 'is-inactive']">
							<div v-if="state?.httpEnabled" class="pulse-ring"></div>
							<span class="indicator-dot"></span>
							<span class="indicator-text">{{ state?.httpEnabled ? 'RUNNING' : 'STOPPED' }}</span>
						</div>
					</div>
					<div class="dashboard-right">
						<button
							type="button"
							:class="['action-btn', state?.httpEnabled ? 'btn-stop' : 'btn-start']"
							@click="toggleHTTP"
						>
							<span class="btn-icon">{{ state?.httpEnabled ? '■' : '▶' }}</span>
							{{ state?.httpEnabled ? '停止服务' : '启动服务' }}
						</button>
					</div>
				</div>

				<!-- 配置与详情区 -->
				<div class="service-details">
					<div class="detail-item">
						<div class="label">配置监听地址</div>
						<div class="config-input-wrap">
							<input v-model="httpAddr" class="input mono" placeholder="127.0.0.1:5030" />
							<button type="button" class="btn btnBrand" @click="saveAddr">保存配置</button>
						</div>
					</div>

					<div v-if="state?.httpAddr" class="detail-item">
						<div class="label">访问入口</div>
						<div class="pill-row">
							<div class="pill pillXs mono">API: http://{{ state.httpAddr }}/api/v1/session</div>
							<div class="pill pillXs mono">MCP: http://{{ state.httpAddr }}/mcp</div>
						</div>
					</div>
				</div>
			</div>
		</div>

		<div class="card cardWide">
			<div class="cardTitle">API Playground</div>
			<div class="cardSub">查看各端点 URL，输入参数后可一键复制 curl 命令。</div>

			<div class="pg-list">
				<div v-for="ep in endpoints" :key="ep.path" class="pg-item">
					<div class="pg-header">
						<div class="pg-left">
							<span class="pg-method">{{ ep.method }}</span>
							<span class="pg-name">{{ ep.name }}</span>
							<span class="pg-desc">{{ ep.desc }}</span>
						</div>
						<div class="pg-actions">
							<button type="button" class="btn pg-try" :disabled="!state?.httpEnabled" @click="tryApi(ep)">
								尝试一下
							</button>
							<button
								type="button"
								:class="['btn', copiedId === ep.path ? 'btnBrand' : '']"
								class="pg-copy"
								@click="copyCmd(ep)"
							>
								{{ copiedId === ep.path ? '已复制 ✓' : '复制 curl' }}
							</button>
						</div>
					</div>

					<div class="pg-url mono">{{ fullUrl(ep) }}</div>

					<div v-if="ep.params" class="pg-params">
						<div v-for="p in ep.params" :key="p.key" class="pg-param">
							<label class="label">{{ p.key }}<span class="pg-paramHint">{{ p.desc }}</span></label>
							<input v-model="epParams[ep.path][p.key]" class="input mono" :placeholder="p.placeholder" />
						</div>
					</div>

					<div v-if="responses[ep.path]" class="pg-res">
						<div class="pg-resHeader">
							<span class="pg-resTitle">响应结果</span>
							<button class="btn btn-xs" @click="responses[ep.path] = null">清除</button>
						</div>
						<pre class="mono scrollbar">{{ responses[ep.path] }}</pre>
					</div>
				</div>
			</div>
		</div>
	</div>
</template>

<style scoped>
.service-container {
	margin-top: 18px;
	display: flex;
	flex-direction: column;
	gap: 16px;
}

.service-dashboard {
	display: flex;
	align-items: center;
	justify-content: space-between;
	padding: 24px;
	background: linear-gradient(135deg, rgba(255, 255, 255, 0.05) 0%, rgba(255, 255, 255, 0.02) 100%);
	border: 1px solid rgba(255, 255, 255, 0.08);
	border-radius: 20px;
	backdrop-filter: blur(12px);
	box-shadow: 0 8px 32px rgba(0, 0, 0, 0.2);
}

.status-indicator {
	display: flex;
	align-items: center;
	gap: 10px;
	padding: 8px 16px;
	border-radius: 999px;
	margin-top: 8px;
	position: relative;
	font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
	letter-spacing: 1px;
	font-size: 14px;
	font-weight: 800;
	transition: all 0.3s ease;
}

.status-indicator.is-active {
	background: rgba(46, 229, 157, 0.12);
	border: 1px solid rgba(46, 229, 157, 0.3);
	color: var(--ok);
	box-shadow: 0 0 20px rgba(46, 229, 157, 0.1);
}

.status-indicator.is-inactive {
	background: rgba(255, 255, 255, 0.05);
	border: 1px solid rgba(255, 255, 255, 0.1);
	color: var(--muted);
}

.indicator-dot {
	width: 10px;
	height: 10px;
	border-radius: 50%;
	background: currentColor;
	box-shadow: 0 0 10px currentColor;
}

.pulse-ring {
	position: absolute;
	left: 16px;
	width: 10px;
	height: 10px;
	border-radius: 50%;
	border: 2px solid var(--ok);
	animation: pulse 2s infinite;
}

@keyframes pulse {
	0% { transform: scale(1); opacity: 0.8; }
	70% { transform: scale(3); opacity: 0; }
	100% { transform: scale(1); opacity: 0; }
}

.action-btn {
	display: flex;
	align-items: center;
	gap: 10px;
	padding: 12px 24px;
	border-radius: 14px;
	font-weight: 700;
	font-size: 15px;
	cursor: pointer;
	transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
	border: 1px solid transparent;
}

.btn-icon {
	font-size: 18px;
	line-height: 1;
}

.btn-start {
	background: var(--brand);
	color: #000;
	box-shadow: 0 4px 15px rgba(53, 215, 255, 0.3);
}

.btn-start:hover {
	transform: translateY(-2px);
	box-shadow: 0 6px 20px rgba(53, 215, 255, 0.4);
	filter: brightness(1.1);
}

.btn-stop {
	background: rgba(255, 91, 127, 0.15);
	border: 1px solid rgba(255, 91, 127, 0.3);
	color: #ff5b7f;
}

.btn-stop:hover {
	background: rgba(255, 91, 127, 0.25);
	transform: translateY(-2px);
}

.service-details {
	display: grid;
	grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
	gap: 16px;
}

.detail-item {
	padding: 16px;
	background: rgba(0, 0, 0, 0.2);
	border: 1px solid rgba(255, 255, 255, 0.05);
	border-radius: 16px;
}

.config-input-wrap {
	display: flex;
	gap: 10px;
	margin-top: 8px;
}

.config-input-wrap .input {
	flex: 1;
}

.pill-row {
	display: flex;
	flex-wrap: wrap;
	gap: 8px;
	margin-top: 8px;
}

.pg-list {
	margin-top: 12px;
	display: flex;
	flex-direction: column;
	gap: 10px;
}

.pg-item {
	border: 1px solid var(--border);
	border-radius: var(--radius-sm);
	background: rgba(0, 0, 0, 0.18);
	padding: 12px;
}

.pg-header {
	display: flex;
	align-items: center;
	justify-content: space-between;
	gap: 10px;
	flex-wrap: wrap;
}

.pg-left {
	display: flex;
	align-items: center;
	gap: 8px;
	flex-wrap: wrap;
}

.pg-method {
	font-size: 11px;
	font-weight: 700;
	padding: 3px 8px;
	border-radius: 6px;
	background: rgba(53, 215, 255, 0.12);
	border: 1px solid rgba(53, 215, 255, 0.25);
	color: var(--brand);
	letter-spacing: 0.5px;
}

.pg-name {
	font-size: 13px;
	font-weight: 650;
}

.pg-desc {
	font-size: 12px;
	color: var(--muted);
}

.pg-copy {
	font-size: 12px;
	height: 32px;
	padding: 0 10px;
}

.pg-url {
	margin-top: 8px;
	font-size: 12px;
	padding: 8px 10px;
	border-radius: 8px;
	background: rgba(0, 0, 0, 0.28);
	border: 1px solid var(--border);
	color: var(--brand);
	word-break: break-all;
	user-select: all;
}

.pg-params {
	margin-top: 10px;
	display: grid;
	grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
	gap: 8px;
}

.pg-param .label {
	display: flex;
	align-items: baseline;
	gap: 6px;
}

.pg-paramHint {
	font-size: 11px;
	color: var(--subtle);
}

.pg-param .input {
	height: 34px;
	font-size: 12px;
}

.pg-actions {
	display: flex;
	gap: 8px;
}

.pg-try {
	font-size: 12px;
	height: 32px;
	padding: 0 12px;
	background: rgba(46, 229, 157, 0.1);
	border-color: rgba(46, 229, 157, 0.2);
	color: var(--ok);
}

.pg-try:hover:not(:disabled) {
	background: rgba(46, 229, 157, 0.2);
	border-color: var(--ok);
}

.pg-try:disabled {
	opacity: 0.5;
	filter: grayscale(1);
}

.pg-res {
	margin-top: 12px;
	border-radius: 8px;
	background: #05070a;
	border: 1px solid var(--border);
	overflow: hidden;
}

.pg-resHeader {
	padding: 6px 10px;
	background: rgba(255, 255, 255, 0.03);
	border-bottom: 1px solid var(--border);
	display: flex;
	justify-content: space-between;
	align-items: center;
}

.pg-resTitle {
	font-size: 11px;
	font-weight: 700;
	color: var(--muted);
}

.pg-res pre {
	margin: 0;
	padding: 12px;
	font-size: 12px;
	max-height: 240px;
	overflow: auto;
	color: var(--ok);
	white-space: pre-wrap;
}

.btn-xs {
	height: 20px;
	font-size: 10px;
	padding: 0 6px;
	border-radius: 4px;
}

@media (max-width: 760px) {
	.service-details {
		grid-template-columns: 1fr;
	}
	.config-input-wrap {
		flex-direction: column;
	}
}
</style>
