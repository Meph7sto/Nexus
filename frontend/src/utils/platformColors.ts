/**
 * Centralized platform color definitions.
 *
 * All components that need platform-specific styling should import from here
 * instead of defining their own color mappings.
 */

export type Platform = 'anthropic' | 'openai' | 'antigravity' | 'gemini' | 'grok'

// ── Badge (bg + text + border, for inline badges with border) ───────
const BADGE: Record<Platform, string> = {
 anthropic: 'bg-orange-500/10 text-orange-600 border-orange-500/30 ',
 openai: 'bg-primary-500/10 text-primary-700 border-primary-500/30 ',
 antigravity: 'bg-purple-500/10 text-purple-600 border-purple-500/30 ',
 gemini: 'bg-blue-500/10 text-blue-600 border-blue-500/30 ',
 grok: 'bg-zinc-800/10 text-zinc-800 border-zinc-800/30 ',
}
const BADGE_DEFAULT = 'bg-slate-500/10 text-slate-600 border-slate-500/30 '

// ── Light badge (softer bg, no border) ──────────────────────────────
const BADGE_LIGHT: Record<Platform, string> = {
 anthropic: 'bg-orange-500/10 text-orange-600 ',
 openai: 'bg-primary-500/10 text-primary-700 ',
 antigravity: 'bg-purple-500/10 text-purple-600 ',
 gemini: 'bg-blue-500/10 text-blue-600 ',
 grok: 'bg-zinc-800/10 text-zinc-800 ',
}

// ── Border ──────────────────────────────────────────────────────────
const BORDER: Record<Platform, string> = {
 anthropic: 'border-orange-500/20 ',
 openai: 'border-primary-500/20 ',
 antigravity: 'border-purple-500/20 ',
 gemini: 'border-blue-500/20 ',
 grok: 'border-zinc-800/20 ',
}
const BORDER_DEFAULT = 'border-gray-200 '

// ── Accent bar (gradient) ───────────────────────────────────────────
const ACCENT_BAR: Record<Platform, string> = {
 anthropic: '  ',
 openai: '  ',
 antigravity: '  ',
 gemini: '  ',
 grok: '  ',
}
const ACCENT_BAR_DEFAULT = '  '

// ── Text (price, icon) ─────────────────────────────────────────────
const TEXT: Record<Platform, string> = {
 anthropic: 'text-orange-600 ',
 openai: 'text-primary-700 ',
 antigravity: 'text-purple-600 ',
 gemini: 'text-blue-600 ',
 grok: 'text-zinc-800 ',
}
const TEXT_DEFAULT = 'text-primary-600 '

// ── Icon (check mark etc.) ──────────────────────────────────────────
const ICON: Record<Platform, string> = {
 anthropic: 'text-orange-500 ',
 openai: 'text-primary-600 ',
 antigravity: 'text-purple-500 ',
 gemini: 'text-blue-500 ',
 grok: 'text-zinc-800 ',
}
const ICON_DEFAULT = 'text-primary-500 '

// ── Button (solid bg) ───────────────────────────────────────────────
const BUTTON: Record<Platform, string> = {
 anthropic: 'bg-orange-500 text-white hover:bg-orange-600 active:bg-orange-700 ',
 openai: 'bg-primary-600 text-white hover:bg-primary-700 active:bg-primary-800 ',
 antigravity: 'bg-purple-500 text-white hover:bg-purple-600 active:bg-purple-700 ',
 gemini: 'bg-blue-500 text-white hover:bg-blue-600 active:bg-blue-700 ',
 grok: 'bg-zinc-800 text-white hover:bg-zinc-900 active:bg-black ',
}
const BUTTON_DEFAULT = 'bg-primary-500 text-white hover:bg-primary-600 '

// ── Discount badge ──────────────────────────────────────────────────
const DISCOUNT: Record<Platform, string> = {
 anthropic: 'bg-orange-100 text-orange-700 ',
 openai: 'bg-primary-100 text-primary-700 ',
 antigravity: 'bg-purple-100 text-purple-700 ',
 gemini: 'bg-blue-100 text-blue-700 ',
 grok: 'bg-zinc-100 text-zinc-800 ',
}
const DISCOUNT_DEFAULT = 'bg-red-100 text-red-700 '

// ── Header gradient (subscription confirm) ─────────────────────────
const GRADIENT: Record<Platform, string> = {
 anthropic: ' ',
 openai: ' ',
 antigravity: ' ',
 gemini: ' ',
 grok: ' ',
}
const GRADIENT_DEFAULT = ' '

// ── Header text (light text on gradient bg) ────────────────────────
const GRADIENT_TEXT: Record<Platform, string> = {
 anthropic: 'text-orange-100',
 openai: 'text-primary-100',
 antigravity: 'text-purple-100',
 gemini: 'text-blue-100',
 grok: 'text-zinc-100',
}
const GRADIENT_TEXT_DEFAULT = 'text-primary-100'

const GRADIENT_SUBTEXT: Record<Platform, string> = {
 anthropic: 'text-orange-200',
 openai: 'text-primary-200',
 antigravity: 'text-purple-200',
 gemini: 'text-blue-200',
 grok: 'text-zinc-300',
}
const GRADIENT_SUBTEXT_DEFAULT = 'text-primary-200'

// ── Public API ──────────────────────────────────────────────────────

function isPlatform(p: string): p is Platform {
 return p === 'anthropic' || p === 'openai' || p === 'antigravity' || p === 'gemini' || p === 'grok'
}

export function platformBadgeClass(p: string): string {
 return isPlatform(p) ? BADGE[p] : BADGE_DEFAULT
}

export function platformBadgeLightClass(p: string): string {
 return isPlatform(p) ? BADGE_LIGHT[p] : BADGE_DEFAULT
}

export function platformBorderClass(p: string): string {
 return isPlatform(p) ? BORDER[p] : BORDER_DEFAULT
}

export function platformAccentBarClass(p: string): string {
 return isPlatform(p) ? ACCENT_BAR[p] : ACCENT_BAR_DEFAULT
}

export function platformTextClass(p: string): string {
 return isPlatform(p) ? TEXT[p] : TEXT_DEFAULT
}

export function platformIconClass(p: string): string {
 return isPlatform(p) ? ICON[p] : ICON_DEFAULT
}

export function platformButtonClass(p: string): string {
 return isPlatform(p) ? BUTTON[p] : BUTTON_DEFAULT
}

export function platformDiscountClass(p: string): string {
 return isPlatform(p) ? DISCOUNT[p] : DISCOUNT_DEFAULT
}

export function platformGradientClass(p: string): string {
 return isPlatform(p) ? GRADIENT[p] : GRADIENT_DEFAULT
}

export function platformGradientTextClass(p: string): string {
 return isPlatform(p) ? GRADIENT_TEXT[p] : GRADIENT_TEXT_DEFAULT
}

export function platformGradientSubtextClass(p: string): string {
 return isPlatform(p) ? GRADIENT_SUBTEXT[p] : GRADIENT_SUBTEXT_DEFAULT
}

export function platformLabel(p: string): string {
 switch (p) {
 case 'anthropic': return 'Anthropic'
 case 'openai': return 'OpenAI'
 case 'antigravity': return 'Antigravity'
 case 'gemini': return 'Gemini'
 case 'grok': return 'Grok'
 default: return p || 'API'
 }
}
