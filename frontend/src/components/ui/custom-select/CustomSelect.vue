<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue';
import { ChevronDown, Check } from 'lucide-vue-next';

export interface SelectOption {
    value: string;
    label: string;
}

const props = withDefaults(defineProps<{
    modelValue: string;
    options: SelectOption[];
    placeholder?: string;
    direction?: 'up' | 'down';
}>(), {
    placeholder: '请选择',
    direction: 'down',
});

const emit = defineEmits<{
    'update:modelValue': [value: string];
}>();

const open = ref(false);
const containerRef = ref<HTMLElement | null>(null);

const selectedLabel = computed(() =>
    props.options.find(o => o.value === props.modelValue)?.label || props.placeholder
);

function select(val: string) {
    emit('update:modelValue', val);
    open.value = false;
}

function onClickOutside(e: MouseEvent) {
    if (!open.value) return;
    if (containerRef.value && !containerRef.value.contains(e.target as Node)) {
        open.value = false;
    }
}

onMounted(() => document.addEventListener('click', onClickOutside));
onUnmounted(() => document.removeEventListener('click', onClickOutside));
</script>

<template>
    <div ref="containerRef" class="relative">
        <button
            type="button"
            :class="[
                'flex h-8 w-full items-center justify-between gap-1 rounded-md border border-input bg-background/40 px-2 text-xs transition-colors hover:bg-background/60 focus:outline-none focus:ring-1 focus:ring-ring',
                open && 'ring-1 ring-ring'
            ]"
            @click.stop="open = !open"
        >
            <span class="truncate">{{ selectedLabel }}</span>
            <ChevronDown :class="['size-3 shrink-0 text-muted-foreground transition-transform', open && 'rotate-180']" />
        </button>
        <Transition
            enter-from-class="opacity-0 -translate-y-1"
            enter-active-class="transition duration-150"
            leave-active-class="transition duration-100"
            leave-to-class="opacity-0 -translate-y-1"
        >
            <div
                v-if="open"
                :class="[
                    'absolute left-0 right-0 z-30 min-w-max overflow-auto rounded-md border border-border/60 bg-popover/95 p-1 shadow-lg backdrop-blur',
                    direction === 'up' ? 'bottom-full mb-1.5' : 'top-full mt-1.5'
                ]"
            >
                <button
                    v-for="opt in options"
                    :key="opt.value"
                    type="button"
                    :class="[
                        'flex w-full items-center justify-between gap-3 rounded-md px-2.5 py-1.5 text-left text-xs transition-colors',
                        opt.value === modelValue
                            ? 'bg-accent text-foreground'
                            : 'text-foreground/85 hover:bg-accent/60'
                    ]"
                    @click="select(opt.value)"
                >
                    <span>{{ opt.label }}</span>
                    <Check v-if="opt.value === modelValue" class="size-3 text-foreground/70" />
                </button>
            </div>
        </Transition>
    </div>
</template>
