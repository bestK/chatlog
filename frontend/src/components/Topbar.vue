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
				<div class="strip-tag">Console</div>
				<div class="strip-title">{{ page }}</div>
			</div>

			<div v-if="state" class="strip-account">
				<span class="lbl">Account</span>
				<span class="val mono">{{ state.account || 'None' }}</span>
				<span :class="['indicator', 'account-state-dot', state.status === 'online' ? 'on' : 'off']"></span>
			</div>
			<div v-else class="strip-loading">Connecting...</div>
		</div>

		<div v-if="state" class="strip-meta">
			<div class="meta-block">
				<div class="meta-item">
					<span class="lbl">PID</span>
					<span class="val mono">{{ state.pid || '-' }}</span>
				</div>
				<div class="divider"></div>
				<div class="meta-item">
					<span class="lbl">Version</span>
					<span class="val mono">{{ state.fullVersion || '-' }}</span>
				</div>
				<div class="divider"></div>
				<div class="meta-item">
					<span class="lbl">Last Session</span>
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
					<span class="lbl">Auto Decrypt</span>
					<span :class="['indicator', state.autoDecrypt ? 'on' : 'off']"></span>
				</div>
			</div>
		</div>

		<div class="strip-actions">
			<button type="button" class="btn btnBrand btnSm" @click="$emit('refresh')">Refresh Status</button>
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
	padding: 10px 16px;
	border-radius: 12px;
	border: 1px solid var(--border);
	background: rgba(255, 255, 255, 0.02);
	backdrop-filter: blur(16px);
	box-shadow: 0 4px 16px rgba(0, 0, 0, 0.25);
	min-height: 48px;
}

.strip-primary {
	display: flex;
	align-items: center;
	gap: 20px;
	min-width: 0;
	flex-shrink: 1;
}

.strip-title-group {
	display: flex;
	align-items: center;
	gap: 12px;
}

.strip-tag {
	font-size: 9px;
	padding: 2px 8px;
	border-radius: 4px;
	border: 1px solid rgba(53, 215, 255, 0.3);
	background: rgba(53, 215, 255, 0.1);
	color: var(--brand);
	text-transform: uppercase;
	font-weight: 800;
	letter-spacing: 0.05em;
}

.strip-title {
	font-size: 15px;
	font-weight: 700;
	color: var(--text);
	white-space: nowrap;
}

.strip-account {
	display: flex;
	align-items: center;
	gap: 8px;
	padding: 4px 10px;
	border-radius: 6px;
	background: rgba(255, 255, 255, 0.04);
	border: 1px solid var(--border);
}

.strip-account .lbl {
	font-size: 10px;
	color: var(--muted);
	font-weight: 700;
	text-transform: uppercase;
}

.strip-account .val {
	font-size: 12px;
	font-weight: 600;
	color: var(--text);
	max-width: 150px;
	overflow: hidden;
	text-overflow: ellipsis;
}

.strip-meta {
	display: flex;
	align-items: center;
	gap: 12px;
	flex-grow: 1;
	justify-content: flex-end;
}

.meta-block {
	display: flex;
	align-items: center;
	padding: 4px 12px;
	border-radius: 8px;
	background: rgba(0, 0, 0, 0.2);
	border: 1px solid var(--border);
}

.meta-item {
	display: flex;
	align-items: center;
	gap: 6px;
	white-space: nowrap;
}

.meta-item .lbl {
	font-size: 9px;
	color: var(--muted);
	text-transform: uppercase;
	font-weight: 700;
}

.meta-item .val {
	font-size: 11px;
	color: var(--text);
	max-width: 120px;
	overflow: hidden;
	text-overflow: ellipsis;
}

.divider {
	width: 1px;
	height: 12px;
	background: var(--border);
	margin: 0 12px;
}

.indicator {
	width: 6px;
	height: 6px;
	border-radius: 50%;
}

.indicator.on {
	background: var(--ok);
	box-shadow: 0 0 8px var(--ok);
}

.indicator.off {
	background: var(--subtle);
}

.btnSm {
	height: 28px;
	font-size: 11px;
	font-weight: 700;
}

@media (max-width: 1200px) {
	.strip-meta {
		display: none;
	}
}
</style>
