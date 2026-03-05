<script setup lang="ts">
defineProps<{
	title: string;
	message: string;
	confirmText: string;
	cancelText: string;
	danger: boolean;
}>();

defineEmits<{
	(e: 'confirm'): void;
	(e: 'cancel'): void;
}>();
</script>

<template>
	<div class="modalOverlay" @click.self="$emit('cancel')">
		<div class="modal">
			<div class="modalTitle">{{ title }}</div>
			<div class="modalMsg">{{ message }}</div>
			<div class="modalActions">
				<button type="button" class="btn" @click="$emit('cancel')">{{ cancelText }}</button>
				<button type="button" :class="['btn', danger ? 'btnDanger' : 'btnBrand']" @click="$emit('confirm')">
					{{ confirmText }}
				</button>
			</div>
		</div>
	</div>
</template>
