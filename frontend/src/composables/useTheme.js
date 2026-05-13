import { ref, provide } from 'vue'

export function useTheme() {
  const isDarkTheme = ref(localStorage.getItem('theme') === 'dark')

  const toggleTheme = () => {
    isDarkTheme.value = !isDarkTheme.value
    localStorage.setItem('theme', isDarkTheme.value ? 'dark' : 'light')
  }

  provide('isDarkTheme', isDarkTheme)

  return { isDarkTheme, toggleTheme }
}