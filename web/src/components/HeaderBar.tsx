import { useState, useEffect, useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import { motion, AnimatePresence } from 'framer-motion'
import { Menu, X, ChevronDown } from 'lucide-react'
import { t, type Language } from '../i18n/translations'
import { useSystemConfig } from '../hooks/useSystemConfig'

type Page =
  | 'competition'
  | 'traders'
  | 'trader'
  | 'chaos'
  | 'chaos-studio'
  | 'backtest'
  | 'strategy'
  | 'strategy-market'
  | 'time-machine'
  | 'data'
  | 'debate'
  | 'transactions'
  | 'faq'
  | 'login'
  | 'register'

interface HeaderBarProps {
  onLoginClick?: () => void
  isLoggedIn?: boolean
  currentPage?: Page
  language?: Language
  onLanguageChange?: (lang: Language) => void
  user?: { email: string } | null
  onLogout?: () => void
  onPageChange?: (page: Page) => void
  onLoginRequired?: (featureName: string) => void
}

export default function HeaderBar({
  isLoggedIn = false,
  currentPage,
  language = 'zh' as Language,
  onLanguageChange,
  user,
  onLogout,
  onPageChange,
  onLoginRequired,
}: HeaderBarProps) {
  const navigate = useNavigate()
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false)
  const [languageDropdownOpen, setLanguageDropdownOpen] = useState(false)
  const [userDropdownOpen, setUserDropdownOpen] = useState(false)
  const dropdownRef = useRef<HTMLDivElement>(null)
  const userDropdownRef = useRef<HTMLDivElement>(null)
  const { config: systemConfig } = useSystemConfig()
  const registrationEnabled = systemConfig?.registration_enabled !== false

  // Close dropdown when clicking outside
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (
        dropdownRef.current &&
        !dropdownRef.current.contains(event.target as Node)
      ) {
        setLanguageDropdownOpen(false)
      }
      if (
        userDropdownRef.current &&
        !userDropdownRef.current.contains(event.target as Node)
      ) {
        setUserDropdownOpen(false)
      }
    }

    document.addEventListener('mousedown', handleClickOutside)
    return () => {
      document.removeEventListener('mousedown', handleClickOutside)
    }
  }, [])

  return (
    <nav className="fixed top-0 w-full z-50 header-bar">
      <div className="flex items-center justify-between h-16 px-4 sm:px-6 max-w-[1920px] mx-auto">
        {/* Logo - Always go to home page */}
        <div
          onClick={() => {
            window.location.href = '/'
          }}
          className="hover:opacity-80 transition-opacity cursor-pointer"
        >
          <span className="text-2xl font-bold text-nofx-gold">
            AgentK
          </span>
        </div>

        {/* Desktop Menu */}
        <div className="hidden md:flex items-center justify-between flex-1">
          {/* Left Side - Navigation Tabs */}
          <div className="flex items-center gap-2">
            {/* Navigation tabs configuration */}
            {(() => {
              // Define all navigation tabs
              const allNavTabs: { page: Page; path: string; label: string; requiresAuth: boolean }[] = [
                { page: 'traders', path: '/traders', label: t('configNav', language), requiresAuth: true },
                { page: 'chaos', path: '/chaos', label: language === 'zh' ? '看板' : 'Dashboard', requiresAuth: true },
                { page: 'strategy', path: '/strategy', label: t('strategyNav', language), requiresAuth: true },
                { page: 'time-machine', path: '/time-machine', label: language === 'zh' ? '时光机' : 'Time Machine', requiresAuth: true },
                { page: 'transactions', path: '/transactions', label: language === 'zh' ? '交易列表' : 'Transactions', requiresAuth: true },
                { page: 'competition', path: '/competition', label: t('realtimeNav', language), requiresAuth: true },
                { page: 'faq', path: '/faq', label: t('faqNav', language), requiresAuth: false },
              ]

              const HIDDEN_PAGES: Page[] = ['data', 'strategy-market', 'backtest', 'chaos-studio', 'debate', 'trader']

              const navTabs = allNavTabs.filter(tab => {
                if (HIDDEN_PAGES.includes(tab.page)) return false
                return true
              })

              const handleNavClick = (tab: typeof navTabs[0]) => {
                // If requires auth and not logged in, show login prompt
                if (tab.requiresAuth && !isLoggedIn) {
                  onLoginRequired?.(tab.label)
                  return
                }
                // Navigate normally
                if (onPageChange) {
                  onPageChange(tab.page)
                }
                navigate(tab.path)
              }

              return navTabs.map((tab) => (
                <button
                  key={tab.page}
                  onClick={() => handleNavClick(tab)}
                  className={`text-sm font-bold transition-all duration-300 relative focus:outline-2 focus:outline-yellow-500 px-3 py-2 rounded-lg
                    ${currentPage === tab.page ? 'text-nofx-gold' : 'text-nofx-text-muted hover:text-nofx-gold'}`}
                >
                  {currentPage === tab.page && (
                    <span
                      className="absolute inset-0 rounded-lg bg-nofx-gold/15 -z-10"
                    />
                  )}
                  {tab.label}
                </button>
              ))
            })()}
          </div>

          {/* Right Side - User Actions */}
          <div className="flex items-center gap-4">

            {/* User Info and Actions */}
            {isLoggedIn && user ? (
              <div className="flex items-center gap-3">
                {/* User Info with Dropdown */}
                <div className="relative" ref={userDropdownRef}>
                  <button
                    onClick={() => setUserDropdownOpen(!userDropdownOpen)}
                    className="flex items-center gap-2 px-3 py-2 rounded transition-colors bg-nofx-bg-lighter border border-nofx-gold/20 hover:bg-white/5"
                  >
                    <div className="w-6 h-6 rounded-full flex items-center justify-center text-xs font-bold bg-nofx-gold text-black">
                      {user.email[0].toUpperCase()}
                    </div>
                    <span className="text-sm text-nofx-text-muted">
                      {user.email}
                    </span>
                    <ChevronDown className="w-4 h-4 text-nofx-text-muted" />
                  </button>

                  {userDropdownOpen && (
                    <div className="absolute right-0 top-full mt-2 w-48 rounded-lg shadow-lg overflow-hidden z-50 bg-nofx-bg-lighter border border-nofx-gold/20">
                      <div className="px-3 py-2 border-b border-nofx-gold/20">
                        <div className="text-xs text-nofx-text-muted">
                          {t('loggedInAs', language)}
                        </div>
                        <div className="text-sm font-medium text-nofx-text-muted">
                          {user.email}
                        </div>
                      </div>
                      {onLogout && (
                        <button
                          onClick={() => {
                            onLogout()
                            setUserDropdownOpen(false)
                          }}
                          className="w-full px-3 py-2 text-sm font-semibold transition-colors hover:opacity-80 text-center bg-nofx-danger/20 text-nofx-danger"
                        >
                          {t('exitLogin', language)}
                        </button>
                      )}
                    </div>
                  )}
                </div>
              </div>
            ) : (
              /* Show login/register buttons when not logged in and not on login/register pages */
              currentPage !== 'login' &&
              currentPage !== 'register' && (
                <div className="flex items-center gap-3">
                  <a
                    href="/login"
                    className="px-3 py-2 text-sm font-medium transition-colors rounded text-nofx-text-muted hover:text-white"
                  >
                    {t('signIn', language)}
                  </a>
                  {registrationEnabled && (
                    <a
                      href="/register"
                      className="px-4 py-2 rounded font-semibold text-sm transition-colors hover:opacity-90 bg-nofx-gold text-black"
                    >
                      {t('signUp', language)}
                    </a>
                  )}
                </div>
              )
            )}

            {/* Language Toggle - Always at the rightmost */}
            <div className="relative" ref={dropdownRef}>
              <button
                onClick={() => setLanguageDropdownOpen(!languageDropdownOpen)}
                className="flex items-center gap-2 px-3 py-2 rounded transition-colors text-nofx-text-muted hover:bg-white/5"
              >
                <span className="text-lg">
                  {language === 'zh' ? '🇨🇳' : '🇺🇸'}
                </span>
                <ChevronDown className="w-4 h-4" />
              </button>

              {languageDropdownOpen && (
                <div className="absolute right-0 top-full mt-2 w-32 rounded-lg shadow-lg overflow-hidden z-50 bg-nofx-bg-lighter border border-nofx-gold/20">
                  <button
                    onClick={() => {
                      onLanguageChange?.('zh')
                      setLanguageDropdownOpen(false)
                    }}
                    className={`w-full flex items-center gap-2 px-3 py-2 transition-colors text-nofx-text-muted hover:text-white
                      ${language === 'zh' ? 'bg-nofx-gold/10' : 'hover:bg-white/5'}`}
                  >
                    <span className="text-base">🇨🇳</span>
                    <span className="text-sm">中文</span>
                  </button>
                  <button
                    onClick={() => {
                      onLanguageChange?.('en')
                      setLanguageDropdownOpen(false)
                    }}
                    className={`w-full flex items-center gap-2 px-3 py-2 transition-colors text-nofx-text-muted hover:text-white
                      ${language === 'en' ? 'bg-nofx-gold/10' : 'hover:bg-white/5'}`}
                  >
                    <span className="text-base">🇺🇸</span>
                    <span className="text-sm">English</span>
                  </button>
                </div>
              )}
            </div>
          </div>
        </div>

        {/* Mobile Menu Button */}
        <motion.button
          onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
          className="md:hidden text-nofx-text-muted hover:text-white"
          whileTap={{ scale: 0.9 }}
        >
          {mobileMenuOpen ? (
            <X className="w-6 h-6" />
          ) : (
            <Menu className="w-6 h-6" />
          )}
        </motion.button>
      </div>

      {/* Mobile Menu Overlay */}
      <AnimatePresence>
        {mobileMenuOpen && (
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            transition={{ duration: 0.2 }}
            className="fixed inset-0 z-40 md:hidden bg-black/90 backdrop-blur-xl"
            style={{ top: '64px' }} // Below header
          >
            <motion.div
              initial={{ y: -20, opacity: 0 }}
              animate={{ y: 0, opacity: 1 }}
              transition={{ delay: 0.1, duration: 0.3 }}
              className="flex flex-col h-[calc(100vh-64px)] overflow-y-auto px-6 py-8"
            >
              {/* Navigation Links */}
              <div className="flex flex-col gap-6 mb-12">
                {(() => {
                  const allNavTabs: { page: Page; path: string; label: string; requiresAuth: boolean }[] = [
                    { page: 'traders', path: '/traders', label: t('configNav', language), requiresAuth: true },
                    { page: 'chaos', path: '/chaos', label: language === 'zh' ? '看板' : 'Dashboard', requiresAuth: true },
                    { page: 'strategy', path: '/strategy', label: t('strategyNav', language), requiresAuth: true },
                    { page: 'time-machine', path: '/time-machine', label: language === 'zh' ? '时光机' : 'Time Machine', requiresAuth: true },
                    { page: 'transactions', path: '/transactions', label: language === 'zh' ? '交易列表' : 'Transactions', requiresAuth: true },
                    { page: 'competition', path: '/competition', label: t('realtimeNav', language), requiresAuth: true },
                    { page: 'faq', path: '/faq', label: t('faqNav', language), requiresAuth: false },
                  ]

                  const HIDDEN_PAGES: Page[] = ['data', 'strategy-market', 'backtest', 'chaos-studio', 'debate', 'trader']

                  const navTabs = allNavTabs.filter(tab => {
                    if (HIDDEN_PAGES.includes(tab.page)) return false
                    return true
                  })

                  const handleMobileNavClick = (tab: typeof navTabs[0]) => {
                    if (tab.requiresAuth && !isLoggedIn) {
                      onLoginRequired?.(tab.label)
                      setMobileMenuOpen(false)
                      return
                    }
                    if (onPageChange) {
                      onPageChange(tab.page)
                    }
                    navigate(tab.path)
                    setMobileMenuOpen(false)
                  }

                  return navTabs.map((tab, i) => (
                    <motion.button
                      key={tab.page}
                      initial={{ x: -20, opacity: 0 }}
                      animate={{ x: 0, opacity: 1 }}
                      transition={{ delay: 0.1 + i * 0.05 }}
                      onClick={() => handleMobileNavClick(tab)}
                      className={`text-2xl font-black tracking-tight text-left flex items-center gap-3
                        ${currentPage === tab.page ? 'text-nofx-gold' : 'text-zinc-500'}`}
                    >
                      {currentPage === tab.page && (
                        <motion.div
                          layoutId="active-indicator"
                          className="w-1.5 h-1.5 rounded-full bg-nofx-gold"
                        />
                      )}
                      {tab.label}
                      {tab.requiresAuth && !isLoggedIn && (
                        <span className="text-[10px] px-1.5 py-0.5 rounded border border-zinc-800 text-zinc-500 font-normal tracking-wide uppercase align-middle relative -top-1">
                          LOGIN_REQ
                        </span>
                      )}
                    </motion.button>
                  ))
                })()}
              </div>

              {/* Bottom Actions */}
              <div className="mt-auto space-y-8">
                {/* Account / Lang */}
                <div className="flex flex-col gap-4">
                  <div className="grid grid-cols-2 gap-4">
                  {/* Lang Switcher */}
                  <div className="flex bg-zinc-900 rounded-lg p-1 border border-zinc-800">
                    {['zh', 'en'].map((lang) => (
                      <button
                        key={lang}
                        onClick={() => {
                          onLanguageChange?.(lang as Language)
                          setMobileMenuOpen(false)
                        }}
                        className={`flex-1 py-3 text-sm font-bold rounded-md transition-colors ${language === lang
                          ? 'bg-zinc-800 text-white shadow-sm'
                          : 'text-zinc-500'
                          }`}
                      >
                        {lang === 'zh' ? 'CN' : 'EN'}
                      </button>
                    ))}
                  </div>

                  {/* Auth Actions */}
                  {isLoggedIn && user ? (
                    <button
                      onClick={() => {
                        onLogout?.()
                        setMobileMenuOpen(false)
                      }}
                      className="bg-red-500/10 border border-red-500/20 text-red-500 rounded-lg font-bold text-sm hover:bg-red-500/20 transition-colors"
                    >
                      {t('exitLogin', language)}
                    </button>
                  ) : (
                    currentPage !== 'login' && currentPage !== 'register' && (
                      <a
                        href="/login"
                        className="flex items-center justify-center bg-nofx-gold text-black rounded-lg font-bold text-sm hover:bg-yellow-400 transition-colors"
                      >
                        {t('signIn', language)}
                      </a>
                    )
                  )}
                </div>
                </div>
              </div>
            </motion.div>
          </motion.div>
        )}
      </AnimatePresence>
    </nav>
  )
}
