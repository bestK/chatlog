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
	<Transition name="modal">
		<div class="modalOverlay" @click.self="$emit('cancel')">
			<div :class="['modal', danger ? 'modalDanger' : '']">
				<div class="modalHeader">
					<div class="modalIcon">
						<span v-if="danger">⚠️</span>
						<span v-else>💡</span>
					</div>
					<div class="modalTitle">{{ title }}</div>
				</div>
				<div class="modalBody">
					<div class="modalMsg">{{ message }}</div>
				</div>
				<div class="modalActions">
					<button type="button" class="btn modalBtn modalBtnCancel" @click="$emit('cancel')">
						{{ cancelText }}
					</button>
					<button
						type="button"
						:class="['btn', 'modalBtn', danger ? 'btnDanger' : 'btnBrand']"
						@click="$emit('confirm')"
					>
						{{ confirmText }}
					</button>
				</div>
			</div>
		</div>
	</Transition>
</template>

<style scoped>
.modalOverlay {
	position: fixed;
	inset: 0;
	background: rgba(0, 0, 0, 0.6);
	backdrop-filter: blur(8px);
	display: flex;
	align-items: center;
	justify-content: center;
	padding: 20px;
	z-index: 10000;
}

.modal {
	width: min(440px, 100%);
	background: linear-gradient(145deg, rgba(30, 35, 45, 0.95), rgba(20, 25, 30, 0.98));
	border: 1px solid rgba(255, 255, 255, 0.1);
	border-radius: 24px;
	box-shadow: 0 20px 50px rgba(0, 0, 0, 0.5), inset 0 1px 1px rgba(255, 255, 255, 0.05);
	overflow: hidden;
	display: flex;
	flex-direction: column;
}

.modalDanger {
	border-color: rgba(255, 91, 127, 0.2);
}

.modalHeader {
	padding: 24px 24px 12px;
	display: flex;
	align-items: center;
	gap: 16px;
}

.modalIcon {
	width: 48px;
	height: 48px;
	border-radius: 14px;
	background: rgba(255, 255, 255, 0.05);
	display: flex;
	align-items: center;
	justify-content: center;
	font-size: 24px;
}

.modalDanger .modalIcon {
	background: rgba(255, 91, 127, 0.1);
}

.modalTitle {
	font-size: 18px;
	font-weight: 800;
	color: #ffffff;
	letter-spacing: -0.5px;
}

.modalBody {
	padding: 0 24px 24px;
}

.modalMsg {
	font-size: 14px;
	line-height: 1.6;
	color: rgba(255, 255, 255, 0.6);
	white-space: pre-wrap;
}

.modalActions {
	padding: 16px 24px 24px;
	display: flex;
	gap: 12px;
	justify-content: flex-end;
	background: rgba(0, 0, 0, 0.2);
}

.modalBtn {
	min-width: 100px;
	height: 44px;
	font-weight: 700;
	font-size: 14px;
	border-radius: 12px;
	transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
}

.modalBtnCancel {
	background: rgba(255, 255, 255, 0.05);
	border: 1px solid rgba(255, 255, 255, 0.1);
	color: rgba(255, 255, 255, 0.7);
}

.modalBtnCancel:hover {
	background: rgba(255, 255, 255, 0.08);
	color: #fff;
}

/* Transitions */
.modal-enter-active, .modal-leave-active {
	transition: all 0.3s cubic-bezier(0.34, 1.56, 0.64, 1);
}

.modal-enter-from {
	opacity: 0;
}

.modal-leave-to {
	opacity: 0;
}

.modal-enter-from .modal {
	transform: scale(0.9) translateY(20px);
	opacity: 0;
}

.modal-leave-to .modal {
	transform: scale(0.95) translateY(10px);
	opacity: 0;
}
</style>
