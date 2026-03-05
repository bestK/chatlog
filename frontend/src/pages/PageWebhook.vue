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
	it.url = (it.url || '').trim();
	it.talker = (it.talker || '').trim();
	it.sender = (it.sender || '').trim();
	it.keyword = (it.keyword || '').trim();
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
	await run(() => backend.SetWebhookConfig(next), '已保存 Webhook 配置');
}

onMounted(() => {
	void load();
});
</script>

<template>
	<div class="grid" style="height: 100%">
		<div class="card cardWide cardFill">
			<div class="cardTitle">Webhook</div>
			<div class="cardSub">自动解密触发时按过滤条件推送新消息到指定 URL。</div>

			<div class="row">
				<div class="field">
					<div class="label">资源 Host</div>
					<input v-model="cfg.host" class="input mono" placeholder="localhost:5030" />
				</div>
				<div class="field" style="max-width: 220px">
					<div class="label">延迟（ms）</div>
					<input v-model.number="cfg.delayMs" class="input mono" type="number" min="0" step="100" />
				</div>
				<button type="button" class="btn" @click="load">刷新</button>
				<button type="button" class="btn btnBrand" @click="save">保存</button>
			</div>

			<div class="toolbar">
				<div class="toolbarGroup">
					<button type="button" class="btn" @click="addItem">添加规则</button>
					<div class="pill">规则数：{{ cfg.items.length }}</div>
				</div>
				<div class="toolbarGroup">
					<div class="navHint">保存后自动生效</div>
				</div>
			</div>

			<div class="surface scrollbar grow" style="overflow: auto; display: flex; flex-direction: column; gap: 10px">
				<div v-if="cfg.items.length === 0" class="listItem">
					<div class="listMain">
						<div class="listTitle">暂无规则</div>
						<div class="listMeta">添加一条规则后保存，即可开始推送。</div>
					</div>
				</div>

				<div v-for="(it, idx) in cfg.items" :key="idx" class="listItem" style="align-items: flex-start">
					<div class="listMain" style="width: 100%">
						<div class="row" style="margin-top: 0">
							<div class="field">
								<div class="label">URL</div>
								<input v-model="it.url" class="input mono" placeholder="http://127.0.0.1:3000/api/v1/webhook" />
							</div>
						</div>
						<div class="row" style="margin-top: 10px">
							<div class="field">
								<div class="label">Talker</div>
								<input v-model="it.talker" class="input mono" placeholder="群聊/私聊名称" />
							</div>
							<div class="field">
								<div class="label">Sender</div>
								<input v-model="it.sender" class="input mono" placeholder="发送者（可选）" />
							</div>
							<div class="field">
								<div class="label">Keyword</div>
								<input v-model="it.keyword" class="input mono" placeholder="关键词（可选）" />
							</div>
						</div>
						<div class="row" style="margin-top: 10px">
							<button
								type="button"
								:class="['btn', it.disabled ? 'btnOff' : 'btnBrand']"
								@click="it.disabled = !it.disabled"
							>
								{{ it.disabled ? '已禁用' : '已启用' }}
							</button>
							<button type="button" class="btn" @click="removeItem(idx)">删除</button>
						</div>
					</div>
				</div>
			</div>
		</div>
	</div>
</template>
