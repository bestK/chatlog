<script lang="ts" setup>
import { computed } from "vue"
import type { ToasterProps } from "vue-sonner"
import { CircleCheckIcon, InfoIcon, Loader2Icon, OctagonXIcon, TriangleAlertIcon, XIcon } from "lucide-vue-next"
import { Toaster as Sonner } from "vue-sonner"
import { reactiveOmit } from "@vueuse/core"
import { cn } from "@/lib/utils"

const props = defineProps<ToasterProps>()
const delegatedProps = reactiveOmit(props, "class", "theme", "toastOptions")

const toastOptions = computed(() => {
  const incoming = props.toastOptions ?? {}
  const incomingClasses = incoming.classes ?? {}

  return {
    ...incoming,
    unstyled: true,
    closeButtonAriaLabel: incoming.closeButtonAriaLabel ?? "关闭通知",
    classes: {
      toast: cn(
        "sonner-toast group pointer-events-auto relative flex w-[--width] items-start gap-4 overflow-hidden",
        "rounded-2xl border border-white/10",
        "bg-[linear-gradient(165deg,oklch(0.19_0_0/0.92),oklch(0.15_0_0/0.95))]",
        "shadow-[0_24px_48px_-12px_rgba(0,0,0,0.5)] backdrop-blur-2xl",
        "before:absolute before:inset-0 before:bg-[radial-gradient(circle_at_0%_0%,white/5,transparent_40%)]",
        "p-4",
        incomingClasses.toast,
      ),
      icon: cn(
        "sonner-icon shrink-0 flex h-9 w-9 items-center justify-center rounded-xl",
        "text-foreground/90 shadow-[inset_0_1px_0_rgba(255,255,255,0.05)]",
        incomingClasses.icon,
      ),
      content: cn(
        "sonner-content min-w-0 flex-1 flex flex-col gap-1 pr-6",
        incomingClasses.content,
      ),
      title: cn(
        "sonner-title text-[14px] font-bold tracking-tight leading-tight text-foreground",
        incomingClasses.title,
      ),
      description: cn(
        "sonner-description text-[12.5px] leading-relaxed text-foreground/60 font-medium",
        incomingClasses.description,
      ),
      closeButton: cn(
        "sonner-close",
        "absolute right-3 top-3 inline-flex h-6 w-6 items-center justify-center rounded-lg",
        "bg-white/5 text-foreground/40 transition-all duration-200",
        "opacity-0 scale-90",
        "group-hover:opacity-100 group-hover:scale-100",
        "hover:bg-white/10 hover:text-foreground",
        incomingClasses.closeButton,
      ),
      actionButton: cn(
        "sonner-action",
        "inline-flex h-8 items-center rounded-lg border border-white/10",
        "bg-white/8 px-3 text-[12px] font-semibold text-foreground/90",
        "hover:bg-white/15",
        incomingClasses.actionButton,
      ),
      cancelButton: cn(
        "sonner-cancel",
        "inline-flex h-8 items-center rounded-lg border border-white/10",
        "bg-transparent px-3 text-[12px] font-semibold text-foreground/60",
        "hover:bg-white/8 hover:text-foreground/90",
        incomingClasses.cancelButton,
      ),
      success: cn("sonner-type-success", incomingClasses.success),
      error: cn("sonner-type-error", incomingClasses.error),
      warning: cn("sonner-type-warning", incomingClasses.warning),
      info: cn("sonner-type-info", incomingClasses.info),
      loading: cn("sonner-type-loading", incomingClasses.loading),
      default: cn("sonner-type-default", incomingClasses.default),
    },
  }
})

const theme = computed(() => props.theme ?? "dark")
</script>

<template>
  <Sonner
    :class="cn('toaster pointer-events-none z-[200]', props.class)"
    :style="{
      '--border-radius': 'var(--radius)',
      // 供 vue-sonner 内部状态使用（richColors 时也不会跑偏成“彩色卡片”）
      '--normal-bg': 'var(--popover)',
      '--normal-text': 'var(--popover-foreground)',
      '--normal-border': 'var(--border)',
      '--success-bg': 'color-mix(in oklab, var(--popover) 92%, oklch(0.72 0.17 145) 8%)',
      '--success-text': 'var(--popover-foreground)',
      '--success-border': 'color-mix(in oklab, var(--border) 82%, oklch(0.72 0.17 145) 18%)',
      '--error-bg': 'color-mix(in oklab, var(--popover) 92%, oklch(0.65 0.18 25) 8%)',
      '--error-text': 'var(--popover-foreground)',
      '--error-border': 'color-mix(in oklab, var(--border) 82%, oklch(0.65 0.18 25) 18%)',
      '--warning-bg': 'color-mix(in oklab, var(--popover) 92%, oklch(0.78 0.16 80) 8%)',
      '--warning-text': 'var(--popover-foreground)',
      '--warning-border': 'color-mix(in oklab, var(--border) 82%, oklch(0.78 0.16 80) 18%)',
      '--info-bg': 'color-mix(in oklab, var(--popover) 92%, oklch(0.68 0.15 250) 8%)',
      '--info-text': 'var(--popover-foreground)',
      '--info-border': 'color-mix(in oklab, var(--border) 82%, oklch(0.68 0.15 250) 18%)',
    }"
    :theme="theme"
    :toast-options="toastOptions"
    v-bind="delegatedProps"
  >
    <template #success-icon>
      <CircleCheckIcon class="size-4" />
    </template>
    <template #info-icon>
      <InfoIcon class="size-4" />
    </template>
    <template #warning-icon>
      <TriangleAlertIcon class="size-4" />
    </template>
    <template #error-icon>
      <OctagonXIcon class="size-4" />
    </template>
    <template #loading-icon>
      <div>
        <Loader2Icon class="size-4 animate-spin" />
      </div>
    </template>
    <template #close-icon>
      <XIcon class="size-4" />
    </template>
  </Sonner>
</template>
