<script setup lang="ts">
import { inject, ref, type Ref } from 'vue';
import { backend } from '../wailsbridge';
import { chatlogKey } from '../composables/chatlogContext';

const injected = inject(chatlogKey);
if (!injected) throw new Error('chatlog not provided');
const chat = injected;

const { dataDir, workDir, dataKey, imgKey, run } = chat;

type ActionStatusMode = 'loading' | 'success' | 'error';
type ActionStatusDialog = {
	mode: ActionStatusMode;
	title: string;
	message: string;
	detail?: string;
};

type ActionDialogOptions<T> = {
	loadingRef: Ref<boolean>;
	loadingTitle: string;
	loadingMessage: string;
	successTitle: string;
	successMessage: string;
	failureTitle: string;
	action: () => Promise<T>;
	detail?: (result: T) => string;
};

const loadingDataKey = ref(false);
const loadingImgKey = ref(false);
const loadingDecrypt = ref(false);
const statusDialog = ref<ActionStatusDialog | null>(null);

function maskSecret(secret: string) {
	if (!secret) return '-';
	if (secret.length <= 10) return secret;
	return `${secret.slice(0, 4)}***${secret.slice(-4)}`;
}

function showLoadingDialog(title: string, message: string) {
	statusDialog.value = { mode: 'loading', title, message };
}

function showResultDialog(mode: 'success' | 'error', title: string, message: string, detail = '') {
	statusDialog.value = { mode, title, message, detail };
}

function closeStatusDialog() {
	if (statusDialog.value?.mode === 'loading') return;
	statusDialog.value = null;
}

function formatError(error: unknown) {
	if (error instanceof Error) return error.message;
	if (typeof error === 'string') return error;
	try {
		return JSON.stringify(error);
	} catch {
		return String(error);
	}
}

async function runWithDialog<T>(options: ActionDialogOptions<T>) {
	if (options.loadingRef.value) return;
	options.loadingRef.value = true;
	showLoadingDialog(options.loadingTitle, options.loadingMessage);
	try {
		const result = await options.action();
		await chat.refreshAll();
		showResultDialog('success', options.successTitle, options.successMessage, options.detail ? options.detail(result) : '');
	} catch (e) {
		showResultDialog('error', options.failureTitle, formatError(e));
	} finally {
		options.loadingRef.value = false;
	}
}

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

function saveDataKey() {
	return chat
		.confirm({
			title: '保存数据库密钥',
			message: '确认保存并写入配置？',
			confirmText: '保存',
			cancelText: '取消',
		})
		.then((ok) => (ok ? run(() => backend.SetDataKey(dataKey.value), '已保存数据库密钥') : undefined));
}

function autoDataKey() {
	return runWithDialog<string>({
		loadingRef: loadingDataKey,
		loadingTitle: '正在获取数据库密钥',
		loadingMessage: '请保持微信处于运行与登录状态，正在读取当前账号数据库密钥。',
		successTitle: '数据库密钥获取成功',
		successMessage: '已完成读取并同步到当前页面。',
		failureTitle: '数据库密钥获取失败',
		action: () => backend.GetDataKey(),
		detail: (key) => `密钥预览：${maskSecret(key)}`,
	});
}

function saveImgKey() {
	return chat
		.confirm({
			title: '保存图片密钥',
			message: '确认保存并写入配置？',
			confirmText: '保存',
			cancelText: '取消',
		})
		.then((ok) => (ok ? run(() => backend.SetImgKey(imgKey.value), '已保存图片密钥') : undefined));
}

function autoImgKey() {
	return runWithDialog<string>({
		loadingRef: loadingImgKey,
		loadingTitle: '正在获取图片密钥',
		loadingMessage: '请保持微信处于运行与登录状态，正在读取当前账号图片密钥。',
		successTitle: '图片密钥获取成功',
		successMessage: '已完成读取并同步到当前页面。',
		failureTitle: '图片密钥获取失败',
		action: () => backend.GetImgKey(),
		detail: (key) => `密钥预览：${maskSecret(key)}`,
	});
}

