<script setup lang="ts">
import { inject } from 'vue';
import { chatlogKey } from '../composables/chatlogContext';
import { maskKey } from '../composables/useChatlog';
import { backend } from '../wailsbridge';

const injected = inject(chatlogKey);
if (!injected) throw new Error('chatlog not provided');
const chat = injected;

const { dataKey, imgKey, dataDir, workDir, state, run } = chat;

async function toggleHTTP() {
	if (state.value?.httpEnabled) {
		const ok = await chat.confirm({
			title: '停止 HTTP 服务',
			message: '确认停止 HTTP 服务？停止后 API 与 MCP 接口将不可访问。',
			confirmText: '停止',
			cancelText: '取消',
			danger: true,
		});
		if (!ok) return;
		return run(() => backend.StopHTTP(), '已停止 HTTP 服务');
	}
	return run(() => backend.StartHTTP(), '已启动 HTTP 服务');
}

</script>

<template>
	<div class="grid overviewGrid">
		<div class="card cardWide overviewHero">
			<div class="cardTitle">密钥摘要</div>
			<div class="row">
				<div class="pill mono">DataKey：{{ maskKey(dataKey) }}</div>
				<div class="pill mono">ImgKey：{{ maskKey(imgKey) }}</div>
			</div>
		</div>

		<div class="card overviewMain">
			<div class="cardTitle">路径信息</div>
			<div class="cardSub">用于解密与查询的核心目录。</div>
			<div class="row">
				<div class="pill mono">数据目录：{{ dataDir || '-' }}</div>
				<div class="pill mono">工作目录：{{ workDir || '-' }}</div>
			</div>
		</div>

		<div class="card overviewAction">
			<div class="cardTitle">快速操作</div>
			<div class="cardSub">服务开关、状态刷新与接口入口。</div>
			<div class="row">
				<button
					type="button"
					:class="['btn', state?.httpEnabled ? 'btnDanger' : 'btnBrand']"
					@click="toggleHTTP"
				>
					{{ state?.httpEnabled ? '停止 HTTP' : '启动 HTTP' }}
				</button>
				<button type="button" class="btn" @click="run(() => backend.Refresh(), '已刷新')">刷新状态</button>
			</div>
			<div v-if="state?.httpAddr" class="row">
				<div class="pill mono">API：http://{{ state.httpAddr }}/api/v1/session</div>
				<div class="pill mono">MCP：http://{{ state.httpAddr }}/mcp</div>
			</div>
		</div>
	</div>
</template>
