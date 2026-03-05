<script setup lang="ts">
import { inject } from 'vue';
import { backend } from '../wailsbridge';
import { chatlogKey } from '../composables/chatlogContext';

const chat = inject(chatlogKey);
if (!chat) throw new Error('chatlog not provided');

const { instances, run } = chat;

function switchTo(pid: number) {
	return run(() => backend.SwitchToPID(pid), '已切换账号');
}
</script>

<template>
	<div class="grid">
		<div class="card cardWide">
			<div class="cardTitle">运行中的微信进程</div>
			<div class="cardSub">选择一个进程作为当前账号（用于获取密钥与解密）。</div>
			<div class="list">
				<div v-if="instances.length === 0" class="listItem">
					<div class="listMain">
						<div class="listTitle">未检测到微信进程</div>
						<div class="listMeta">请先启动并登录微信</div>
					</div>
				</div>
				<div v-else v-for="x in instances" :key="x.pid" class="listItem">
					<div class="listMain">
						<div class="listTitle">
							{{ x.name || '未命名' }}
							<span class="pill pillXs mono">PID {{ x.pid }}</span>
						</div>
						<div class="listMeta mono">{{ x.fullVersion || '-' }} · {{ x.platform || '-' }} · {{ x.status || '-' }}</div>
						<div class="listMeta mono">{{ x.dataDir || '' }}</div>
					</div>
					<button type="button" class="btn btnBrand" @click="switchTo(x.pid)">切换</button>
				</div>
			</div>
		</div>
	</div>
</template>
