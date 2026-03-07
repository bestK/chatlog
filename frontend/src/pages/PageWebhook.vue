<script setup lang="ts">
import { inject, onMounted, ref } from 'vue';
import { chatlogKey } from '../composables/chatlogContext';
import { backend, type WebhookConfig, type WebhookItem } from '../wailsbridge';

const chat = inject(chatlogKey);
if (!chat) throw new Error('chatlog not provided');
const injected = chat;

const toast = injected.toast;
const run = injected.run;

const cfg = ref<WebhookConfig>({ host: '', delayMs: 0, items: [] });

async function load() {
	try {
		cfg.value = await backend.GetWebhookConfig();
		if (!cfg.value.items) cfg.value.items = [];
	} catch (e) {
		toast('读取失败', String(e));
	}
}

function addItem() {
	cfg.value.items.push({
		description: '',
		type: 'message',
		url: '',
		talker: '',
		sender: '',
		keyword: '',
		disabled: false,
	});
}

async function removeItem(index: number) {
	const ok = await injected.confirm({
		title: '删除规则',
		message: '确定删除该规则？此操作将从配置中移除，保存后生效。',
		confirmText: '删除',
		cancelText: '取消',
		danger: true,
	});
	if (!ok) return;
	cfg.value.items.splice(index, 1);
}

function normalizeItem(it: WebhookItem) {
	it.type = 'message';
	it.description = (it.description || '').trim();
	it.url = (it.url || '').trim();
	it.talker = (it.talker || '').trim();
	it.sender = (it.sender || '').trim();
	it.keyword = (it.keyword || '').trim();
}

function validateItems(items: WebhookItem[]) {
	for (let i = 0; i < items.length; i++) {
		const item = items[i];
		if (!item.url) {
			toast('校验失败', `第 ${i + 1} 条规则缺少 URL`);
			return false;
		}
		if (!item.talker) {
			toast('校验失败', `第 ${i + 1} 条规则缺少 Talker`);
			return false;
		}
	}
	return true;
}

async function save() {
	const ok = await injected.confirm({
		title: '保存 Webhook 配置',
		message: '确认保存并立即应用当前配置？',
		confirmText: '保存',
		cancelText: '取消',
	});
	if (!ok) return;
	const next: WebhookConfig = {
		host: (cfg.value.host || '').trim(),
		delayMs: Number(cfg.value.delayMs || 0),
		items: cfg.value.items.map((x) => ({ ...x })),
	};
	for (const it of next.items) normalizeItem(it);
	if (!validateItems(next.items)) return;
	await run(() => backend.SetWebhookConfig(next), '已保存 Webhook 配置');
}

onMounted(() => {
	void load();
});
</script>

<template>
	<div class="webhook-container">
		<div class="section-header">
			<span class="section-number">01</span>
			<span class="section-title">Configuration</span>
			<div class="section-dot"></div>
		</div>

		<div class="card cardWide">
			<div class="row config-row">
				<div class="field">
					<div class="label">Resource Host</div>
					<input v-model="cfg.host" class="input mono" placeholder="localhost:5030" />
				</div>
				<div class="field delay-field">
					<div class="label">Delay (ms)</div>
					<input v-model.number="cfg.delayMs" class="input mono" type="number" min="0" step="100" />
				</div>
				<div class="config-actions">
					<button type="button" class="btn" @click="load">Refresh</button>
					<button type="button" class="btn btnBrand" @click="save">Save Changes</button>
				</div>
			</div>
		</div>

		<div class="section-header">
			<span class="section-number">02</span>
			<span class="section-title">Rules</span>
			<div class="section-dot"></div>
		</div>

		<div class="card cardWide grow flex-column">
			<div class="toolbar">
				<div class="toolbarGroup">
					<button type="button" class="btn" @click="addItem">Add Rule</button>
					<div class="pill">Rules: {{ cfg.items.length }}</div>
				</div>
				<div class="toolbarGroup">
					<div class="navHint">Changes take effect after saving</div>
				</div>
			</div>

			<div class="rules-list scrollbar">
				<div v-if="cfg.items.length === 0" class="empty-state">
					<div class="listMain">
						<div class="listTitle">No Rules Defined</div>
						<div class="listMeta">Add a rule and save to start pushing messages.</div>
					</div>
				</div>

				<div v-for="(it, idx) in cfg.items" :key="idx" class="rule-item">
					<div class="rule-main">
						<div class="row" style="margin-top: 0">
							<div class="field">
								<div class="label">Description</div>
								<input v-model="it.description" class="input" placeholder="e.g. Forward group messages to local service" />
							</div>
						</div>
						<div class="row">
							<div class="field">
								<div class="label">URL</div>
								<input v-model="it.url" class="input mono" placeholder="http://127.0.0.1:3000/api/v1/webhook" required />
							</div>
						</div>
						<div class="row grid-row">
							<div class="field">
								<div class="label">Talker</div>
								<input v-model="it.talker" class="input mono" placeholder="Group or User name" required />
							</div>
							<div class="field">
								<div class="label">Sender</div>
								<input v-model="it.sender" class="input mono" placeholder="Sender (Optional)" />
							</div>
							<div class="field">
								<div class="label">Keyword</div>
								<input v-model="it.keyword" class="input mono" placeholder="Keyword (Optional)" />
							</div>
						</div>
						<div class="row rule-footer">
							<button
								type="button"
								:class="['btn', it.disabled ? 'btnOff' : 'btnBrand']"
								@click="it.disabled = !it.disabled"
							>
								{{ it.disabled ? 'Disabled' : 'Enabled' }}
							</button>
							<button type="button" class="btn" @click="removeItem(idx)">Delete Rule</button>
						</div>
					</div>
				</div>
			</div>
		</div>
	</div>
</template>

<style scoped>
.webhook-container {
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

.config-row {
	align-items: flex-end;
}

.delay-field {
	max-width: 140px;
}

.config-actions {
	display: flex;
	gap: 12px;
}

.toolbar {
	margin-bottom: 20px;
	padding: 0;
	background: transparent;
	border: none;
}

.rules-list {
	flex: 1;
	overflow-y: auto;
	display: flex;
	flex-direction: column;
	gap: 16px;
	padding-right: 8px;
}

.empty-state {
	padding: 40px;
	text-align: center;
	background: rgba(0, 0, 0, 0.2);
	border: 1px dashed var(--border);
	border-radius: var(--radius);
}

.rule-item {
	background: var(--panel);
	border: 1px solid var(--border);
	border-radius: var(--radius);
	padding: 24px;
}

.grid-row {
	display: grid;
	grid-template-columns: 1fr 1fr 1fr;
	gap: 16px;
}

.rule-footer {
	margin-top: 24px;
	justify-content: flex-start;
	border-top: 1px solid var(--border);
	padding-top: 20px;
}

@media (max-width: 800px) {
	.grid-row {
		grid-template-columns: 1fr;
	}
	.config-row {
		flex-direction: column;
		align-items: stretch;
	}
	.delay-field {
		max-width: none;
	}
}
</style>
