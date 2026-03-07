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

        <div class="card overview-section">
            <div class="stats-grid">
                <div class="stat-item">
                    <span class="stat-label">Data Key</span>
                    <span class="stat-value mono">{{ maskKey(dataKey) }}</span>
                </div>
                <div class="stat-item">
                    <span class="stat-label">Img Key</span>
                    <span class="stat-value mono">{{ maskKey(imgKey) }}</span>
                </div>
                <div class="stat-item">
                    <span class="stat-label">Process PID</span>
                    <span class="stat-value">{{ state?.pid || '-' }}</span>
                </div>
            </div>
        </div>

        <div class="section-header">
            <span class="section-number">02</span>
            <span class="section-title">Directory Paths</span>
            <div class="section-dot"></div>
        </div>

        <div class="card overview-section">
            <div class="path-info">
                <div class="path-item">
                    <span class="stat-label">Data Directory</span>
                    <div class="path-value">{{ dataDir || '-' }}</div>
                </div>
                <div class="path-item" style="margin-top: 24px">
                    <span class="stat-label">Work Directory</span>
                    <div class="path-value">{{ workDir || '-' }}</div>
                </div>
            </div>
        </div>

        <div class="section-header">
            <span class="section-number">03</span>
            <span class="section-title">Quick Actions & Services</span>
            <div class="section-dot"></div>
        </div>

        <div class="card">
            <div class="actions-row">
                <button
                    type="button"
                    :class="['btn', state?.httpEnabled ? 'btnDanger' : 'btnBrand']"
                    @click="toggleHTTP"
                >
                    {{ state?.httpEnabled ? 'Stop HTTP Service' : 'Start HTTP Service' }}
                </button>
                <button type="button" class="btn" @click="run(() => backend.Refresh(), 'Refreshed')">
                    Refresh Status
                </button>
            </div>
            <div v-if="state?.httpAddr" class="api-links">
                <div class="api-item">
                    <span class="stat-label">API Endpoint</span>
                    <div class="path-value link-style">http://{{ state.httpAddr }}/api/v1/session</div>
                </div>
                <div class="api-item">
                    <span class="stat-label">MCP Endpoint</span>
                    <div class="path-value link-style">http://{{ state.httpAddr }}/mcp</div>
                </div>
            </div>
        </div>
    </div>
</template>

<style scoped>
.overview-container {
    display: flex;
    flex-direction: column;
}

.overview-section {
    margin-bottom: 40px;
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

.stats-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
}

.stat-item {
    padding: 0 24px;
    display: flex;
    flex-direction: column;
    gap: 8px;
    border-right: 1px solid var(--border);
}

.stat-item:first-child {
    padding-left: 0;
}

.stat-item:last-child {
    border-right: none;
}

.stat-label {
    font-size: 10px;
    color: var(--muted);
    text-transform: uppercase;
    letter-spacing: 0.05em;
    font-weight: 700;
}

.stat-value {
    font-size: 18px;
    font-weight: 600;
    color: var(--text);
}

.stat-value.mono {
    font-family: ui-monospace, SFMono-Regular, monospace;
    font-size: 15px;
    letter-spacing: 0.02em;
}

.path-value {
    font-family: ui-monospace, SFMono-Regular, monospace;
    font-size: 12px;
    color: var(--text);
    margin-top: 8px;
    word-break: break-all;
    padding: 10px 14px;
    background: rgba(0, 0, 0, 0.2);
    border-radius: var(--radius-sm);
    border: 1px solid var(--border);
}

.link-style {
    color: var(--brand);
    border-color: rgba(53, 215, 255, 0.2);
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
    border-top: 1px solid var(--border);
}
</style>
