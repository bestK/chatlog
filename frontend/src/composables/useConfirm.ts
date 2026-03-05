import { ref } from 'vue';

export type ConfirmOptions = {
	title: string;
	message: string;
	confirmText?: string;
	cancelText?: string;
	danger?: boolean;
};

export type ConfirmState = Required<Pick<ConfirmOptions, 'title' | 'message'>> & {
	confirmText: string;
	cancelText: string;
	danger: boolean;
};

export function useConfirm() {
	const state = ref<ConfirmState | null>(null);
	let resolveFn: ((value: boolean) => void) | null = null;

	function confirm(options: ConfirmOptions) {
		state.value = {
			title: options.title,
			message: options.message,
			confirmText: options.confirmText ?? '确认',
			cancelText: options.cancelText ?? '取消',
			danger: options.danger ?? false,
		};
		return new Promise<boolean>((resolve) => {
			resolveFn = resolve;
		});
	}

	function close(result: boolean) {
		const fn = resolveFn;
		resolveFn = null;
		state.value = null;
		if (fn) fn(result);
	}

	function accept() {
		close(true);
	}

	function cancel() {
		close(false);
	}

	return { state, confirm, accept, cancel };
}
