<script setup lang="ts">
import type { StatusPill } from '../composables/useChatlog';
import type { State } from '../wailsbridge';

withDefaults(
	defineProps<{
		page: string;
		state: State | null;
		statusPill: StatusPill;
	}>(),
	{},
);
defineEmits<{
	(e: 'refresh'): void;
}>();
</script>

<template>
	<div class="topbar strip-bar">
		<div class="strip-primary">
			<div class="strip-title-group">
				<div class="strip-tag">控制台</div>
				<div class="strip-title">{{ page }}</div>
			</div>
			
			<div v-if="state" class="strip-account">
				<span class="lbl">当前账号</span>
				<span class="val mono">{{ state.account || '未选择' }}</span>
				<span :class="['indicator', 'account-state-dot', state.status === 'online' ? 'on' : 'off']"></span>
			</div>
			<div v-else class="strip-loading">正在连接后端…</div>
		</div>

		<div v-if="state" class="strip-meta">
			<div class="meta-block">
				<div class="meta-item">
					<span class="lbl">PID</span>
					<span class="val mono">{{ state.pid || '-' }}</span>
				</div>
				<div class="divider"></div>
				<div class="meta-item">
					<span class="lbl">版本</span>
					<span class="val mono">{{ state.fullVersion || '-' }}</span>
				</div>
				<div class="divider"></div>
				<div class="meta-item">
					<span class="lbl">最近会话</span>
					<span class="val" :title="state.lastSession || ''">{{ state.lastSession || '-' }}</span>
				</div>
			</div>

			<div class="meta-block services-block">
				<div class="meta-item">
					<span class="lbl">HTTP</span>
					<span :class="['indicator', state.httpEnabled ? 'on' : 'off']"></span>
				</div>
				<div class="divider"></div>
				<div class="meta-item">
					<span class="lbl">自动解密</span>
					<span :class="['indicator', state.autoDecrypt ? 'on' : 'off']"></span>
				</div>
			</div>
		</div>

		<div class="strip-actions">
			<button type="button" class="btn btnBrand btnSm" @click="$emit('refresh')">刷新状态</button>
		</div>
	</div>
</template>

<style scoped>
.strip-bar {
	display: flex;
	align-items: center;
	justify-content: space-between;
	flex-wrap: wrap;
	gap: 16px;
	padding: 8px 12px;
	border-radius: 10px;
	border: 1px solid rgba(255, 255, 255, 0.08);
	background: linear-gradient(180deg, rgba(30, 36, 44, 0.6), rgba(16, 20, 26, 0.8));
	backdrop-filter: blur(16px);
	-webkit-backdrop-filter: blur(16px);
	box-shadow: 0 4px 16px rgba(0, 0, 0, 0.25), inset 0 1px 0 rgba(255, 255, 255, 0.05);
	min-height: 44px;
}

.strip-primary {
	display: flex;
	align-items: center;
	gap: 16px;
	min-width: 0;
	flex-shrink: 1;
}

.strip-title-group {
	display: flex;
	align-items: center;
	gap: 8px;
	min-width: 0;
	flex-shrink: 1;
}

.strip-tag {
	font-size: 10px;
	padding: 2px 7px;
	border-radius: 999px;
	border: 1px solid rgba(53, 215, 255, 0.3);
	background: rgba(53, 215, 255, 0.1);
	color: rgba(207, 246, 255, 0.95);
	white-space: nowrap;
	flex-shrink: 0;
}

.strip-title {
	font-size: 14px;
	font-weight: 650;
	color: rgba(255, 255, 255, 0.95);
	white-space: nowrap;
	overflow: hidden;
	text-overflow: ellipsis;
	min-width: 0;
}

.strip-account {
	display: flex;
	align-items: center;
	gap: 6px;
	padding: 3px 8px;
	border-radius: 6px;
	background: rgba(53, 215, 255, 0.08);
	border: 1px solid rgba(53, 215, 255, 0.15);
	box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.05);
	min-width: 0;
	flex-shrink: 1;
}

