import type { InjectionKey } from 'vue';
import type { useChatlog } from './useChatlog';

export type ChatlogContext = ReturnType<typeof useChatlog> & {
	confirm: (options: import('./useConfirm').ConfirmOptions) => Promise<boolean>;
};

export const chatlogKey: InjectionKey<ChatlogContext> = Symbol('chatlog');
