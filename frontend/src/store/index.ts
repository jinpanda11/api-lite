import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { UserInfo } from '../types'

interface AppState {
  user: UserInfo | null
  loggedIn: boolean
  theme: 'dark' | 'light'
  language: 'zh' | 'en'
  setUser: (user: UserInfo | null) => void
  setLoggedIn: (v: boolean) => void
  setTheme: (theme: 'dark' | 'light') => void
  setLanguage: (lang: 'zh' | 'en') => void
  logout: () => void
}

export const useAppStore = create<AppState>()(
  persist(
    (set) => ({
      user: null,
      loggedIn: false,
      theme: 'dark',
      language: 'zh',
      setUser: (user) => set({ user }),
      setLoggedIn: (loggedIn) => set({ loggedIn }),
      setTheme: (theme) => {
        document.body.setAttribute('theme-mode', theme)
        set({ theme })
      },
      setLanguage: (language) => set({ language }),
      logout: () => set({ user: null, loggedIn: false }),
    }),
    {
      name: 'new-api-lite-store',
      partialize: (state) => ({
        loggedIn: state.loggedIn,
        theme: state.theme,
        language: state.language,
      }),
    }
  )
)
