<script setup lang="ts">
import { computed, inject, nextTick, onMounted, ref, watch } from 'vue';
import { backend, type Contact, type Instance } from '../wailsbridge';
import { chatlogKey } from '../composables/chatlogContext';

const injected = inject(chatlogKey);
if (!injected) throw new Error('chatlog not provided');
const chat = injected;

const { instances, run, state } = chat;

const contactKeyword = ref('');
const contactsLoading = ref(false);
const contactsTotal = ref(0);
const contacts = ref<Contact[]>([]);
const contactLimit = ref(50);
const contactOffset = ref(0);
const contactLoadingSource = ref<'init' | 'refresh' | 'search' | 'prev' | 'next' | 'limit' | 'account' | null>(null);

const contactRangeText = computed(() => {
	if (contactsTotal.value <= 0) return '0';
	const start = Math.min(contactOffset.value + 1, contactsTotal.value);
	const end = Math.min(contactOffset.value + contacts.value.length, contactsTotal.value);
	return `${start}-${end}/${contactsTotal.value}`;
});

const hasContacts = computed(() => contacts.value.length > 0);

const prevButtonText = computed(() => (contactsLoading.value && contactLoadingSource.value === 'prev' ? '加载中…' : '上一页'));

const nextButtonText = computed(() => (contactsLoading.value && contactLoadingSource.value === 'next' ? '加载中…' : '下一页'));

let contactLoadTimer: number | undefined;

function getPageScrollContainer() {
	if (typeof document === 'undefined') return null;
	const page = document.querySelector('.page');
	return page instanceof HTMLElement ? page : null;
}

function restorePageScroll(scrollTop: number | null) {
	if (scrollTop === null) return;
	const container = getPageScrollContainer();
	if (!container) return;
	window.requestAnimationFrame(() => {
		container.scrollTop = scrollTop;
	});
}

async function loadContacts(options?: {
	preserveScroll?: boolean;
	source?: 'init' | 'refresh' | 'search' | 'prev' | 'next' | 'limit' | 'account';
}) {
	if (!backend.isWails) {
		contacts.value = [];
		contactsTotal.value = 0;
		return;
	}
	contactLoadingSource.value = options?.source ?? 'refresh';
	const preservedScrollTop = options?.preserveScroll ? getPageScrollContainer()?.scrollTop ?? 0 : null;
	contactsLoading.value = true;
	try {
		const resp = await backend.GetContacts(
			contactKeyword.value.trim(),
			-1,
			contactLimit.value,
			contactOffset.value,
		);
		contactsTotal.value = resp.total || 0;
		contacts.value = Array.isArray(resp.items) ? resp.items : [];
	} catch (e) {
		chat.toast('加载联系人失败', String(e));
	} finally {
		contactsLoading.value = false;
		contactLoadingSource.value = null;
		await nextTick();
		restorePageScroll(preservedScrollTop);
	}
}

function scheduleLoadContacts(delayMs = 200) {
	if (contactLoadTimer) window.clearTimeout(contactLoadTimer);
	contactLoadTimer = window.setTimeout(() => {
		void loadContacts({ preserveScroll: true, source: 'search' });
	}, delayMs);
}

function prevContactsPage() {
	contactOffset.value = Math.max(0, contactOffset.value - contactLimit.value);
	void loadContacts({ preserveScroll: true, source: 'prev' });
}

function nextContactsPage() {
	if (contactOffset.value + contactLimit.value >= contactsTotal.value) return;
	contactOffset.value = contactOffset.value + contactLimit.value;
	void loadContacts({ preserveScroll: true, source: 'next' });
}

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

function getContactName(c: Contact) {
	return c.remark || c.nickName || c.alias || c.userName || '未知联系人';
}

function getContactAvatar(c: Contact) {
	return c.smallHeadImgUrl || '';
}

function getContactAvatarFallback(c: Contact) {
	const name = getContactName(c).trim();
	return name ? name.slice(0, 1).toUpperCase() : '?';
}

onMounted(() => {
	void loadContacts({ source: 'init' });
});

watch(
	() => state.value?.account,
	() => {
		contactOffset.value = 0;
		if (contactLoadTimer) window.clearTimeout(contactLoadTimer);
		void loadContacts({ preserveScroll: true, source: 'account' });
	},
);

watch(contactKeyword, () => {
	contactOffset.value = 0;
	scheduleLoadContacts();
});

