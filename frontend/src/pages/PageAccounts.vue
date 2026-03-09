<script setup lang="ts">
import { inject } from 'vue';
import { backend, type Instance } from '../wailsbridge';
import { chatlogKey } from '../composables/chatlogContext';

const chat = inject(chatlogKey);
if (!chat) throw new Error('chatlog not provided');

const { instances, run, state } = chat;

function getAccountName(instance: Instance) {
	if (state.value?.pid === instance.pid && state.value.nickname) {
		return state.value.nickname;
	}
	return instance.name || '未知账号';
}

function getAccountAvatar(instance: Instance) {
	if (state.value?.pid === instance.pid) {
		return state.value.smallHeadImgUrl || '';
	}
	return '';
}

function getAvatarFallback(instance: Instance) {
	const name = getAccountName(instance).trim();
	return name ? name.slice(0, 1).toUpperCase() : '?';
}

function switchTo(pid: number) {
	return run(() => backend.SwitchToPID(pid), '已切换账号');
}
</script>

<template>
    <div class="accounts-container">
        <div class="section-header">
            <span class="section-number">01</span>
            <span class="section-title">WeChat Processes</span>
            <div class="section-dot"></div>
        </div>

        <div class="card cardWide">
            <div class="toolbar accountToolbar">
                <div class="toolbarGroup">
                    <div class="pill">Detected: {{ instances.length }}</div>
                    <div class="navHint">Check status before switching accounts</div>
                </div>
            </div>

            <div class="list">
                <div v-if="instances.length === 0" class="listItem accountItemEmpty">
                    <div class="listMain">
                        <div class="listTitle">No WeChat Process Detected</div>
                        <div class="listMeta">Please start and login to WeChat first.</div>
                    </div>
                </div>
                <div v-else v-for="x in instances" :key="x.pid" class="listItem accountItem">
                    <div class="accountTop">
                        <div class="listMain accountMain">
                            <div class="accountHeader">
                                <div class="accountIdentity">
                                    <div class="accountAvatarWrap">
                                        <img
                                            v-if="getAccountAvatar(x)"
                                            :src="getAccountAvatar(x)"
                                            :alt="`${getAccountName(x)} 头像`"
                                            class="accountAvatar"
                                        />
                                        <div v-else class="accountAvatar accountAvatarFallback">
                                            {{ getAvatarFallback(x) }}
                                        </div>
                                    </div>
                                    <div class="listTitle accountTitle">{{ getAccountName(x) }}</div>
                                </div>
                                <span :class="['status-badge-mini', x.status === 'online' ? 'ok' : 'bad']">
                                    {{
                                        x.status === 'online'
                                            ? 'Online'
                                            : x.status === 'offline'
                                              ? 'Offline'
                                              : x.status || 'Unknown'
                                    }}
                                </span>
                            </div>
                            <div class="listMeta mono accountMetaLine">
                                PID {{ x.pid }} · v{{ x.fullVersion || '-' }} · {{ x.platform || '-' }}
                            </div>
                        </div>
                        <div class="accountActionWrap">
                            <button type="button" class="btn btnBrand accountAction" @click="switchTo(x.pid)">
                                Switch
                            </button>
                        </div>
                    </div>
                    <div class="accountPathWrap">
                        <div class="accountPathLabel">Data Directory</div>
                        <div class="listMeta mono accountPath" :title="x.dataDir || '-'">{{ x.dataDir || '-' }}</div>
                    </div>
                </div>
            </div>
        </div>
    </div>
</template>

<style scoped>
.accounts-container {
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

.accountToolbar {
    margin-bottom: 16px;
    background: transparent;
    border: none;
    padding: 0;
}

.accountItem {
    display: flex;
    flex-direction: column;
    align-items: stretch;
    gap: 16px;
    padding: 20px;
    background: var(--panel);
    border: 1px solid var(--border);
    border-radius: var(--radius);
}

.accountItemEmpty {
    min-height: 100px;
    justify-content: center;
    text-align: center;
}

.accountTop {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
}

.accountMain {
    min-width: 0;
    gap: 6px;
}

.accountHeader {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
}

.accountIdentity {
	display: flex;
	align-items: center;
	gap: 10px;
	min-width: 0;
}

.accountAvatarWrap {
	flex-shrink: 0;
}

.accountAvatar {
	width: 32px;
	height: 32px;
	border-radius: 50%;
	object-fit: cover;
	display: block;
	border: 1px solid rgba(255, 255, 255, 0.08);
	background: rgba(255, 255, 255, 0.04);
}

.accountAvatarFallback {
	display: flex;
	align-items: center;
	justify-content: center;
	font-size: 13px;
	font-weight: 700;
	color: var(--text);
	background: linear-gradient(135deg, rgba(139, 92, 246, 0.22), rgba(59, 130, 246, 0.22));
}

.accountTitle {
    font-size: 15px;
    font-weight: 600;
    min-width: 0;
}

.status-badge-mini {
    padding: 2px 8px;
    font-size: 10px;
    font-weight: 700;
    text-transform: uppercase;
    border-radius: 4px;
    letter-spacing: 0.02em;
}

.status-badge-mini.ok {
    background: rgba(46, 229, 157, 0.1);
    color: var(--ok);
    border: 1px solid rgba(46, 229, 157, 0.2);
}

.status-badge-mini.bad {
    background: rgba(255, 91, 127, 0.1);
    color: var(--bad);
    border: 1px solid rgba(255, 91, 127, 0.2);
}

.accountMetaLine {
    font-size: 12px;
    color: var(--muted);
}

.accountPathWrap {
    padding: 12px;
    border-radius: var(--radius-sm);
    background: rgba(0, 0, 0, 0.2);
    border: 1px solid var(--border);
}

.accountPathLabel {
    font-size: 10px;
    text-transform: uppercase;
    font-weight: 700;
    color: var(--muted);
    margin-bottom: 6px;
    letter-spacing: 0.05em;
}

.accountPath {
    line-height: 1.4;
    word-break: break-all;
    font-size: 11px;
}

.accountActionWrap {
    flex-shrink: 0;
}

.accountAction {
    min-width: 80px;
}

@media (max-width: 600px) {
    .accountTop {
        flex-direction: column;
        align-items: flex-start;
    }
    .accountActionWrap {
        width: 100%;
    }
    .accountAction {
        width: 100%;
    }
}
</style>
