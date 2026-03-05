<script setup lang="ts">
import { inject } from 'vue';
import { backend } from '../wailsbridge';
import { chatlogKey } from '../composables/chatlogContext';

const injected = inject(chatlogKey);
if (!injected) throw new Error('chatlog not provided');
const chat = injected;

const { dataDir, workDir, run } = chat;

function saveDataDir() {
	return chat
		.confirm({
			title: '保存数据目录',
			message: '确认保存并写入配置？',
			confirmText: '保存',
			cancelText: '取消',
		})
		.then((ok) => (ok ? run(() => backend.SetDataDir(dataDir.value), '已保存数据目录') : undefined));
}

function saveWorkDir() {
	return chat
		.confirm({
			title: '保存工作目录',
			message: '确认保存并写入配置？',
			confirmText: '保存',
			cancelText: '取消',
		})
		.then((ok) => (ok ? run(() => backend.SetWorkDir(workDir.value), '已保存工作目录') : undefined));
}
</script>

<template>
	<div class="grid">
		<div class="card cardWide">
			<div class="cardTitle">路径与参数</div>
			<div class="cardSub">修改会写入配置并影响后续解密与服务。</div>
			<div class="row">
				<div class="field">
					<div class="label">数据目录</div>
					<input v-model="dataDir" class="input mono" />
				</div>
				<button type="button" class="btn" @click="saveDataDir">保存</button>
			</div>
			<div class="row">
				<div class="field">
					<div class="label">工作目录</div>
					<input v-model="workDir" class="input mono" />
				</div>
				<button type="button" class="btn" @click="saveWorkDir">保存</button>
			</div>
		</div>
	</div>
</template>