async function decryptNow() {
	const ok = await chat.confirm({
			title: '开始解密',
			message: '确认开始解密数据库到工作目录？',
			confirmText: '开始',
			cancelText: '取消',
	});
	if (!ok) return;
	return runWithDialog<void>({
		loadingRef: loadingDecrypt,
		loadingTitle: '正在解密数据库',
		loadingMessage: '正在解密并写入工作目录，过程可能持续一段时间，请勿关闭程序。',
		successTitle: '解密完成',
		successMessage: '数据库已成功解密，可前往服务页或日志页继续操作。',
		failureTitle: '解密失败',
		action: () => backend.Decrypt(),
		detail: () => `工作目录：${workDir.value || '-'}`,
	});
}
</script>

<template>
	<div class="settings-container">
		<div class="section-header">
			<span class="section-number">01</span>
			<span class="section-title">Global Settings</span>
			<div class="section-dot"></div>
		</div>

		<div class="card cardWide settingsHero">
			<div class="cardSub">Follow the order: Directories → Keys → Decrypt to avoid errors.</div>
			<div class="row steps-row">
				<div class="step-pill">1. Directories</div>
				<div class="step-pill">2. Secret Keys</div>
				<div class="step-pill">3. Execution</div>
				<div class="navHint">Save each step before proceeding</div>
			</div>
		</div>

		<div class="settings-grid">
			<div class="card settingsCard">
				<div class="cardTitle">Directory Config</div>
				<div class="cardSub">Paths for reading data and writing decrypted files.</div>
				<div class="list settingsList">
					<div class="settings-field">
						<div class="label">Data Directory</div>
						<div class="input-with-btn">
							<input v-model="dataDir" class="input mono" />
							<button type="button" class="btn" @click="saveDataDir">Save</button>
						</div>
					</div>
					<div class="settings-field">
						<div class="label">Work Directory</div>
						<div class="input-with-btn">
							<input v-model="workDir" class="input mono" />
							<button type="button" class="btn" @click="saveWorkDir">Save</button>
						</div>
					</div>
				</div>
			</div>

			<div class="card settingsCard">
				<div class="cardTitle">Security Keys</div>
				<div class="cardSub">Manual entry or auto-fetch from WeChat.</div>
				<div class="list settingsList">
					<div class="settings-field">
						<div class="label">Database Key</div>
						<div class="input-with-btn">
							<input v-model="dataKey" class="input mono" placeholder="Hex string" />
							<div class="field-actions">
								<button type="button" class="btn" @click="saveDataKey">Save</button>
								<button
									type="button"
									class="btn btnBrand"
									:disabled="loadingDataKey || loadingImgKey || loadingDecrypt"
									@click="autoDataKey"
								>
									{{ loadingDataKey ? 'Fetching...' : 'Auto' }}
								</button>
							</div>
						</div>
					</div>
					<div class="settings-field">
						<div class="label">Image Key</div>
						<div class="input-with-btn">
							<input v-model="imgKey" class="input mono" placeholder="Hex string" />
							<div class="field-actions">
								<button type="button" class="btn" @click="saveImgKey">Save</button>
								<button
									type="button"
									class="btn btnBrand"
									:disabled="loadingDataKey || loadingImgKey || loadingDecrypt"
									@click="autoImgKey"
								>
									{{ loadingImgKey ? 'Fetching...' : 'Auto' }}
								</button>
							</div>
						</div>
					</div>
				</div>
			</div>
		</div>

		<div class="section-header">
			<span class="section-number">02</span>
			<span class="section-title">Maintenance</span>
			<div class="section-dot"></div>
		</div>

		<div class="card cardWide decrypt-card">
			<div class="cardTitle">Database Decryption</div>
			<div class="cardSub">Ensure directories and keys are saved before starting. This may take time.</div>
			<div class="toolbar">
				<div class="toolbarGroup">
					<div class="pill-mini warn">High Resource Usage</div>
					<div class="navHint">Check disk space before proceeding</div>
				</div>
				<div class="toolbarGroup">
					<button
						type="button"
						class="btn btnBrand"
						:disabled="loadingDataKey || loadingImgKey || loadingDecrypt"
						@click="decryptNow"
					>
						{{ loadingDecrypt ? 'Decrypting...' : 'Run Decryption Now' }}
					</button>
				</div>
			</div>
		</div>
	</div>

	<div v-if="statusDialog" class="modalOverlay" @click.self="closeStatusDialog">
		<div class="modal statusModal">
			<div class="statusHead">
				<div class="modalTitle">{{ statusDialog.title }}</div>
				<span
					:class="['status-badge-mini', statusDialog.mode === 'success' ? 'ok' : statusDialog.mode === 'error' ? 'bad' : 'loading']"
				>
					{{ statusDialog.mode === 'loading' ? 'Processing' : statusDialog.mode === 'success' ? 'Success' : 'Failure' }}
				</span>
			</div>
			<div class="modalMsg">{{ statusDialog.message }}</div>
			<div v-if="statusDialog.detail" class="statusDetail mono">{{ statusDialog.detail }}</div>
			<div class="modalActions">
				<button v-if="statusDialog.mode === 'loading'" type="button" class="btn btnOff" disabled>Processing...</button>
				<button v-else type="button" class="btn btnBrand" @click="closeStatusDialog">Close</button>
			</div>
		</div>
	</div>