watch(contactLimit, () => {
	contactOffset.value = 0;
	void loadContacts({ preserveScroll: true, source: 'limit' });
});
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

        <div class="section-header">
            <span class="section-number">02</span>
            <span class="section-title">Contacts</span>
            <div class="section-dot"></div>
        </div>

        <div class="card cardWide">
            <div class="toolbar contactToolbar">
                <div class="contactToolbarInfo">
                    <div class="contactToolbarTitleWrap">
                        <div class="contactToolbarTitle">联系人筛选</div>
	                        <div class="contactToolbarHint">搜索昵称、备注或微信 ID，并快速调整分页大小</div>
                    </div>
                    <div class="contactToolbarMeta">
                        <div class="pill contactMetaPill">范围: {{ contactRangeText }}</div>
                        <div class="pill contactMetaPill">关键字: {{ contactKeyword.trim() ? '已过滤' : '全部' }}</div>
                    </div>
                </div>
                <div class="contactToolbarControls">
                    <div class="contactSearchWrap">
                        <input v-model="contactKeyword" class="input mono contactSearch" placeholder="搜索昵称 / 备注 / ID" />
                    </div>
                    <div class="contactActions">
                        <select v-model.number="contactLimit" class="input mono contactPageSize" title="每页数量">
                            <option :value="20">20/页</option>
                            <option :value="50">50/页</option>
                            <option :value="100">100/页</option>
                            <option :value="200">200/页</option>
                        </select>
                        <button
                            type="button"
                            :class="['btn', 'contactActionBtn']"
                            :disabled="contactsLoading"
                            @click="loadContacts({ source: 'refresh' })"
                        >
                            刷新
                        </button>
                        <button
                            type="button"
                            :class="['btn', 'contactActionBtn']"
                            :disabled="contactsLoading || contactOffset === 0"
                            @click="prevContactsPage"
                        >
                            {{ prevButtonText }}
                        </button>
                        <button
                            type="button"
                            :class="['btn', 'btnBrand', 'contactActionBtn']"
                            :disabled="contactsLoading || contactOffset + contactLimit >= contactsTotal"
                            @click="nextContactsPage"
                        >
                            {{ nextButtonText }}
                        </button>
                    </div>
                </div>
            </div>

            <div :class="['list', { contactListBusy: contactsLoading && hasContacts }]">
                <div v-if="contactsLoading && !hasContacts" class="listItem contactItemEmpty">
                    <div class="listMain">
                        <div class="listTitle">正在加载联系人…</div>
                        <div class="listMeta">请稍候</div>
                    </div>
                </div>
                <div v-else-if="!hasContacts" class="listItem contactItemEmpty">
                    <div class="listMain">
                        <div class="listTitle">暂无联系人</div>
                        <div class="listMeta">可尝试先解密数据或切换账号后刷新</div>
                    </div>
                </div>

                <div v-else v-for="c in contacts" :key="c.userName" class="listItem contactItem">
                    <div class="contactHeader">
                        <div class="contactIdentity">
                            <div class="contactAvatarWrap">
                                <img v-if="getContactAvatar(c)" :src="getContactAvatar(c)" :alt="`${getContactName(c)} 头像`" class="contactAvatar" />
                                <div v-else class="contactAvatar contactAvatarFallback">{{ getContactAvatarFallback(c) }}</div>
                            </div>
                             <div class="contactTitleWrap">
                                 <div class="listTitle contactTitle">{{ getContactName(c) }}</div>
                                 <div class="listMeta mono contactMeta">
                                     {{ c.userName }}<span v-if="c.alias"> · {{ c.alias }}</span>
                                 </div>
                             </div>
                         </div>
				</div>
				<div class="contactFacts">
					<div class="contactFact">localType: {{ c.localType }}</div>
					<div class="contactFact">flag: {{ c.flag }}</div>
					<div class="contactFact">is_in_chat_room: {{ c.isInChatRoom }}</div>
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

.contactToolbar {
	position: sticky;
	top: 0;
	z-index: 5;
	display: grid;
	grid-template-columns: minmax(240px, 1fr) minmax(420px, 1.35fr);
	align-items: center;
	gap: 18px;
	margin: -14px -14px 12px;
	padding: 14px 14px 12px;
	background: linear-gradient(180deg, rgba(14, 18, 28, 0.96), rgba(10, 12, 18, 0.96));
	backdrop-filter: blur(14px);
	border: none;
	border-bottom: 1px solid var(--border);
	border-top-left-radius: var(--radius);
	border-top-right-radius: var(--radius);
	box-shadow: 0 10px 24px rgba(0, 0, 0, 0.18);
}

