import { computed, onMounted, ref } from 'vue';
import { backend, type Instance, type State } from '../wailsbridge';

export type Page = '概览' | '账号' | '密钥' | '解密' | '服务' | 'Webhook' | '设置' | '日志';
export type Toast = { id: string; title: string; message: string };
export type StatusPill = { cls: string; text: string };

function isState(v: unknown): v is State {
	if (!v || typeof v !== 'object') return false;
	const r = v as Record<string, unknown>;
	return (
		typeof r.account === 'string' &&
		typeof r.platform === 'string' &&
		typeof r.fullVersion === 'string' &&
		typeof r.dataDir === 'string' &&
		typeof r.dataKey === 'string' &&
		typeof r.imgKey === 'string' &&
		typeof r.workDir === 'string' &&
		typeof r.httpEnabled === 'boolean' &&
		typeof r.httpAddr === 'string' &&
		typeof r.autoDecrypt === 'boolean' &&
		typeof r.lastSession === 'string' &&
		typeof r.pid === 'number' &&
		typeof r.exePath === 'string' &&
		typeof r.status === 'string'
	);
}

export function maskKey(s: string) {
	if (!s) return '';
	if (s.length <= 10) return s;
	return `${s.slice(0, 4)}***${s.slice(-4)}`;
}

export function useChatlog() {
	const page = ref<Page>('概览');
	const state = ref<State | null>(null);
	const instances = ref<Instance[]>([]);
	const httpAddr = ref('');
	const workDir = ref('');
	const dataDir = ref('');
	const dataKey = ref('');
	const imgKey = ref('');
	const toasts = ref<Toast[]>([]);

	const nav: Array<{ name: Page; hint: string }> = [
		{ name: '概览', hint: '快捷操作' },
		{ name: '账号', hint: '进程/历史' },
		{ name: '密钥', hint: '数据/图片' },
		{ name: '解密', hint: '数据库' },
		{ name: '服务', hint: 'HTTP/MCP' },
		{ name: 'Webhook', hint: '回调' },
		{ name: '设置', hint: '路径/参数' },
		{ name: '日志', hint: '诊断' },
	];

	const statusPill = computed<StatusPill>(() => {
		const st = state.value;
		if (!st) return { cls: 'pill', text: '未连接' };
		if (st.status === 'online') return { cls: 'pill pillOk', text: '在线' };
		if (st.status === 'offline') return { cls: 'pill pillBad', text: '离线' };
		return { cls: 'pill', text: st.status || '未知' };
	});

	const previewBanner = computed(() => {
		return backend.isWails ? '' : '浏览器预览模式：后端能力不可用';
	});

	function toast(title: string, message: string) {
		const id = `${Date.now()}-${Math.random()}`;
		toasts.value = [...toasts.value, { id, title, message }];
		setTimeout(() => {
			toasts.value = toasts.value.filter((x) => x.id !== id);
		}, 3800);
	}

	async function refreshAll() {
		try {
			const st = await backend.GetState();
			state.value = st;
			httpAddr.value = st.httpAddr || '';
			workDir.value = st.workDir || '';
			dataDir.value = st.dataDir || '';
			dataKey.value = st.dataKey || '';
			imgKey.value = st.imgKey || '';
			instances.value = await backend.ListInstances();
		} catch (e) {
			toast('刷新失败', String(e));
		}
	}

	async function run(action: () => Promise<unknown>, okMsg: string) {
		try {
			await action();
			toast('完成', okMsg);
			await refreshAll();
		} catch (e) {
			toast('操作失败', String(e));
		}
	}

	onMounted(async () => {
		if (backend.isWails) {
			try {
				await backend.EnableStateEvents(true);
			} catch {
			}
		}
		await refreshAll();
		const off = backend.EventsOn('state', (payload) => {
			if (!isState(payload)) return;
			state.value = payload;
			httpAddr.value = payload.httpAddr || '';
			workDir.value = payload.workDir || '';
			dataDir.value = payload.dataDir || '';
			dataKey.value = payload.dataKey || '';
			imgKey.value = payload.imgKey || '';
		});
		if (off) {
			window.addEventListener(
				'beforeunload',
				() => {
					off();
					if (backend.isWails) {
						void backend.EnableStateEvents(false);
					}
				},
				{ once: true },
			);
		}
	});

	return {
		page,
		nav,
		state,
		instances,
		httpAddr,
		workDir,
		dataDir,
		dataKey,
		imgKey,
		toasts,
		statusPill,
		previewBanner,
		toast,
		run,
		refreshAll,
	};
}
