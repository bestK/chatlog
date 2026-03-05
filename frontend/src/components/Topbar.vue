<script setup lang="ts">
import type { StatusPill } from '../composables/useChatlog';
import type { State } from '../wailsbridge';

withDefaults(
	defineProps<{
		page: string;
		state: State | null;
		statusPill: StatusPill;
		showGetKeys?: boolean;
		showDecrypt?: boolean;
	}>(),
	{
		showGetKeys: false,
		showDecrypt: false,
	},
);
defineEmits<{
	(e: 'refresh'): void;
	(e: 'getkeys'): void;
	(e: 'decrypt'): void;
}>();
</script>

<template>
	<div class="topbar">
		<div class="topLeft">
			<div class="title">{{ page }}</div>
			<div class="subtitle">
				{{
					state
						? `账号：${state.account || '未选择'} · 版本：${state.fullVersion || '-'} · PID：${state.pid || '-'}`
						: '正在连接后端…'
				}}
			</div>
			<div class="pillRow">
				<span :class="statusPill.cls">{{ statusPill.text }}</span>
				<span :class="['pill', state?.httpEnabled ? 'pillOk' : '']">HTTP：{{ state?.httpEnabled ? '已启动' : '未启动' }}</span>
				<span :class="['pill', state?.autoDecrypt ? 'pillOk' : '']">自动解密：{{ state?.autoDecrypt ? '已开启' : '未开启' }}</span>
				<span v-if="state?.lastSession" class="pill">最近会话：{{ state.lastSession }}</span>
			</div>
		</div>
		<div class="actions">
			<button type="button" class="btn" @click="$emit('refresh')">刷新</button>
			<button v-if="showGetKeys" type="button" class="btn btnBrand" @click="$emit('getkeys')">获取密钥</button>
			<button v-if="showDecrypt" type="button" class="btn" @click="$emit('decrypt')">解密数据</button>
		</div>
	</div>
</template>