.contactToolbarInfo {
	display: flex;
	flex-direction: column;
	gap: 10px;
	min-width: 0;
}

.contactToolbarTitleWrap {
	display: flex;
	flex-direction: column;
	gap: 4px;
}

.contactToolbarTitle {
	font-size: 13px;
	font-weight: 700;
	color: var(--text);
	letter-spacing: 0.02em;
}

.contactToolbarHint {
	font-size: 11px;
	line-height: 1.45;
	color: var(--muted);
}

.contactToolbarMeta {
	display: flex;
	flex-wrap: wrap;
	gap: 8px;
}

.contactMetaPill {
	min-width: 140px;
	justify-content: center;
}

.contactToolbarControls {
	display: flex;
	flex-direction: column;
	gap: 10px;
	min-width: 0;
}

.contactSearchWrap {
	width: 100%;
}

.contactSearch {
	width: 100%;
}

.contactActions {
	display: flex;
	justify-content: flex-end;
	align-items: center;
	flex-wrap: wrap;
	gap: 8px;
}

.contactActions :deep(.btn) {
	margin-left: 0;
}

.contactActionBtn {
	min-width: 96px;
}

.contactActions :deep(.btn:disabled) {
	opacity: 0.42;
	color: var(--subtle);
	background: rgba(255, 255, 255, 0.03);
	border-color: rgba(255, 255, 255, 0.08);
	cursor: not-allowed;
	box-shadow: none;
}

.contactActions :deep(.btn:disabled:hover) {
	background: rgba(255, 255, 255, 0.03);
}

.contactPageSize {
	width: 94px;
	flex: 0 0 auto;
}

.contactItem {
	display: flex;
	flex-direction: column;
	align-items: stretch;
	gap: 10px;
	padding: 16px;
	background: var(--panel);
	border: 1px solid var(--border);
	border-radius: var(--radius);
}

.contactItemEmpty {
	min-height: 100px;
	justify-content: center;
	text-align: center;
}

.contactListBusy {
	position: relative;
}

.contactListBusy::after {
	content: '';
	position: absolute;
	inset: 0;
	border-radius: var(--radius);
	background: linear-gradient(180deg, rgba(8, 10, 16, 0.04), rgba(8, 10, 16, 0.1));
	pointer-events: none;
}

.contactListBusy :deep(.contactItem) {
	opacity: 0.76;
}

.contactHeader {
	display: flex;
	align-items: center;
	justify-content: space-between;
	gap: 12px;
}

.contactBadges {
	display: flex;
	align-items: center;
	flex-wrap: wrap;
	justify-content: flex-end;
	gap: 8px;
}

.contactIdentity {
	display: flex;
	align-items: center;
	gap: 10px;
	min-width: 0;
}

.contactTitleWrap {
	min-width: 0;
}

.contactAvatarWrap {
	flex-shrink: 0;
}

.contactAvatar {
	width: 32px;
	height: 32px;
	border-radius: 50%;
	object-fit: cover;
	display: block;
	border: 1px solid rgba(255, 255, 255, 0.08);
	background: rgba(255, 255, 255, 0.04);
}

.contactAvatarFallback {
	display: flex;
	align-items: center;
	justify-content: center;
	font-size: 13px;
	font-weight: 700;
	color: var(--text);
	background: linear-gradient(135deg, rgba(16, 185, 129, 0.22), rgba(59, 130, 246, 0.22));
}

.contactTitle {
	font-size: 15px;
	font-weight: 600;
	min-width: 0;
}

.contactMeta {
	font-size: 11px;
	color: var(--muted);
	word-break: break-all;
}

.contactFacts {
	display: flex;
	flex-wrap: wrap;
	gap: 8px;
}

.contactFact {
	padding: 4px 8px;
	border-radius: 999px;
	border: 1px solid rgba(255, 255, 255, 0.08);
	background: rgba(255, 255, 255, 0.03);
	font-size: 11px;
	line-height: 1.4;
	color: var(--muted);
	font-family: ui-monospace, SFMono-Regular, SFMono-Regular, Consolas, 'Liberation Mono', Menlo, monospace;
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
	.contactToolbar {
		grid-template-columns: 1fr;
		gap: 14px;
	}
	.contactToolbarControls {
		gap: 12px;
	}
	.contactActions {
		justify-content: stretch;
	}
	.contactActions :deep(.btn),
	.contactPageSize {
		width: 100%;
	}
	.contactHeader {
		align-items: flex-start;
		flex-direction: column;
	}
	.contactBadges {
		justify-content: flex-start;
	}
}
</style>
