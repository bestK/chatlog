<script setup lang="ts">
import {
    DialogClose,
    DialogContent,
    DialogOverlay,
    DialogPortal,
    DialogRoot,
} from 'reka-ui';

const props = defineProps<{
    src: string;
}>();

const open = defineModel<boolean>('open', { default: false });
</script>

<template>
    <DialogRoot v-model:open="open">
        <DialogPortal>
            <DialogOverlay
                class="fixed inset-0 z-[100] bg-black/80 backdrop-blur-sm data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0"
            />
            <DialogContent
                class="fixed inset-0 z-[100] flex items-center justify-center p-4 focus:outline-none"
                @pointer-down-outside="open = false"
                @interact-outside="open = false"
            >
                <DialogClose class="absolute inset-0 cursor-pointer" />
                <img
                    v-if="src"
                    :src="src"
                    class="relative max-w-[90vw] max-h-[90vh] rounded-lg object-contain shadow-2xl"
                />
            </DialogContent>
        </DialogPortal>
    </DialogRoot>
</template>
