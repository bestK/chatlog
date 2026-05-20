<script setup lang="ts">
import { computed, ref, onMounted, onUnmounted } from 'vue'
import { ChevronLeft, ChevronRight, ChevronDown, Check } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'

interface Props {
  total: number
  limit: number
  offset: number
  loading?: boolean
  limitOptions?: number[]
}

const props = withDefaults(defineProps<Props>(), {
  loading: false,
  limitOptions: () => [20, 50, 100, 200],
})

const emit = defineEmits<{
  'update:offset': [value: number]
  'update:limit': [value: number]
}>()

const dropdownOpen = ref(false)
const dropdownRef = ref<HTMLElement | null>(null)

const currentPage = computed(() =>
  props.total <= 0 ? 0 : Math.floor(props.offset / props.limit) + 1
)

const totalPages = computed(() =>
  props.total <= 0 ? 0 : Math.ceil(props.total / props.limit)
)

const rangeText = computed(() => {
  if (props.total <= 0) return '0 条'
  const start = Math.min(props.offset + 1, props.total)
  const end = Math.min(props.offset + props.limit, props.total)
  return `${start}-${end} / ${props.total}`
})

const hasPrev = computed(() => props.offset > 0)
const hasNext = computed(() => props.offset + props.limit < props.total)

function prev() {
  if (!hasPrev.value) return
  emit('update:offset', Math.max(0, props.offset - props.limit))
}

function next() {
  if (!hasNext.value) return
  emit('update:offset', props.offset + props.limit)
}

function selectLimit(val: number) {
  dropdownOpen.value = false
  if (val > 0 && val !== props.limit) emit('update:limit', val)
}

function onClickOutside(e: MouseEvent) {
  if (!dropdownOpen.value) return
  if (dropdownRef.value && !dropdownRef.value.contains(e.target as Node)) {
    dropdownOpen.value = false
  }
}

onMounted(() => document.addEventListener('click', onClickOutside))
onUnmounted(() => document.removeEventListener('click', onClickOutside))
</script>

<template>
  <div class="flex items-center gap-2 text-xs text-muted-foreground">
    <span>{{ rangeText }}</span>
    <div ref="dropdownRef" class="relative">
      <button
        type="button"
        :class="[
          'flex h-7 items-center gap-1 rounded-md border border-input bg-background/40 px-2 text-xs transition-colors hover:bg-background/60 focus:outline-none focus:ring-1 focus:ring-ring',
          dropdownOpen && 'ring-1 ring-ring'
        ]"
        @click.stop="dropdownOpen = !dropdownOpen"
      >
        <span>{{ limit }} / 页</span>
        <ChevronDown :class="['size-3 text-muted-foreground transition-transform', dropdownOpen && 'rotate-180']" />
      </button>
      <Transition
        enter-from-class="opacity-0 -translate-y-1"
        enter-active-class="transition duration-150"
        leave-active-class="transition duration-100"
        leave-to-class="opacity-0 -translate-y-1"
      >
        <div
          v-if="dropdownOpen"
          class="absolute bottom-full left-0 z-30 mb-1.5 min-w-max overflow-auto rounded-md border border-border/60 bg-popover/95 p-1 shadow-lg backdrop-blur"
        >
          <button
            v-for="opt in limitOptions"
            :key="opt"
            type="button"
            :class="[
              'flex w-full items-center justify-between gap-3 rounded-md px-2.5 py-1.5 text-left text-xs transition-colors',
              opt === limit
                ? 'bg-accent text-foreground'
                : 'text-foreground/85 hover:bg-accent/60'
            ]"
            @click="selectLimit(opt)"
          >
            <span>{{ opt }} / 页</span>
            <Check v-if="opt === limit" class="size-3 text-foreground/70" />
          </button>
        </div>
      </Transition>
    </div>
    <div class="flex items-center gap-0.5">
      <Button
        variant="ghost"
        size="icon-sm"
        :disabled="loading || !hasPrev"
        @click="prev"
      >
        <ChevronLeft class="size-4" />
      </Button>
      <span class="min-w-[3ch] text-center tabular-nums">{{ currentPage }} / {{ totalPages }}</span>
      <Button
        variant="ghost"
        size="icon-sm"
        :disabled="loading || !hasNext"
        @click="next"
      >
        <ChevronRight class="size-4" />
      </Button>
    </div>
  </div>
</template>