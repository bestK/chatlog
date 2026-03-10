<script setup lang="ts">
import { computed, inject, onMounted, ref, watch } from 'vue';
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

const contactRangeText = computed(() => {
	if (contactsTotal.value <= 0) return '0';
	const start = Math.min(contactOffset.value + 1, contactsTotal.value);
	const end = Math.min(contactOffset.value + contacts.value.length, contactsTotal.value);
	return `${start}-${end}/${contactsTotal.value}`;
});

let contactLoadTimer: number | undefined;

async function loadContacts() {
	if (!backend.isWails) {
		contacts.value = [];
		contactsTotal.value = 0;
		return;
	}
	contactsLoading.value = true;
	try {
		const resp = await backend.GetContacts(contactKeyword.value.trim(), contactLimit.value, contactOffset.value);
		contactsTotal.value = resp.total || 0;
		contacts.value = Array.isArray(resp.items) ? resp.items : [];
	} catch (e) {
		chat.toast('加载联系人失败', String(e));
	} finally {
		contactsLoading.value = false;
	}
}

function scheduleLoadContacts(delayMs = 200) {
	if (contactLoadTimer) window.clearTimeout(contactLoadTimer);
	contactLoadTimer = window.setTimeout(() => {
		void loadContacts();
	}, delayMs);
}

function prevContactsPage() {
	contactOffset.value = Math.max(0, contactOffset.value - contactLimit.value);
	void loadContacts();
}

function nextContactsPage() {
	if (contactOffset.value + contactLimit.value >= contactsTotal.value) return;
	contactOffset.value = contactOffset.value + contactLimit.value;
	void loadContacts();
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
	void loadContacts();
});

watch(
	() => state.value?.account,
	() => {
		contactOffset.value = 0;
		scheduleLoadContacts(0);
	},
);

watch(contactKeyword, () => {
	contactOffset.value = 0;
	scheduleLoadContacts();
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
                <div class="toolbarGroup">
                    <div class="pill">范围: {{ contactRangeText }}</div>
                    <div class="pill">关键字: {{ contactKeyword.trim() ? '已过滤' : '全部' }}</div>
                </div>
                <div class="toolbarGroup toolbarRight">
                    <input v-model="contactKeyword" class="input mono contactSearch" placeholder="搜索昵称/备注/ID" />
                    <button type="button" class="btn" :disabled="contactsLoading" @click="loadContacts">刷新</button>
                    <button type="button" class="btn" :disabled="contactsLoading || contactOffset === 0" @click="prevContactsPage">
                        上一页
                    </button>
                    <button
                        type="button"
                        class="btn"
                        :disabled="contactsLoading || contactOffset + contactLimit >= contactsTotal"
                        @click="nextContactsPage"
                    >
                        下一页
                    </button>
                </div>
            </div>

            <div class="list">
                <div v-if="contactsLoading" class="listItem contactItemEmpty">
                    <div class="listMain">
                        <div class="listTitle">正在加载联系人…</div>
                        <div class="listMeta">请稍候</div>
                    </div>
                </div>
                <div v-else-if="contacts.length === 0" class="listItem contactItemEmpty">
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
						<span :class="['status-badge-mini', c.isFriend ? 'ok' : 'bad']">{{ c.isFriend ? '好友' : '非好友' }}</span>
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
	margin-bottom: 16px;
	background: transparent;
	border: none;
	padding: 0;
}

.toolbarRight {
	justify-content: flex-end;
}

.contactSearch {
	min-width: 220px;
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

.contactHeader {
	display: flex;
	align-items: center;
	justify-content: space-between;
	gap: 12px;
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
