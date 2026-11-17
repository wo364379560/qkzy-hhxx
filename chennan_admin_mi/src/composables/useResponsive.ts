import { computed, onMounted, onUnmounted, ref } from 'vue'

const MOBILE_BREAKPOINT = 768
const TABLET_BREAKPOINT = 1200

export function useResponsive() {
  const width = ref(typeof window !== 'undefined' ? window.innerWidth : TABLET_BREAKPOINT)

  const updateWidth = () => {
    if (typeof window === 'undefined') {
      return
    }
    width.value = window.innerWidth
  }

  onMounted(() => {
    updateWidth()
    window.addEventListener('resize', updateWidth, { passive: true })
  })

  onUnmounted(() => {
    if (typeof window === 'undefined') {
      return
    }
    window.removeEventListener('resize', updateWidth)
  })

  const isMobile = computed(() => width.value < MOBILE_BREAKPOINT)
  const isTablet = computed(() => width.value >= MOBILE_BREAKPOINT && width.value < TABLET_BREAKPOINT)
  const isDesktop = computed(() => width.value >= TABLET_BREAKPOINT)
  const isCompact = computed(() => width.value < TABLET_BREAKPOINT)

  return {
    width,
    isMobile,
    isTablet,
    isDesktop,
    isCompact,
  }
}
