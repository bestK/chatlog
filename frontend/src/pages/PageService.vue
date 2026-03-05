<script setup lang="ts">
import { inject } from 'vue';
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
		.then((ok) => (ok ? run(() => backend.SetHTTPAddr(httpAddr.value), '已保存') : undefined));
}

function toggleHTTP() {
	return run(() => (state.value?.httpEnabled ? backend.StopHTTP() : backend.StartHTTP()), state.value?.httpEnabled ? '已停止' : '已启动');
}

function toggleAutoDecrypt() {
	return run(() => backend.SetAutoDecrypt(!state.value?.autoDecrypt), state.value?.autoDecrypt ? '已停止自动解密' : '已开启自动解密');
}
</script>

<template>
	<div class="grid">
		<div class="card cardWide">
			<div class="cardTitle">HTTP & MCP</div>
			<div class="cardSub">启动后可访问：/api/v1/... 与 /mcp。</div>
			<div class="row">
				<div class="field">
					<div class="label">监听地址</div>
					<input v-model="httpAddr" class="input mono" placeholder="127.0.0.1:5030" />
				</div>
				<button type="button" class="btn" @click="saveAddr">保存</button>
				<button
					type="button"
					:class="['btn', state?.httpEnabled ? '' : 'btnBrand']"
					@click="toggleHTTP"
				>
					{{ state?.httpEnabled ? '停止' : '启动' }}
				</button>
			</div>
			<div v-if="state?.httpAddr" class="row">
				<div class="pill mono">API：http://{{ state.httpAddr }}/api/v1/session</div>
				<div class="pill mono">MCP：http://{{ state.httpAddr }}/mcp</div>
			</div>
			<div class="row">
				<button
					type="button"
					:class="['btn', state?.autoDecrypt ? '' : 'btnBrand']"
					@click="toggleAutoDecrypt"
				>
					{{ state?.autoDecrypt ? '停止自动解密' : '开启自动解密' }}
				</button>
			</div>
		</div>
	</div>
</template>
