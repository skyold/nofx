import { useState, useEffect, useCallback } from 'react'
import { useAuth } from '../contexts/AuthContext'
import { useLanguage } from '../contexts/LanguageContext'
import {
  Plus,
  Trash2,
  Check,
  Save,
  Zap,
  Play,
  Clock,
  Bot,
  FileText,
  User,
  Power,
  RefreshCw,
} from 'lucide-react'
import type { AnalystProfile, AIModel } from '../types'
import { confirmToast, notify } from '../lib/notify'
import { api } from '../lib/api'
import { DeepVoidBackground } from '../components/DeepVoidBackground'

export function AnalystStudioPage() {
  const { token } = useAuth()
  const { language } = useLanguage()

  const [profiles, setProfiles] = useState<AnalystProfile[]>([])
  const [selectedProfile, setSelectedProfile] = useState<AnalystProfile | null>(null)
  const [editingProfile, setEditingProfile] = useState<Partial<AnalystProfile> | null>(null)
  const [aiModels, setAiModels] = useState<AIModel[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [isSaving, setIsSaving] = useState(false)
  const [hasChanges, setHasChanges] = useState(false)
  const [isTriggering, setIsTriggering] = useState(false)

  // Fetch profiles and models
  const loadData = useCallback(async () => {
    if (!token) return
    try {
      const [profilesData, modelsData] = await Promise.all([
        api.getAnalystProfiles(),
        api.getModelConfigs(),
      ])
      
      setProfiles(profilesData)
      setAiModels(modelsData.filter(m => m.enabled))
      
      // Select first profile if none selected
      if (!selectedProfile && profilesData.length > 0) {
        setSelectedProfile(profilesData[0])
        setEditingProfile(profilesData[0])
      }
    } catch (err) {
      notify.error(language === 'zh' ? '加载数据失败' : 'Failed to load data')
    } finally {
      setIsLoading(false)
    }
  }, [token, language, selectedProfile])

  useEffect(() => {
    loadData()
  }, [loadData])

  // Create new profile
  const handleCreateProfile = async () => {
    const newProfile = {
      name: language === 'zh' ? '新分析师' : 'New Analyst',
      system_prompt: 'You are a professional crypto market analyst.',
      model_id: aiModels[0]?.id || '',
      cron_schedule: '0 * * * *', // Hourly
      is_enabled: false,
    }
    
    try {
      const created = await api.createAnalystProfile(newProfile as any)
      await loadData()
      setSelectedProfile(created)
      setEditingProfile(created)
      setHasChanges(false)
      notify.success(language === 'zh' ? '分析师已创建' : 'Analyst created')
    } catch (err) {
      notify.error(language === 'zh' ? '创建失败' : 'Failed to create')
    }
  }

  // Save profile
  const handleSaveProfile = async () => {
    if (!selectedProfile || !editingProfile) return
    setIsSaving(true)
    try {
      const updated = await api.updateAnalystProfile(selectedProfile.id, editingProfile as any)
      await loadData()
      setSelectedProfile(updated)
      setEditingProfile(updated)
      setHasChanges(false)
      notify.success(language === 'zh' ? '已保存' : 'Saved')
    } catch (err) {
      notify.error(language === 'zh' ? '保存失败' : 'Failed to save')
    } finally {
      setIsSaving(false)
    }
  }

  // Delete profile
  const handleDeleteProfile = async (id: string) => {
    const confirmed = await confirmToast(
      language === 'zh' ? '确定删除此分析师？' : 'Delete this analyst?',
      {
        title: language === 'zh' ? '确认删除' : 'Confirm Delete',
        okText: language === 'zh' ? '删除' : 'Delete',
        cancelText: language === 'zh' ? '取消' : 'Cancel',
      }
    )
    if (!confirmed) return

    try {
      await api.deleteAnalystProfile(id)
      await loadData()
      if (selectedProfile?.id === id) {
        setSelectedProfile(null)
        setEditingProfile(null)
      }
      notify.success(language === 'zh' ? '已删除' : 'Deleted')
    } catch (err) {
      notify.error(language === 'zh' ? '删除失败' : 'Failed to delete')
    }
  }

  // Trigger analysis
  const handleTriggerAnalysis = async () => {
    if (!selectedProfile) return
    setIsTriggering(true)
    try {
      await api.triggerAnalysis(selectedProfile.id)
      notify.success(language === 'zh' ? '分析已触发' : 'Analysis triggered')
    } catch (err) {
      notify.error(language === 'zh' ? '触发失败' : 'Failed to trigger')
    } finally {
      setIsTriggering(false)
    }
  }

  // Helper to update field
  const updateField = (field: keyof AnalystProfile, value: any) => {
    setEditingProfile(prev => ({ ...prev, [field]: value }))
    setHasChanges(true)
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-[70vh]">
        <div className="w-16 h-16 rounded-full border-4 border-blue-500/20 border-t-blue-500 animate-spin" />
      </div>
    )
  }

  return (
    <DeepVoidBackground className="h-[calc(100vh-64px)] flex flex-col bg-nofx-bg relative overflow-hidden">
      {/* Header */}
      <div className="flex-shrink-0 px-4 py-3 border-b border-nofx-gold/20 bg-nofx-bg/60 backdrop-blur-md z-10">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="p-2 rounded-lg bg-gradient-to-br from-blue-600 to-cyan-600">
              <User className="w-5 h-5 text-white" />
            </div>
            <div>
              <h1 className="text-lg font-bold text-nofx-text">
                {language === 'zh' ? '分析师工作室' : 'Analyst Studio'}
              </h1>
              <p className="text-xs text-nofx-text-muted">
                {language === 'zh' ? '管理虚拟分析师团队' : 'Manage virtual analyst team'}
              </p>
            </div>
          </div>
        </div>
      </div>

      <div className="flex-1 flex overflow-hidden">
        {/* Sidebar List */}
        <div className="w-64 flex-shrink-0 border-r border-nofx-gold/20 overflow-y-auto bg-nofx-bg/30 backdrop-blur-sm z-10">
          <div className="p-2">
            <div className="flex items-center justify-between mb-2 px-2">
              <span className="text-xs font-medium text-nofx-text-muted">
                {language === 'zh' ? '分析师列表' : 'Analysts'}
              </span>
              <button
                onClick={handleCreateProfile}
                className="p-1 rounded hover:bg-white/10 transition-colors text-blue-400"
              >
                <Plus className="w-4 h-4" />
              </button>
            </div>
            <div className="space-y-1">
              {profiles.map(profile => (
                <div
                  key={profile.id}
                  onClick={() => {
                    setSelectedProfile(profile)
                    setEditingProfile(profile)
                    setHasChanges(false)
                  }}
                  className={`group px-2 py-2 rounded-lg cursor-pointer transition-all ${
                    selectedProfile?.id === profile.id
                      ? 'ring-1 ring-blue-500/50 bg-blue-500/10 shadow-[0_0_15px_rgba(59,130,246,0.1)]'
                      : 'hover:bg-nofx-bg-lighter/60 hover:ring-1 hover:ring-blue-500/20 bg-transparent'
                  }`}
                >
                  <div className="flex items-center justify-between">
                    <span className="text-sm truncate text-nofx-text">
                      {profile.name}
                    </span>
                    <button
                      onClick={(e) => {
                        e.stopPropagation()
                        handleDeleteProfile(profile.id)
                      }}
                      className="p-1 rounded hover:bg-nofx-danger/20 text-nofx-danger opacity-0 group-hover:opacity-100 transition-opacity"
                    >
                      <Trash2 className="w-3 h-3" />
                    </button>
                  </div>
                  <div className="flex items-center gap-1 mt-1">
                    {profile.is_enabled ? (
                      <span className="px-1.5 py-0.5 text-[10px] rounded bg-nofx-success/15 text-nofx-success flex items-center gap-1">
                        <div className="w-1.5 h-1.5 rounded-full bg-nofx-success animate-pulse" />
                        {language === 'zh' ? '运行中' : 'Active'}
                      </span>
                    ) : (
                      <span className="px-1.5 py-0.5 text-[10px] rounded bg-nofx-text-muted/10 text-nofx-text-muted">
                        {language === 'zh' ? '已暂停' : 'Paused'}
                      </span>
                    )}
                    <span className="px-1.5 py-0.5 text-[10px] rounded bg-blue-500/10 text-blue-400 flex items-center gap-1">
                      <Clock className="w-2.5 h-2.5" />
                      {profile.cron_schedule}
                    </span>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>

        {/* Editor Area */}
        <div className="flex-1 flex flex-col min-w-0 bg-nofx-bg/50">
          {selectedProfile && editingProfile ? (
            <div className="flex-1 overflow-y-auto p-6">
              {/* Toolbar */}
              <div className="flex items-center justify-between mb-6">
                <div className="flex-1">
                  <input
                    type="text"
                    value={editingProfile.name || ''}
                    onChange={(e) => updateField('name', e.target.value)}
                    className="text-2xl font-bold bg-transparent border-none outline-none w-full text-nofx-text placeholder-nofx-text-muted mb-1"
                    placeholder={language === 'zh' ? '分析师名称' : 'Analyst Name'}
                  />
                  <div className="text-xs text-nofx-text-muted font-mono">ID: {selectedProfile.id}</div>
                </div>
                <div className="flex items-center gap-3">
                  <button
                    onClick={handleTriggerAnalysis}
                    disabled={isTriggering}
                    className="flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium bg-blue-500/10 text-blue-400 border border-blue-500/30 hover:bg-blue-500/20 disabled:opacity-50"
                  >
                    {isTriggering ? <RefreshCw className="w-4 h-4 animate-spin" /> : <Play className="w-4 h-4" />}
                    {language === 'zh' ? '立即运行' : 'Run Now'}
                  </button>
                  <button
                    onClick={handleSaveProfile}
                    disabled={!hasChanges || isSaving}
                    className={`flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-colors disabled:opacity-50
                      ${hasChanges ? 'bg-nofx-gold text-black hover:bg-yellow-500' : 'bg-nofx-bg-lighter text-nofx-text-muted cursor-not-allowed'}`}
                  >
                    <Save className="w-4 h-4" />
                    {isSaving ? (language === 'zh' ? '保存中...' : 'Saving...') : (language === 'zh' ? '保存配置' : 'Save Changes')}
                  </button>
                </div>
              </div>

              {/* Form Grid */}
              <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                {/* Left Column: Settings */}
                <div className="space-y-6">
                  {/* Status Card */}
                  <div className="p-4 rounded-xl bg-nofx-bg-lighter border border-nofx-gold/10">
                    <div className="flex items-center justify-between mb-4">
                      <div className="flex items-center gap-2 text-nofx-text font-medium">
                        <Power className="w-4 h-4 text-nofx-gold" />
                        {language === 'zh' ? '运行状态' : 'Status'}
                      </div>
                      <label className="relative inline-flex items-center cursor-pointer">
                        <input
                          type="checkbox"
                          className="sr-only peer"
                          checked={editingProfile.is_enabled || false}
                          onChange={(e) => updateField('is_enabled', e.target.checked)}
                        />
                        <div className="w-11 h-6 bg-gray-700 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-nofx-success"></div>
                      </label>
                    </div>
                    <p className="text-xs text-nofx-text-muted">
                      {editingProfile.is_enabled 
                        ? (language === 'zh' ? '分析师将按照计划自动运行' : 'Analyst will run automatically according to schedule')
                        : (language === 'zh' ? '自动运行已暂停' : 'Automatic execution is paused')}
                    </p>
                  </div>

                  {/* Schedule Card */}
                  <div className="p-4 rounded-xl bg-nofx-bg-lighter border border-nofx-gold/10">
                    <div className="flex items-center gap-2 text-nofx-text font-medium mb-4">
                      <Clock className="w-4 h-4 text-blue-400" />
                      {language === 'zh' ? '运行计划 (Cron)' : 'Schedule (Cron)'}
                    </div>
                    <input
                      type="text"
                      value={editingProfile.cron_schedule || ''}
                      onChange={(e) => updateField('cron_schedule', e.target.value)}
                      className="w-full bg-nofx-bg border border-nofx-gold/20 rounded-lg px-3 py-2 text-sm text-nofx-text focus:border-blue-500 outline-none font-mono"
                      placeholder="*/30 * * * *"
                    />
                    <div className="mt-2 flex gap-2">
                      {['0 * * * *', '*/30 * * * *', '0 8 * * *'].map(cron => (
                        <button
                          key={cron}
                          onClick={() => updateField('cron_schedule', cron)}
                          className="px-2 py-1 text-xs rounded bg-white/5 hover:bg-white/10 text-nofx-text-muted transition-colors"
                        >
                          {cron}
                        </button>
                      ))}
                    </div>
                  </div>

                  {/* Model Card */}
                  <div className="p-4 rounded-xl bg-nofx-bg-lighter border border-nofx-gold/10">
                    <div className="flex items-center gap-2 text-nofx-text font-medium mb-4">
                      <Bot className="w-4 h-4 text-purple-400" />
                      {language === 'zh' ? 'AI 模型' : 'AI Model'}
                    </div>
                    <select
                      value={editingProfile.model_id || ''}
                      onChange={(e) => updateField('model_id', e.target.value)}
                      className="w-full bg-nofx-bg border border-nofx-gold/20 rounded-lg px-3 py-2 text-sm text-nofx-text focus:border-purple-500 outline-none"
                    >
                      {aiModels.map(model => (
                        <option key={model.id} value={model.id}>
                          {model.name} ({model.provider})
                        </option>
                      ))}
                    </select>
                  </div>
                </div>

                {/* Right Column: Prompt */}
                <div className="flex flex-col h-full">
                  <div className="flex-1 p-4 rounded-xl bg-nofx-bg-lighter border border-nofx-gold/10 flex flex-col">
                    <div className="flex items-center justify-between mb-4">
                      <div className="flex items-center gap-2 text-nofx-text font-medium">
                        <FileText className="w-4 h-4 text-nofx-gold" />
                        {language === 'zh' ? '系统提示词 (System Prompt)' : 'System Prompt'}
                      </div>
                    </div>
                    <textarea
                      value={editingProfile.system_prompt || ''}
                      onChange={(e) => updateField('system_prompt', e.target.value)}
                      className="flex-1 w-full bg-nofx-bg border border-nofx-gold/20 rounded-lg px-4 py-3 text-sm text-nofx-text focus:border-nofx-gold outline-none font-mono resize-none leading-relaxed"
                      placeholder={language === 'zh' ? '定义分析师的角色和分析方法...' : 'Define the analyst role and methodology...'}
                    />
                  </div>
                </div>
              </div>
            </div>
          ) : (
            <div className="flex items-center justify-center h-full text-nofx-text-muted">
              <div className="text-center">
                <User className="w-12 h-12 mx-auto mb-2 opacity-20" />
                <p>{language === 'zh' ? '选择或创建一个分析师' : 'Select or create an analyst'}</p>
              </div>
            </div>
          )}
        </div>
      </div>
    </DeepVoidBackground>
  )
}
