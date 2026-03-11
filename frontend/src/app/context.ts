import type { InjectionKey } from 'vue'
import type { createAppState } from './state'

export type AppContext = ReturnType<typeof createAppState> & {
  feedback: ReturnType<typeof import('./feedback').createFeedbackService>
}

export const appContextKey: InjectionKey<AppContext> = Symbol('app-context')
