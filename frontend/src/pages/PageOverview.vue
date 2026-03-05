<script setup lang="ts">
import { inject } from 'vue';
import { chatlogKey } from '../composables/chatlogContext';
import { maskKey } from '../composables/useChatlog';
import { backend } from '../wailsbridge';

const chat = inject(chatlogKey);
if (!chat) throw new Error('chatlog not provided');

const { dataKey, imgKey, dataDir, workDir, state, run } = chat;

function getDataKey() {
	return run(() => backend.GetDataKey(), '已获取数据库密钥');
}

function getImgKey() {
	return run(() => backend.GetImgKey(), '已获取图片密钥');
}

function toggleHTTP() {
	return run(() => (state.value?.httpEnabled ? backend.StopHTTP() : backend.StartHTTP()), state.value?.httpEnabled ? '已停止 HTTP 服务' : '已启动 HTTP 服务');
}

function toggleAutoDecrypt() {
	return run(() => backend.SetAutoDecrypt(!state.value?.autoDecrypt), state.value?.autoDecrypt ? '已停止自动解密' : '已开启自动解密');
}
</script>

<template>
	<div class="grid">
		<div class="card">
			<div class="cardTitle">密钥</div>
			<div class="cardSub">数据库密钥与图片密钥，用于解密与多媒体处理。</div>
			<div class="row">
				<button type="button" class="btn btnBrand" @click="getDataKey">获取数据库密钥</button>
				<button type="button" class="btn" @click="getImgKey">获取图片密钥</button>
			</div>
			<div class="row">
				<div class="pill mono">DataKey：{{ maskKey(dataKey) }}</div>
				<div class="pill mono">ImgKey：{{ maskKey(imgKey) }}</div>
			</div>
		</div>

		<div class="card">
			<div class="cardTitle">服务</div>
			<div class="cardSub">本地 HTTP & MCP 服务，便于 API 查询与集成 AI。</div>
			<div class="row">
				<button
					type="button"
					:class="['btn', state?.httpEnabled ? '' : 'btnBrand']"
					@click="toggleHTTP"
				>
					{{ state?.httpEnabled ? '停止 HTTP' : '启动 HTTP' }}
				</button>
				<button
					type="button"
					:class="['btn', state?.autoDecrypt ? '' : 'btnBrand']"
					@click="toggleAutoDecrypt"
				>
					{{ state?.autoDecrypt ? '停止自动解密' : '开启自动解密' }}
				</button>
			</div>
			<div v-if="state?.httpAddr" class="row">
				<div class="pill mono">API：http://{{ state.httpAddr }}/api/v1/session</div>
				<div class="pill mono">MCP：http://{{ state.httpAddr }}/mcp</div>
			</div>
		</div>

		<div class="card cardWide">
			<div class="cardTitle">路径</div>
			<div class="cardSub">数据目录为微信原始数据所在路径，工作目录为解密输出。</div>
			<div class="row">
				<div class="pill mono">数据目录：{{ dataDir || '-' }}</div>
				<div class="pill mono">工作目录：{{ workDir || '-' }}</div>
			</div>
		</div>
	</div>
</template>
