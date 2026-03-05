<script setup lang="ts">
import { inject } from 'vue';
import { backend } from '../wailsbridge';
import { chatlogKey } from '../composables/chatlogContext';

const injected = inject(chatlogKey);
if (!injected) throw new Error('chatlog not provided');
const chat = injected;

const { state, run } = chat;

function decrypt() {
	return chat
		.confirm({
			title: '开始解密',
			message: '确认开始解密数据库到工作目录？',
			confirmText: '开始',
			cancelText: '取消',
		})
		.then((ok) => (ok ? run(() => backend.Decrypt(), '解密完成') : undefined));
}
</script>

<template>
	<div class="grid">
		<div class="card cardWide">
			<div class="cardTitle">解密数据</div>
			<div class="cardSub">将微信数据库解密到工作目录，供 HTTP API 与查询使用。</div>
			<div class="row">
				<div class="pill mono">数据目录：{{ state?.dataDir || '-' }}</div>
				<div class="pill mono">工作目录：{{ state?.workDir || '-' }}</div>
			</div>
			<div class="row">
				<button type="button" class="btn btnBrand" @click="decrypt">开始解密</button>
			</div>
		</div>
	</div>
</template>
