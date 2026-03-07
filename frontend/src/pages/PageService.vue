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
        .then(ok => (ok ? run(() => backend.SetHTTPAddr(httpAddr.value), '已保存') : undefined));
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
    <div class="service-container-main">
        <div class="section-header">
            <span class="section-number">01</span>
            <span class="section-title">HTTP & MCP Service</span>
            <div class="section-dot"></div>
        </div>

        <div class="card cardWide">
            <div class="service-dashboard">
                <div class="dashboard-left">
                    <div class="label">Service Status</div>
                    <div :class="['status-indicator', state?.httpEnabled ? 'is-active' : 'is-inactive']">
                        <div v-if="state?.httpEnabled" class="pulse-ring"></div>
                        <span class="indicator-dot"></span>
                        <span class="indicator-text">{{ state?.httpEnabled ? 'RUNNING' : 'STOPPED' }}</span>
                    </div>
                </div>
                <div class="dashboard-right">
                    <button
                        type="button"
                        :class="['btn', state?.httpEnabled ? 'btnDanger' : 'btnBrand']"
                        @click="toggleHTTP"
                    >
                        <span class="btn-icon">{{ state?.httpEnabled ? '■' : '▶' }}</span>
                        {{ state?.httpEnabled ? 'Stop' : 'Start' }}
                    </button>
                </div>
            </div>

            <div class="service-details">
                <div class="detail-item">
                    <div class="label">Listen Address</div>
                    <div class="config-input-wrap">
                        <input v-model="httpAddr" class="input mono" placeholder="127.0.0.1:5030" />
                        <button type="button" class="btn" @click="saveAddr">Save</button>
                    </div>
                </div>

                <div v-if="state?.httpAddr" class="detail-item">
                    <div class="label">Access Endpoints</div>
                    <div class="pill-row">
                        <div class="pill pillXs mono">API: http://{{ state.httpAddr }}/api/v1/session</div>
                        <div class="pill pillXs mono">MCP: http://{{ state.httpAddr }}/mcp</div>
                    </div>
                </div>
            </div>
        </div>

        <div class="section-header">
            <span class="section-number">02</span>
            <span class="section-title">API Playground</span>
            <div class="section-dot"></div>
        </div>

        <div class="card cardWide">
            <div class="pg-list">
                <div v-for="ep in endpoints" :key="ep.path" class="pg-item">
                    <div class="pg-header">
                        <div class="pg-left">
                            <span class="pg-method">{{ ep.method }}</span>
                            <span class="pg-name">{{ ep.name }}</span>
                            <span class="pg-desc">{{ ep.desc }}</span>
                        </div>
                        <div class="pg-actions">
                            <button
                                type="button"
                                class="btn pg-try"
                                :disabled="!state?.httpEnabled"
                                @click="tryApi(ep)"
                            >
                                Try it
                            </button>
                            <button
                                type="button"
                                :class="['btn', copiedId === ep.path ? 'btnBrand' : '']"
                                class="pg-copy"
                                @click="copyCmd(ep)"
                            >
                                {{ copiedId === ep.path ? 'Copied ✓' : 'Copy curl' }}
                            </button>
                        </div>
                    </div>

                    <div class="pg-url mono">{{ fullUrl(ep) }}</div>

                    <div v-if="ep.params" class="pg-params">
                        <div v-for="p in ep.params" :key="p.key" class="pg-param">
                            <label class="label"
                                >{{ p.key }}<span class="pg-paramHint">{{ p.desc }}</span></label
                            >
                            <input v-model="epParams[ep.path][p.key]" class="input mono" :placeholder="p.placeholder" />
                        </div>
                    </div>

                    <div v-if="responses[ep.path]" class="pg-res">
                        <div class="pg-resHeader">
                            <span class="pg-resTitle">Response</span>
                            <button class="btn btn-xs" @click="responses[ep.path] = null">Clear</button>
                        </div>
                        <pre class="mono scrollbar">{{ responses[ep.path] }}</pre>
                    </div>
                </div>
            </div>
        </div>
    </div>
</template>

<style scoped>
.service-container-main {
    display: flex;
    flex-direction: column;
}

