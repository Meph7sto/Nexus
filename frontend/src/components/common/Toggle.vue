<template>
 <button
 type="button"
 @click="toggle"
 class="relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-[rgba(255,86,0,0.22)] focus:ring-offset-2"
 :class="[modelValue ? 'bg-[var(--nx-accent)]' : 'bg-[var(--nx-border-strong)]']"
 role="switch"
 :aria-checked="modelValue"
 >
 <span
 class="pointer-events-none inline-block h-5 w-5 transform rounded-full bg-[var(--nx-surface)] shadow-sm ring-0 transition duration-200 ease-in-out"
 :class="[modelValue ? 'translate-x-5' : 'translate-x-0']"
 />
 </button>
</template>

<script setup lang="ts">
const props = defineProps<{
 modelValue: boolean
}>()

const emit = defineEmits<{
 (e: 'update:modelValue', value: boolean): void
}>()

function toggle() {
 emit('update:modelValue', !props.modelValue)
}
</script>