</template>

<style scoped>
.settings-container {
	display: flex;
	flex-direction: column;
}

.section-header {
	display: flex;
	align-items: center;
	margin-bottom: 24px;
	border-bottom: 1px solid var(--border);
	padding-bottom: 12px;
	position: relative;
}

.section-number {
	font-size: 11px;
	color: var(--muted);
	margin-right: 12px;
	font-weight: 700;
}

.section-title {
	font-size: 13px;
	font-weight: 600;
	color: var(--text);
	text-transform: uppercase;
	letter-spacing: 0.05em;
}

.section-dot {
	width: 4px;
	height: 4px;
	background-color: var(--brand);
	border-radius: 50%;
	position: absolute;
	bottom: -2.5px;
	left: 0;
}

.settingsHero {
	background: linear-gradient(180deg, rgba(53, 215, 255, 0.05) 0%, transparent 100%);
	border-color: rgba(53, 215, 255, 0.2);
}

.steps-row {
	margin-top: 16px;
	gap: 12px;
}

.step-pill {
	padding: 4px 12px;
	background: var(--panel);
	border: 1px solid var(--border);
	border-radius: 999px;
	font-size: 11px;
	font-weight: 600;
	color: var(--muted);
}

.settings-grid {
	display: grid;
	grid-template-columns: 1fr 1fr;
	gap: 24px;
	margin-bottom: 40px;
}

.settingsList {
	margin-top: 24px;
	display: flex;
	flex-direction: column;
	gap: 20px;
}

.settings-field .label {
	margin-bottom: 8px;
	display: block;
}

.input-with-btn {
	display: flex;
	flex-direction: column;
	gap: 10px;
}

.field-actions {
	display: flex;
	gap: 8px;
}

.field-actions .btn {
	flex: 1;
}

.decrypt-card {
	background: linear-gradient(180deg, rgba(255, 213, 106, 0.05) 0%, transparent 100%);
	border-color: rgba(255, 213, 106, 0.2);
}

.toolbar {
	margin-top: 24px;
	padding: 0;
	background: transparent;
	border: none;
}

.pill-mini {
	padding: 2px 8px;
	font-size: 10px;
	font-weight: 700;
	border-radius: 4px;
	text-transform: uppercase;
}

.pill-mini.warn {
	background: rgba(255, 213, 106, 0.1);
	color: var(--warn);
	border: 1px solid rgba(255, 213, 106, 0.2);
}

.statusModal {
	width: min(500px, 94vw);
	padding: 32px;
}

.statusHead {
	display: flex;
	justify-content: space-between;
	align-items: center;
	margin-bottom: 16px;
}

.status-badge-mini {
	padding: 2px 8px;
	font-size: 10px;
	font-weight: 800;
	border-radius: 4px;
	text-transform: uppercase;
}

.status-badge-mini.ok {
	background: rgba(46, 229, 157, 0.1);
	color: var(--ok);
	border: 1px solid rgba(46, 229, 157, 0.2);
}

.status-badge-mini.bad {
	background: rgba(255, 91, 127, 0.1);
	color: var(--bad);
	border: 1px solid rgba(255, 91, 127, 0.2);
}

.status-badge-mini.loading {
	background: rgba(53, 215, 255, 0.1);
	color: var(--brand);
	border: 1px solid rgba(53, 215, 255, 0.2);
}

.statusDetail {
	margin-top: 20px;
	padding: 16px;
	border-radius: var(--radius-sm);
	background: rgba(0, 0, 0, 0.2);
	border: 1px solid var(--border);
	font-size: 11px;
	color: var(--muted);
	max-height: 200px;
	overflow-y: auto;
}

.modalActions {
	margin-top: 32px;
}

@media (max-width: 900px) {
	.settings-grid {
		grid-template-columns: 1fr;
	}
}
</style>