.section-header {
    display: flex;
    align-items: center;
    margin-bottom: 24px;
    margin-top: 24px;
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

.service-dashboard {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 24px;
    background: var(--panel);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    margin-bottom: 24px;
}

.status-indicator {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 8px 16px;
    border-radius: 999px;
    margin-top: 12px;
    position: relative;
    font-family: ui-monospace, SFMono-Regular, monospace;
    font-size: 13px;
    font-weight: 700;
}

.status-indicator.is-active {
    background: rgba(46, 229, 157, 0.1);
    color: var(--ok);
    border: 1px solid rgba(46, 229, 157, 0.2);
}

.status-indicator.is-inactive {
    background: rgba(255, 255, 255, 0.05);
    color: var(--muted);
    border: 1px solid var(--border);
}

.indicator-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: currentColor;
}

.pulse-ring {
    position: absolute;
    left: 16px;
    width: 8px;
    height: 8px;
    border-radius: 50%;
    border: 2px solid var(--ok);
    animation: pulse 2s infinite;
}

@keyframes pulse {
    0% {
        transform: scale(1);
        opacity: 0.8;
    }
    70% {
        transform: scale(3);
        opacity: 0;
    }
    100% {
        transform: scale(1);
        opacity: 0;
    }
}

.service-details {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
    gap: 20px;
}

.detail-item {
    padding: 20px;
    background: rgba(0, 0, 0, 0.2);
    border: 1px solid var(--border);
    border-radius: var(--radius);
}

.config-input-wrap {
    display: flex;
    gap: 12px;
    margin-top: 12px;
}

.config-input-wrap .input {
    flex: 1;
}

.pill-row {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    margin-top: 12px;
}

.pg-list {
    display: flex;
    flex-direction: column;
    gap: 16px;
}

.pg-item {
    border: 1px solid var(--border);
    border-radius: var(--radius);
    background: var(--panel);
    padding: 20px;
}

.pg-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    margin-bottom: 16px;
}

.pg-left {
    display: flex;
    align-items: center;
    gap: 12px;
    flex-wrap: wrap;
}

.pg-method {
    font-size: 10px;
    font-weight: 800;
    padding: 3px 8px;
    border-radius: 4px;
    background: rgba(53, 215, 255, 0.1);
    color: var(--brand);
    border: 1px solid rgba(53, 215, 255, 0.2);
}

.pg-name {
    font-size: 14px;
    font-weight: 600;
}

.pg-desc {
    font-size: 12px;
    color: var(--muted);
}

.pg-url {
    margin-top: 12px;
    font-size: 11px;
    padding: 10px 14px;
    border-radius: var(--radius-sm);
    background: rgba(0, 0, 0, 0.2);
    border: 1px solid var(--border);
    color: var(--text);
    word-break: break-all;
}

.pg-params {
    margin-top: 20px;
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
    gap: 12px;
}

.pg-paramHint {
    font-size: 10px;
    margin-left: 6px;
    color: var(--muted);
}

.pg-res {
    margin-top: 20px;
    border-radius: var(--radius-sm);
    background: #000;
    border: 1px solid var(--border);
    overflow: hidden;
}

.pg-resHeader {
    padding: 8px 16px;
    background: var(--panel-2);
    border-bottom: 1px solid var(--border);
    display: flex;
    justify-content: space-between;
    align-items: center;
}

.pg-resTitle {
    font-size: 10px;
    font-weight: 800;
    color: var(--muted);
    text-transform: uppercase;
}

.pg-res pre {
    margin: 0;
    padding: 16px;
    font-size: 12px;
    max-height: 300px;
    overflow: auto;
    color: var(--ok);
    white-space: pre-wrap;
}

.btn-xs {
    height: 24px;
    font-size: 10px;
    padding: 0 8px;
}

@media (max-width: 800px) {
    .pg-header {
        flex-direction: column;
        align-items: flex-start;
    }
    .pg-actions {
        width: 100%;
        display: flex;
        gap: 8px;
    }
    .pg-actions .btn {
        flex: 1;
    }
}
</style>
