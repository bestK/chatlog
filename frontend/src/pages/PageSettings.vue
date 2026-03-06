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
	<div class="grid settingsGrid">
		<div class="card cardWide settingsHero">
			<div class="cardTitle">设置</div>
			<div class="cardSub">按“目录 → 密钥 → 解密”顺序配置，减少重复操作与误触。</div>
			<div class="row">
				<div class="pill">1. 配置目录</div>
				<div class="pill">2. 配置密钥</div>
				<div class="pill">3. 执行解密</div>
				<div class="navHint">建议每次修改后先保存，再进入下一步</div>
			</div>
		</div>

		<div class="card settingsCard">
			<div class="cardTitle">目录配置</div>
			<div class="cardSub">用于确定读取数据与输出解密文件的位置。</div>
			<div class="list settingsList">
				<div class="listItem settingsItem">
					<div class="listMain settingsMain">
						<div class="label">数据目录</div>
						<input v-model="dataDir" class="input mono" />
					</div>
					<button type="button" class="btn" @click="saveDataDir">保存数据目录</button>
				</div>
				<div class="listItem settingsItem">
					<div class="listMain settingsMain">
						<div class="label">工作目录</div>
						<input v-model="workDir" class="input mono" />
					</div>
					<button type="button" class="btn" @click="saveWorkDir">保存工作目录</button>
				</div>
			</div>
		</div>

		<div class="card settingsCard">
			<div class="cardTitle">密钥配置</div>
			<div class="cardSub">支持手动填写，也可一键读取当前账号密钥。</div>
			<div class="list settingsList">
				<div class="listItem settingsItem">
					<div class="listMain settingsMain">
						<div class="label">数据库密钥</div>
						<input v-model="dataKey" class="input mono" placeholder="十六进制" />
					</div>
					<div class="actions settingsActions">
						<button type="button" class="btn" @click="saveDataKey">保存数据库密钥</button>
						<button
							type="button"
							class="btn btnBrand"
							:disabled="loadingDataKey || loadingImgKey || loadingDecrypt"
							@click="autoDataKey"
						>
							{{ loadingDataKey ? '获取中…' : '自动获取' }}
						</button>
					</div>
				</div>
				<div class="listItem settingsItem">
					<div class="listMain settingsMain">
						<div class="label">图片密钥</div>
						<input v-model="imgKey" class="input mono" placeholder="十六进制" />
					</div>
					<div class="actions settingsActions">
						<button type="button" class="btn" @click="saveImgKey">保存图片密钥</button>
						<button
							type="button"
							class="btn btnBrand"
							:disabled="loadingDataKey || loadingImgKey || loadingDecrypt"
							@click="autoImgKey"
						>
							{{ loadingImgKey ? '获取中…' : '自动获取' }}
						</button>
					</div>
				</div>
			</div>
		</div>

		<div class="card cardWide settingsDecrypt">
			<div class="cardTitle">解密执行</div>
			<div class="cardSub">确认目录和密钥都已保存后，再开始解密数据库到工作目录。</div>
			<div class="toolbar">
				<div class="toolbarGroup">
					<div class="pill pillWarn">高耗时操作</div>
					<div class="navHint">解密会扫描并写入工作目录，请确保磁盘空间充足</div>
				</div>
				<div class="toolbarGroup">
					<button
						type="button"
						class="btn btnBrand"
						:disabled="loadingDataKey || loadingImgKey || loadingDecrypt"
						@click="decryptNow"
					>
						{{ loadingDecrypt ? '解密中…' : '开始解密' }}
					</button>
				</div>
			</div>
		</div>
	</div>

	<div v-if="statusDialog" class="modalOverlay" @click.self="closeStatusDialog">
		<div class="modal statusModal">
			<div class="statusHead">
				<div class="modalTitle">{{ statusDialog.title }}</div>
				<span :class="['pill', statusDialog.mode === 'success' ? 'pillOk' : statusDialog.mode === 'error' ? 'pillBad' : '']">
					{{ statusDialog.mode === 'loading' ? '处理中' : statusDialog.mode === 'success' ? '成功' : '失败' }}
				</span>
			</div>
			<div class="modalMsg">{{ statusDialog.message }}</div>
			<div v-if="statusDialog.detail" class="statusDetail mono">{{ statusDialog.detail }}</div>
			<div class="modalActions">
				<button v-if="statusDialog.mode === 'loading'" type="button" class="btn btnOff" disabled>处理中…</button>
				<button v-else type="button" class="btn btnBrand" @click="closeStatusDialog">我知道了</button>
			</div>
		</div>
	</div>
</template>

<style scoped>
.settingsGrid {
	align-items: start;
}

.settingsHero {
	background: linear-gradient(180deg, rgba(53, 215, 255, 0.12), rgba(255, 255, 255, 0.04));
	border-color: rgba(53, 215, 255, 0.32);
}

.settingsCard {
	display: flex;
	flex-direction: column;
	gap: 2px;
}

.settingsList {
	margin-top: 8px;
}

.settingsItem {
	align-items: flex-end;
}

.settingsMain {
	flex: 1;
	min-width: 0;
	width: 100%;
}

.settingsActions {
	justify-content: flex-end;
}

.settingsActions .btn:disabled,
.settingsDecrypt .btn:disabled {
	opacity: 0.6;
	cursor: not-allowed;
}

.settingsDecrypt {
	background: linear-gradient(180deg, rgba(255, 213, 106, 0.12), rgba(255, 255, 255, 0.04));
	border-color: rgba(255, 213, 106, 0.3);
}

.statusModal {
	width: min(560px, 96vw);
}

.statusHead {
	display: flex;
	justify-content: space-between;
	align-items: center;
	gap: 10px;
}

.statusDetail {
	margin-top: 10px;
	padding: 10px 12px;
	border-radius: 10px;
	border: 1px solid var(--border);
	background: rgba(0, 0, 0, 0.22);
	font-size: 12px;
	line-height: 1.45;
}

@media (max-width: 980px) {
	.settingsItem {
		flex-direction: column;
		align-items: stretch;
	}

	.settingsActions {
		width: 100%;
		justify-content: flex-start;
	}
}
</style>
