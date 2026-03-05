<script setup lang="ts">
import { inject } from 'vue';
import { backend } from '../wailsbridge';
import { chatlogKey } from '../composables/chatlogContext';

const injected = inject(chatlogKey);
if (!injected) throw new Error('chatlog not provided');
const chat = injected;

const { dataKey, imgKey, run } = chat;

function saveDataKey() {
	return chat
		.confirm({
			title: '保存数据库密钥',
			message: '确认保存并写入配置？',
			confirmText: '保存',
			cancelText: '取消',
		})
		.then((ok) => (ok ? run(() => backend.SetDataKey(dataKey.value), '已保存数据库密钥') : undefined));
}

function autoDataKey() {
	return run(() => backend.GetDataKey(), '已获取数据库密钥');
}

function saveImgKey() {
	return chat
		.confirm({
			title: '保存图片密钥',
			message: '确认保存并写入配置？',
			confirmText: '保存',
			cancelText: '取消',
		})
		.then((ok) => (ok ? run(() => backend.SetImgKey(imgKey.value), '已保存图片密钥') : undefined));
}

function autoImgKey() {
	return run(() => backend.GetImgKey(), '已获取图片密钥');
}
</script>

<template>
	<div class="grid">
		<div class="card cardWide">
			<div class="cardTitle">查看与设置密钥</div>
			<div class="cardSub">手动设置会写入配置，用于无注入场景。</div>
			<div class="row">
				<div class="field">
					<div class="label">数据库密钥</div>
					<input v-model="dataKey" class="input mono" placeholder="十六进制" />
				</div>
				<button type="button" class="btn" @click="saveDataKey">保存</button>
				<button type="button" class="btn btnBrand" @click="autoDataKey">自动获取</button>
			</div>
			<div class="row">
				<div class="field">
					<div class="label">图片密钥</div>
					<input v-model="imgKey" class="input mono" placeholder="十六进制" />
				</div>
				<button type="button" class="btn" @click="saveImgKey">保存</button>
				<button type="button" class="btn btnBrand" @click="autoImgKey">自动获取</button>
			</div>
		</div>
	</div>
</template>