.account-state-dot {
	margin-left: 2px;
	flex-shrink: 0;
}

.strip-account .lbl {
	font-size: 10px;
	color: rgba(53, 215, 255, 0.7);
	white-space: nowrap;
	flex-shrink: 0;
}

.strip-account .val {
	font-size: 12px;
	font-weight: 600;
	color: rgba(207, 246, 255, 0.95);
	white-space: nowrap;
	overflow: hidden;
	text-overflow: ellipsis;
	max-width: 140px;
	min-width: 0;
}

.strip-meta {
	display: flex;
	align-items: center;
	gap: 10px;
	flex-grow: 1;
	justify-content: flex-end;
	min-width: 0;
	flex-shrink: 1;
}

.meta-block {
	display: flex;
	align-items: center;
	padding: 4px 10px;
	border-radius: 6px;
	background: rgba(0, 0, 0, 0.25);
	border: 1px solid rgba(255, 255, 255, 0.06);
	box-shadow: inset 0 1px 2px rgba(0, 0, 0, 0.2);
	min-width: 0;
	flex-shrink: 1;
}

.services-block {
	background: rgba(0, 0, 0, 0.35);
}

.meta-item {
	display: flex;
	align-items: center;
	gap: 5px;
	white-space: nowrap;
	min-width: 0;
	flex-shrink: 1;
}

.meta-item .lbl {
	font-size: 10px;
	color: rgba(255, 255, 255, 0.45);
	flex-shrink: 0;
}

.meta-item .val {
	font-size: 11px;
	color: rgba(255, 255, 255, 0.85);
	white-space: nowrap;
	overflow: hidden;
	text-overflow: ellipsis;
	max-width: 110px;
	min-width: 0;
}

.divider {
	width: 1px;
	height: 12px;
	background: rgba(255, 255, 255, 0.1);
	margin: 0 10px;
}

.indicator {
	width: 6px;
	height: 6px;
	border-radius: 50%;
}

.indicator.on {
	background: #4ade80;
	box-shadow: 0 0 8px rgba(74, 222, 128, 0.5);
}

.indicator.off {
	background: rgba(255, 255, 255, 0.15);
}

.strip-actions {
	flex-shrink: 0;
}

.btnSm {
	height: 28px;
	padding: 0 12px;
	font-size: 12px;
	border-radius: 6px;
	font-weight: 600;
}

.strip-loading {
	font-size: 12px;
	color: rgba(255, 255, 255, 0.5);
}

/* Responsive for <= 1200px */
@media (max-width: 1200px) {
	.strip-bar {
		gap: 12px;
	}
	.strip-primary {
		flex-grow: 1;
		flex-basis: 50%;
	}
	.strip-meta {
		order: 3;
		flex-basis: 100%;
		justify-content: flex-start;
	}
	.strip-actions {
		order: 2;
	}
}

/* Responsive for <= 980px */
@media (max-width: 980px) {
	.strip-primary {
		flex-wrap: wrap;
		gap: 10px;
	}
	.strip-title-group {
		flex-wrap: wrap;
	}
	.strip-meta {
		flex-wrap: wrap;
		gap: 8px;
	}
	.meta-block {
		flex-wrap: wrap;
		gap: 8px;
	}
	.divider {
		display: none;
	}
	.strip-account .val {
		max-width: 100px;
	}
	.meta-item .val {
		max-width: 90px;
	}
}

/* Responsive for <= 760px */
@media (max-width: 760px) {
	.strip-bar {
		flex-direction: column;
		align-items: stretch;
	}
	.strip-primary {
		flex-direction: row;
		justify-content: space-between;
		width: 100%;
	}
	.strip-title-group {
		flex-grow: 1;
	}
	.strip-account {
		flex-shrink: 0;
	}
	.strip-meta {
		flex-direction: column;
		align-items: stretch;
	}
	.meta-block {
		justify-content: flex-start;
	}
	.strip-actions {
		order: 4;
		width: 100%;
	}
	.btnSm {
		width: 100%;
		justify-content: center;
	}
}
</style>
