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
			<div class="toolbar accountToolbar">
				<div class="toolbarGroup">
					<div class="pill">当前检测：{{ instances.length }} 个进程</div>
					<div class="navHint">先看“在线状态”，再切换账号</div>
				</div>
			</div>
			<div class="list">
				<div v-if="instances.length === 0" class="listItem accountItem accountItemEmpty">
					<div class="listMain">
						<div class="listTitle">未检测到微信进程</div>
						<div class="listMeta">请先启动并登录微信</div>
					</div>
				</div>
				<div v-else v-for="x in instances" :key="x.pid" class="listItem accountItem">
					<div class="accountTop">
						<div class="listMain accountMain">
							<div class="accountHeader">
								<div class="listTitle accountTitle">{{ x.name || '未命名进程' }}</div>
								<span :class="['pill', 'pillXs', x.status === 'online' ? 'pillOk' : x.status === 'offline' ? 'pillBad' : '']">
									{{ x.status === 'online' ? '在线' : x.status === 'offline' ? '离线' : x.status || '未知' }}
								</span>
							</div>
							<div class="listMeta mono accountMetaLine">
								PID {{ x.pid }} · 版本 {{ x.fullVersion || '-' }} · 平台 {{ x.platform || '-' }}
							</div>
						</div>
						<div class="accountActionWrap">
							<button type="button" class="btn btnBrand accountAction" @click="switchTo(x.pid)">切换到该账号</button>
						</div>
					</div>
					<div class="accountPathWrap">
						<div class="accountPathLabel">数据目录</div>
						<div class="listMeta mono accountPath" :title="x.dataDir || '-'">{{ x.dataDir || '-' }}</div>
					</div>
				</div>
			</div>
		</div>
	</div>
</template>

<style scoped>
.accountToolbar {
	margin-top: 12px;
	margin-bottom: 4px;
}

.accountItem {
	display: flex;
	flex-direction: column;
	align-items: stretch;
	justify-content: flex-start;
	gap: 12px;
	padding: 12px 14px;
	background: linear-gradient(180deg, rgba(255, 255, 255, 0.06), rgba(255, 255, 255, 0.03));
}

.accountItemEmpty {
	min-height: 72px;
	justify-content: center;
}

.accountTop {
	display: grid;
	grid-template-columns: minmax(0, 1fr) auto;
	align-items: center;
	gap: 12px;
}

.accountMain {
	min-width: 0;
	gap: 6px;
}

.accountHeader {
	display: flex;
	align-items: center;
	gap: 10px;
	flex-wrap: wrap;
}

.accountTitle {
	font-size: 14px;
	font-weight: 700;
}

.accountMetaLine {
	line-height: 1.45;
}

.accountPathWrap {
	margin-top: 2px;
	padding: 8px 10px;
	border-radius: 10px;
	border: 1px solid var(--border);
	background: rgba(0, 0, 0, 0.18);
}

.accountPathLabel {
	font-size: 11px;
	color: var(--subtle);
	margin-bottom: 4px;
}

.accountPath {
	line-height: 1.45;
	word-break: break-all;
}

.accountActionWrap {
	min-width: 142px;
	display: flex;
	align-items: center;
	justify-content: flex-end;
}

.accountAction {
	min-width: 120px;
}

@media (max-width: 980px) {
	.accountTop {
		grid-template-columns: 1fr;
		align-items: start;
	}

	.accountActionWrap {
		width: 100%;
		justify-content: flex-start;
	}

	.accountAction {
		min-width: 0;
	}
}
</style>
