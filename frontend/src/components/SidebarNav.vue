<script setup lang="ts">
import type { Page } from '../composables/useChatlog';

const props = defineProps<{ nav: Array<{ name: Page; hint: string }>; page: Page }>();
const emit = defineEmits<{ (e: 'update:page', value: Page): void }>();

function setPage(p: Page) {
	emit('update:page', p);
}
</script>

<template>
	<div class="brand">
		<div class="brandMark" />
		<div>
			<div class="brandTitle">Chatlog</div>
			<div class="brandSub">Wails 桌面版</div>
		</div>
	</div>
	<div class="nav">
		<button
			v-for="n in props.nav"
			:key="n.name"
			type="button"
			:class="['navItem', props.page === n.name ? 'navItemActive' : '']"
			@click="setPage(n.name)"
		>
			<span>{{ n.name }}</span>
			<span class="navHint">{{ n.hint }}</span>
		</button>
	</div>
</template>
