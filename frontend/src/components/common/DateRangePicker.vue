<template>
 <div class="relative" ref="containerRef">
 <button
 type="button"
 @click="toggle"
 :class="['date-picker-trigger', isOpen && 'date-picker-trigger-open']"
 >
 <span class="date-picker-icon">
 <Icon name="calendar" size="sm" />
 </span>
 <span class="date-picker-value">
 {{ displayValue }}
 </span>
 <span class="date-picker-chevron">
 <Icon
 name="chevronDown"
 size="sm"
 :class="['transition-transform duration-200', isOpen && 'rotate-180']"
 />
 </span>
 </button>

 <Transition name="date-picker-dropdown">
 <div
 v-if="isOpen"
 ref="dropdownRef"
 class="date-picker-dropdown"
 :style="{ left: `${dropdownOffsetX}px` }"
 >
 <!-- Quick presets -->
 <div class="date-picker-presets">
 <button
 v-for="preset in presets"
 :key="preset.value"
 @click="selectPreset(preset)"
 :class="['date-picker-preset', isPresetActive(preset) && 'date-picker-preset-active']"
 >
 {{ t(preset.labelKey) }}
 </button>
 </div>

 <div class="date-picker-divider"></div>

 <!-- Custom date range inputs -->
 <div class="date-picker-custom">
 <div class="date-picker-field">
 <label class="date-picker-label">{{ t('dates.startDate') }}</label>
 <input
 type="date"
 v-model="localStartDate"
 :max="localEndDate || tomorrow"
 class="date-picker-input"
 @change="onDateChange"
 />
 </div>
 <div class="date-picker-separator">
 <Icon name="arrowRight" size="sm" class="text-[var(--nx-subtle)]" />
 </div>
 <div class="date-picker-field">
 <label class="date-picker-label">{{ t('dates.endDate') }}</label>
 <input
 type="date"
 v-model="localEndDate"
 :min="localStartDate"
 :max="tomorrow"
 class="date-picker-input"
 @change="onDateChange"
 />
 </div>
 </div>

 <!-- Apply button -->
 <div class="date-picker-actions">
 <button @click="apply" class="date-picker-apply">
 {{ t('dates.apply') }}
 </button>
 </div>
 </div>
 </Transition>
 </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'

interface DatePreset {
 labelKey: string
 value: string
 getRange: () => { start: string; end: string }
}

interface Props {
 startDate: string
 endDate: string
}

