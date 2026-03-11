import { ref } from 'vue'
import { toast as sonnerToast } from 'vue-sonner'

export type ConfirmOptions = {
  title: string
  message: string
  confirmText?: string
  cancelText?: string
  danger?: boolean
}

export type StatusDialogMode = 'loading' | 'success' | 'error'

export type StatusDialogState = {
  mode: StatusDialogMode
  title: string
  message: string
  detail?: string
}

export type FeedbackConfirmState = Required<Pick<ConfirmOptions, 'title' | 'message'>> & {
  confirmText: string
  cancelText: string
  danger: boolean
}

export function createFeedbackService() {
  const confirmState = ref<FeedbackConfirmState | null>(null)
  const statusState = ref<StatusDialogState | null>(null)
  let confirmResolve: ((value: boolean) => void) | null = null

  function toast(title: string, message: string) {
    sonnerToast(title, {
      description: message,
    })
  }

  function confirm(options: ConfirmOptions) {
    confirmState.value = {
      title: options.title,
      message: options.message,
      confirmText: options.confirmText ?? '确认',
      cancelText: options.cancelText ?? '取消',
      danger: options.danger ?? false,
    }
    return new Promise<boolean>((resolve) => {
      confirmResolve = resolve
    })
  }

  function closeConfirm(result: boolean) {
    const resolver = confirmResolve
    confirmResolve = null
    confirmState.value = null
    if (resolver) resolver(result)
  }

  function acceptConfirm() {
    closeConfirm(true)
  }

  function cancelConfirm() {
    closeConfirm(false)
  }

  function openLoading(title: string, message: string, detail = '') {
    statusState.value = { mode: 'loading', title, message, detail }
  }

  function openResult(mode: Exclude<StatusDialogMode, 'loading'>, title: string, message: string, detail = '') {
    statusState.value = { mode, title, message, detail }
  }

  function updateStatus(message: string, detail?: string) {
    if (!statusState.value || statusState.value.mode !== 'loading') return
    statusState.value = {
      ...statusState.value,
      message,
      detail: detail ?? statusState.value.detail,
    }
  }

  function closeStatus() {
    if (statusState.value?.mode === 'loading') return
    statusState.value = null
  }

  const status = {
    state: statusState,
    openLoading,
    openResult,
    update: updateStatus,
    close: closeStatus,
  }

  return {
    toast,
    confirmState,
    confirm,
    acceptConfirm,
    cancelConfirm,
    statusState,
    status,
  }
}
