import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { UserInfo } from '../types'

interface AppState {
  user: UserInfo | null
  token: string | null
  theme: 'dark' | 'light'
  language: 'zh' | 'en'
  setUser: (user: UserInfo | null) => void
  setToken: (token: string | null) => void
  setTheme: (theme: 'dark' | 'light') => void
  setLanguage: (lang: 'zh' | 'en') => void
  logout: () => void
}

export const useAppStore = create<AppState>()(
  persist(
    (set) => ({
      user: null,
      token: null,
      theme: 'dark',
      language: 'zh',
      setUser: (user) => set({ user }),
      setToken: (token) => set({ token }),
      setTheme: (theme) => {
        document.body.setAttribute('theme-mode', theme)
        set({ theme })
      },
      setLanguage: (language) => set({ language }),
      logout: () => set({ user: null, token: null }),
    }),
    {
      name: 'new-api-lite-store',
      partialize: (state) => ({
        token: state.token,
        theme: state.theme,
        language: state.language,
      }),
    }
  )
)
