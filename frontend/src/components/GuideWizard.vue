<script setup lang="ts">
import { inject, ref, computed } from 'vue'
import { backend } from '../wailsbridge'
import { appContextKey } from '../app/context'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Badge } from '@/components/ui/badge'

const injected = inject(appContextKey)
if (!injected) throw new Error('chatlog not provided')
const app = injected
const { onboardingStep, state, run, feedback } = app

const loading = ref(false)

const open = computed({
  get: () => onboardingStep.value > 0,
  set: (val) => {
    if (!val) onboardingStep.value = 0
  },
})

async function nextStep() {
  if (loading.value) return
  loading.value = true
  
  try {
    if (onboardingStep.value === 1) {
      // Step 1: Get Data Key
      const key = await backend.GetDataKey()
      await app.refreshAll()
      if (key) {
        onboardingStep.value = 2
        feedback.toast('第一步完成', '已成功获取数据库密钥')
      } else {
        throw new Error('未获取到密钥，请确认微信已登录')
      }
    } else if (onboardingStep.value === 2) {
      // Step 2: Decrypt
      await backend.Decrypt()
      await app.refreshAll()
      onboardingStep.value = 3
      feedback.toast('第二步完成', '数据库解密成功')
    } else if (onboardingStep.value === 3) {
      // Step 3: Start HTTP
      await backend.StartHTTP()
      await app.refreshAll()
      onboardingStep.value = 0
      feedback.toast('引导完成', '服务已启动，现在可以开始使用了')
    }
  } catch (e: any) {
    feedback.toast('操作失败', e.message || String(e))
  } finally {
    loading.value = false
  }
}

function skip() {
  onboardingStep.value = 0
}
</script>

<template>
  <Dialog v-model:open="open">
    <DialogContent class="sm:max-w-[500px] border-border/40 bg-card/95 backdrop-blur-xl">
      <DialogHeader>
        <div class="flex items-center gap-2 mb-2">
          <Badge variant="outline" class="bg-primary/10 text-primary border-primary/20 font-bold">首次使用引导</Badge>
          <div class="text-[10px] text-muted-foreground uppercase tracking-widest font-bold">Step {{ onboardingStep }} of 3</div>
        </div>
        <DialogTitle class="text-xl font-bold tracking-tight">
          <template v-if="onboardingStep === 1">获取数据库密钥</template>
          <template v-else-if="onboardingStep === 2">解密数据库</template>
          <template v-else-if="onboardingStep === 3">开启 HTTP 服务</template>
        </DialogTitle>
        <DialogDescription class="text-sm leading-relaxed">
          <template v-if="onboardingStep === 1">
            我们需要从运行中的微信进程中获取数据库解密密钥。请确保电脑端微信已登录。
          </template>
          <template v-else-if="onboardingStep === 2">
            现在我们将解密后的数据存储到工作目录中，以便快速查询和分析。
          </template>
          <template v-else-if="onboardingStep === 3">
            最后一步，启动 HTTP API 服务。这将允许 AI 助手或其他工具访问您的聊天记录。
          </template>
        </DialogDescription>
      </DialogHeader>

      <div class="py-6">
        <div class="flex items-center justify-between gap-4 p-4 rounded-xl border border-border/40 bg-muted/20">
          <div class="flex items-center gap-3">
            <div :class="['flex size-8 items-center justify-center rounded-full text-xs font-bold', onboardingStep >= 1 ? 'bg-primary text-primary-foreground' : 'bg-muted text-muted-foreground']">1</div>
            <div :class="['h-px w-4 bg-border']" />
            <div :class="['flex size-8 items-center justify-center rounded-full text-xs font-bold', onboardingStep >= 2 ? 'bg-primary text-primary-foreground' : 'bg-muted text-muted-foreground']">2</div>
            <div :class="['h-px w-4 bg-border']" />
            <div :class="['flex size-8 items-center justify-center rounded-full text-xs font-bold', onboardingStep >= 3 ? 'bg-primary text-primary-foreground' : 'bg-muted text-muted-foreground']">3</div>
          </div>
          <div class="text-[10px] font-bold uppercase tracking-widest text-muted-foreground/60">
            {{ onboardingStep === 1 ? 'Preparation' : onboardingStep === 2 ? 'Processing' : 'Finalizing' }}
          </div>
        </div>
      </div>

      <div class="flex items-center justify-between gap-3">
        <Button variant="ghost" class="text-xs font-bold uppercase tracking-widest text-muted-foreground px-0 hover:bg-transparent hover:text-foreground" @click="skip">
          跳过引导
        </Button>
        <Button :disabled="loading" class="min-w-[120px] font-bold shadow-lg shadow-primary/20" @click="nextStep">
          <span v-if="loading" class="mr-2 size-3 animate-spin rounded-full border-2 border-primary-foreground border-t-transparent" />
          {{ onboardingStep === 3 ? '完成并启动' : '下一步' }}
        </Button>
      </div>
    </DialogContent>
  </Dialog>
</template>