interface Emits {
 (e: 'update:startDate', value: string): void
 (e: 'update:endDate', value: string): void
 (e: 'change', range: { startDate: string; endDate: string; preset: string | null }): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const { t, locale } = useI18n()

const isOpen = ref(false)
const containerRef = ref<HTMLElement | null>(null)
const dropdownRef = ref<HTMLElement | null>(null)
const dropdownOffsetX = ref(0)
const localStartDate = ref(props.startDate)
const localEndDate = ref(props.endDate)
const activePreset = ref<string | null>('last24Hours')
const viewportPadding = 16
const dropdownMaxWidth = 420

const today = computed(() => {
 // Use local timezone to avoid UTC timezone issues
 const now = new Date()
 const year = now.getFullYear()
 const month = String(now.getMonth() + 1).padStart(2, '0')
 const day = String(now.getDate()).padStart(2, '0')
 return `${year}-${month}-${day}`
})

// Tomorrow's date - used for max date to handle timezone differences
// When user is in a timezone behind the server, "today" on server might be "tomorrow" locally
const tomorrow = computed(() => {
 const d = new Date()
 d.setDate(d.getDate() + 1)
 return formatDateToString(d)
})

// Helper function to format date to YYYY-MM-DD using local timezone
const formatDateToString = (date: Date): string => {
 const year = date.getFullYear()
 const month = String(date.getMonth() + 1).padStart(2, '0')
 const day = String(date.getDate()).padStart(2, '0')
 return `${year}-${month}-${day}`
}

const presets: DatePreset[] = [
 {
 labelKey: 'dates.today',
 value: 'today',
 getRange: () => {
 const t = today.value
 return { start: t, end: t }
 }
 },
 {
 labelKey: 'dates.yesterday',
 value: 'yesterday',
 getRange: () => {
 const d = new Date()
 d.setDate(d.getDate() - 1)
 const yesterday = formatDateToString(d)
 return { start: yesterday, end: yesterday }
 }
 },
 {
 labelKey: 'dates.last24Hours',
 value: 'last24Hours',
 getRange: () => {
 const end = new Date()
 const start = new Date(end.getTime() - 24 * 60 * 60 * 1000)
 return {
 start: formatDateToString(start),
 end: formatDateToString(end)
 }
 }
 },
 {
 labelKey: 'dates.last7Days',
 value: '7days',
 getRange: () => {
 const end = today.value
 const d = new Date()
 d.setDate(d.getDate() - 6)
 const start = formatDateToString(d)
 return { start, end }
 }
 },
 {
 labelKey: 'dates.last14Days',
 value: '14days',
 getRange: () => {
 const end = today.value
 const d = new Date()
 d.setDate(d.getDate() - 13)
 const start = formatDateToString(d)
 return { start, end }
 }
 },
 {
 labelKey: 'dates.last30Days',
 value: '30days',
 getRange: () => {
 const end = today.value
 const d = new Date()
 d.setDate(d.getDate() - 29)
 const start = formatDateToString(d)
 return { start, end }
 }
 },
 {
 labelKey: 'dates.thisMonth',
 value: 'thisMonth',
 getRange: () => {
 const now = new Date()
 const start = formatDateToString(new Date(now.getFullYear(), now.getMonth(), 1))
 return { start, end: today.value }
 }
 },
 {
 labelKey: 'dates.lastMonth',
 value: 'lastMonth',
 getRange: () => {
 const now = new Date()
 const start = formatDateToString(new Date(now.getFullYear(), now.getMonth() - 1, 1))
 const end = formatDateToString(new Date(now.getFullYear(), now.getMonth(), 0))
 return { start, end }
 }
 }
]

const displayValue = computed(() => {
 if (activePreset.value) {
 const preset = presets.find((p) => p.value === activePreset.value)
 if (preset) return t(preset.labelKey)
 }

 if (localStartDate.value && localEndDate.value) {
 if (localStartDate.value === localEndDate.value) {
 return formatDate(localStartDate.value)
 }
 return `${formatDate(localStartDate.value)} - ${formatDate(localEndDate.value)}`
 }

 return t('dates.selectDateRange')
})

const formatDate = (dateStr: string): string => {
 const date = new Date(dateStr + 'T00:00:00')
 const dateLocale = locale.value === 'zh' ? 'zh-CN' : 'en-US'
 return date.toLocaleDateString(dateLocale, { month: 'short', day: 'numeric' })
}

const isPresetActive = (preset: DatePreset): boolean => {
 return activePreset.value === preset.value
}

const selectPreset = (preset: DatePreset) => {
 const range = preset.getRange()
 localStartDate.value = range.start
 localEndDate.value = range.end
 activePreset.value = preset.value
}

const onDateChange = () => {
 // Check if current dates match any preset
 activePreset.value = null
 for (const preset of presets) {
 const range = preset.getRange()
 if (range.start === localStartDate.value && range.end === localEndDate.value) {
 activePreset.value = preset.value
 break
 }
 }
}

const calculateDropdownOffset = () => {
 const container = containerRef.value
 if (!container) return 0

 const triggerRect = container.getBoundingClientRect()
 const viewportWidth = window.innerWidth || document.documentElement.clientWidth
 const dropdownWidth = Math.min(dropdownMaxWidth, Math.max(0, viewportWidth - viewportPadding * 2))
 let offset = 0
 const maxRight = viewportWidth - viewportPadding

 if (triggerRect.left + dropdownWidth > maxRight) {
 offset = maxRight - (triggerRect.left + dropdownWidth)
 }

 if (triggerRect.left + offset < viewportPadding) {
 offset += viewportPadding - (triggerRect.left + offset)
 }

 return Math.round(offset)
}

const updateDropdownPosition = async () => {
 if (!isOpen.value) return
 await nextTick()

 const dropdown = dropdownRef.value
 if (!dropdown) return

 const rect = dropdown.getBoundingClientRect()
 const viewportWidth = window.innerWidth || document.documentElement.clientWidth
 let offset = 0
 const maxRight = viewportWidth - viewportPadding

 if (rect.right > maxRight) {
 offset = maxRight - rect.right
 }

 if (rect.left + offset < viewportPadding) {
 offset += viewportPadding - (rect.left + offset)
 }

 dropdownOffsetX.value = Math.round(offset)
}

const toggle = () => {
 if (isOpen.value) {
 isOpen.value = false
 return
 }

 dropdownOffsetX.value = calculateDropdownOffset()
 isOpen.value = true
}

const apply = () => {
 emit('update:startDate', localStartDate.value)
 emit('update:endDate', localEndDate.value)
 emit('change', {
 startDate: localStartDate.value,
 endDate: localEndDate.value,
 preset: activePreset.value
 })
 isOpen.value = false
}

const handleClickOutside = (event: MouseEvent) => {
 if (containerRef.value && !containerRef.value.contains(event.target as Node)) {
 isOpen.value = false
 }
}

const handleEscape = (event: KeyboardEvent) => {
 if (event.key === 'Escape' && isOpen.value) {
 isOpen.value = false
 }
}

const handleViewportChange = () => {
 dropdownOffsetX.value = calculateDropdownOffset()
 void updateDropdownPosition()
}

// Sync local state with props
watch(
 () => props.startDate,
 (val) => {
 localStartDate.value = val
 onDateChange()
 }
)

watch(
 () => props.endDate,
 (val) => {
 localEndDate.value = val
 onDateChange()
 }
)

onMounted(() => {
 document.addEventListener('click', handleClickOutside)
 document.addEventListener('keydown', handleEscape)
 window.addEventListener('resize', handleViewportChange)
 window.addEventListener('scroll', handleViewportChange, true)
 // Initialize active preset detection
 onDateChange()
})

onUnmounted(() => {
 document.removeEventListener('click', handleClickOutside)
 document.removeEventListener('keydown', handleEscape)
 window.removeEventListener('resize', handleViewportChange)
 window.removeEventListener('scroll', handleViewportChange, true)
})
</script>

<style scoped>
.date-picker-trigger {
 @apply flex items-center gap-2;
 @apply px-3 py-2 text-sm;
 @apply transition-all duration-200;
 @apply focus:outline-none;
 @apply cursor-pointer;
 border: 1px solid var(--nx-border);
 border-radius: 4px;
 background: var(--nx-surface);
 color: var(--nx-text);
}

.date-picker-trigger:hover {
 background: var(--nx-bg);
}

.date-picker-trigger-open {
 border-color: var(--nx-text);
 box-shadow: 0 0 0 2px rgba(17, 17, 17, 0.06);
}

.date-picker-icon {
 color: var(--nx-subtle);
}

.date-picker-value {
 @apply font-medium;
}

.date-picker-chevron {
 color: var(--nx-subtle);
}

.date-picker-dropdown {
 @apply absolute z-[100] mt-2;
 @apply overflow-hidden;
 width: min(420px, calc(100vw - 32px));
 min-width: min(320px, calc(100vw - 32px));
 box-sizing: border-box;
 background: var(--nx-surface);
 border: 1px solid var(--nx-border);
 border-radius: 8px;
 box-shadow: 0 8px 20px rgba(17, 17, 17, 0.08);
}

.date-picker-presets {
 @apply grid grid-cols-2 gap-1 p-2;
}

.date-picker-preset {
 @apply px-3 py-1.5 text-xs font-medium;
 @apply transition-colors duration-150;
 border-radius: 6px;
 color: var(--nx-muted);
}

.date-picker-preset:hover {
 background: var(--nx-bg);
 color: var(--nx-text);
}

.date-picker-preset-active {
 background: rgba(255, 86, 0, 0.1);
 color: var(--nx-accent);
}

.date-picker-divider {
 border-top: 1px solid var(--nx-border);
}

.date-picker-custom {
 @apply grid items-end gap-2 p-3;
 grid-template-columns: minmax(0, 1fr) auto minmax(0, 1fr);
}

.date-picker-field {
 @apply flex-1;
}

.date-picker-label {
 @apply mb-1 block text-xs font-medium;
 color: var(--nx-muted);
}

.date-picker-input {
 @apply w-full px-2 py-1.5 text-sm;
 @apply focus:outline-none;
 background: var(--nx-bg);
 border: 1px solid var(--nx-border);
 border-radius: 4px;
 color: var(--nx-text);
}

.date-picker-input:focus {
 border-color: var(--nx-text);
 box-shadow: 0 0 0 2px rgba(17, 17, 17, 0.06);
}

.date-picker-input::-webkit-calendar-picker-indicator {
 @apply cursor-pointer opacity-60 hover:opacity-100;
 filter: invert(0.5);
}

.date-picker-separator {
 @apply flex items-center justify-center pb-1;
}

.date-picker-actions {
 @apply flex justify-end p-2 pt-0;
}

.date-picker-apply {
 @apply px-4 py-1.5 text-sm font-medium text-white;
 @apply transition-colors duration-150;
 border-radius: 4px;
 background: var(--nx-accent);
}

.date-picker-apply:hover {
 background: var(--nx-accent-hover);
}

/* Dropdown animation */
.date-picker-dropdown-enter-active,
.date-picker-dropdown-leave-active {
 transition: all 0.2s ease;
}

.date-picker-dropdown-enter-from,
.date-picker-dropdown-leave-to {
 opacity: 0;
 transform: translateY(-8px);
}
</style>
