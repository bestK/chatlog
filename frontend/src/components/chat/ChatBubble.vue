<script setup lang="ts">
import { useMediaCache } from '@/composables/useMediaCache';

export type ChatMessage = {
    time: string;
    senderName: string;
    sender: string;
    content: string;
    type: number;
    contents?: Record<string, any>;
    isSelf: boolean;
};

const props = defineProps<{ msg: ChatMessage }>();
const emit = defineEmits<{ preview: [src: string] }>();

const { getMediaSrc, getMediaError } = useMediaCache();

function msgDisplay(msg: ChatMessage) {
    const c = msg.contents || {};
    switch (msg.type) {
        case 1:
            return { kind: 'text' as const, text: msg.content };
        case 3:
            return { kind: 'image' as const, mediaType: 'image', mediaKeys: [c.md5, c.path, c.thumbpath] };
        case 34:
            return { kind: 'voice' as const, mediaType: 'voice', mediaKeys: [c.voice] };
        case 43:
            return { kind: 'video' as const, mediaType: 'video', mediaKeys: [c.md5, c.rawmd5, c.path] };
        case 47:
            return { kind: 'emoji' as const, url: c.cdnurl || '' };
        case 48:
            return { kind: 'text' as const, text: `[位置] ${c.label || ''}` };
        case 49: {
            const title = c.title || '';
            const url = c.url || '';
            if (title) return { kind: 'text' as const, text: `[${title}]${url ? ' ' + url : ''}` };
            return { kind: 'text' as const, text: msg.content || '[分享]' };
        }
        case 10000:
            return { kind: 'text' as const, text: msg.content || '[系统消息]' };
        default:
            return { kind: 'text' as const, text: msg.content || `[类型:${msg.type}]` };
    }
}

const display = msgDisplay(props.msg);
</script>

<template>
    <div class="space-y-0.5" :class="msg.isSelf ? 'text-right' : ''">
        <div class="text-[10px] text-muted-foreground/60">
            <span>{{ msg.senderName }}</span>
            <span class="ml-1.5">{{ msg.time }}</span>
        </div>
        <div
            class="inline-block max-w-[85%] whitespace-pre-wrap break-words rounded-lg px-2.5 py-1.5 text-left text-xs"
            :class="msg.isSelf ? 'bg-primary/10 text-foreground' : 'bg-muted/40 text-foreground/90'"
        >
            <template v-if="display.kind === 'image'">
                <template v-if="getMediaSrc(display.mediaType!, display.mediaKeys!) === null">
                    <span class="text-muted-foreground/50">[图片加载中…]</span>
                </template>
                <template v-else-if="getMediaSrc(display.mediaType!, display.mediaKeys!)">
                    <img
                        :src="getMediaSrc(display.mediaType!, display.mediaKeys!)!"
                        class="max-w-[200px] max-h-[150px] rounded object-cover cursor-pointer hover:opacity-80 transition-opacity"
                        @click.stop="emit('preview', getMediaSrc(display.mediaType!, display.mediaKeys!) || '')"
                    />
                </template>
                <span v-else class="text-muted-foreground/50 text-[10px]" :title="getMediaError(display.mediaType!, display.mediaKeys!)">[图片] {{ getMediaError(display.mediaType!, display.mediaKeys!) }}</span>
            </template>
            <template v-else-if="display.kind === 'video'">
                <template v-if="getMediaSrc(display.mediaType!, display.mediaKeys!) === null">
                    <span class="text-muted-foreground/50">[视频加载中…]</span>
                </template>
                <template v-else-if="getMediaSrc(display.mediaType!, display.mediaKeys!)">
                    <video
                        :src="getMediaSrc(display.mediaType!, display.mediaKeys!)!"
                        class="max-w-[200px] max-h-[150px] rounded"
                        controls
                        preload="none"
                    />
                </template>
                <span v-else class="text-muted-foreground/50">[视频]</span>
            </template>
            <template v-else-if="display.kind === 'voice'">
                <template v-if="getMediaSrc(display.mediaType!, display.mediaKeys!) === null">
                    <span class="text-muted-foreground/50">[语音加载中…]</span>
                </template>
                <template v-else-if="getMediaSrc(display.mediaType!, display.mediaKeys!)">
                    <audio
                        :src="getMediaSrc(display.mediaType!, display.mediaKeys!)!"
                        controls
                        preload="none"
                        class="h-8 max-w-[180px]"
                    />
                </template>
                <span v-else class="text-muted-foreground/50">[语音]</span>
            </template>
            <template v-else-if="display.kind === 'emoji'">
                <img v-if="display.url" :src="display.url" class="size-16 object-contain" loading="lazy" />
                <span v-else class="text-muted-foreground">[表情]</span>
            </template>
            <template v-else>
                {{ display.text }}
            </template>
        </div>
    </div>
</template>