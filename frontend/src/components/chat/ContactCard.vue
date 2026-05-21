<script setup lang="ts">
import type { Contact } from '@/wailsbridge';

defineProps<{ contact: Contact }>();
defineEmits<{ click: [contact: Contact] }>();

function getContactName(c: Contact) {
    return c.remark || c.nickName || c.alias || c.userName || '未知联系人';
}

function getContactAvatar(c: Contact) {
    return c.smallHeadImgUrl || '';
}

function getContactAvatarFallback(c: Contact) {
    const name = getContactName(c).trim();
    return name ? name.slice(0, 1).toUpperCase() : '?';
}
</script>

<template>
    <div
        class="group cursor-pointer rounded-lg border border-border/30 bg-card/20 p-4 transition-colors hover:bg-card/40"
        @click="$emit('click', contact)"
    >
        <div class="flex items-center gap-3">
            <img
                v-if="getContactAvatar(contact)"
                :src="getContactAvatar(contact)"
                class="size-10 rounded-full object-cover"
            />
            <div
                v-else
                class="flex size-10 items-center justify-center rounded-full bg-muted/40 text-sm font-medium text-muted-foreground"
            >
                {{ getContactAvatarFallback(contact) }}
            </div>

            <div class="min-w-0 flex-1 space-y-0.5">
                <div class="truncate text-sm font-medium tracking-tight">
                    {{ getContactName(contact) }}
                </div>
                <div class="truncate font-mono text-xs text-muted-foreground">
                    {{ contact.alias || contact.userName }}
                </div>
            </div>
            <span
                v-if="contact.isInChatRoom"
                class="shrink-0 rounded-full bg-primary/10 px-2 py-0.5 text-xs text-primary"
            >
                群聊
            </span>
        </div>
    </div>
</template>