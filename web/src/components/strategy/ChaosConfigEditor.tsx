import { Dna, FileText } from 'lucide-react'
import type { ChaosStrategyConfig } from '../../types'

interface ChaosConfigEditorProps {
  config: ChaosStrategyConfig
  onChange: (config: ChaosStrategyConfig) => void
  disabled?: boolean
  language: string
}

// Default Chaos config
export const defaultChaosConfig: ChaosStrategyConfig = {
  chaos_prompt: JSON.stringify({
    prompt_meta: {
      type: "chaos",
      prompt_name: "New Chaos Strategy",
      author: "User",
      version: "1.0"
    },
    system_prompt_template: "You are a Chaos Trader...",
  }, null, 2),
  risk_control: {
    max_positions: 3,
    btc_eth_max_leverage: 5,
    altcoin_max_leverage: 5,
    btc_eth_max_position_value_ratio: 5.0,
    altcoin_max_position_value_ratio: 1.0,
    max_margin_usage: 0.9,
    min_position_size: 12,
    min_risk_reward_ratio: 1.5,
    min_confidence: 60,
  },
  system_prompt_variant: 's1',
  user_prompt_version: 'v4',

}

export function ChaosConfigEditor({
  config,
  onChange,
  disabled,
  language,
}: ChaosConfigEditorProps) {
  const t = (key: string) => {
    const translations: Record<string, Record<string, string>> = {
      // Section titles
      chaosParameters: { zh: 'Prompt 变体', en: 'Prompt Variants' },
      chaosPrompt: { zh: 'Chaos Prompt', en: 'Chaos Prompt' },
      riskControl: { zh: '独立风控', en: 'Independent Risk Control' },
      advancedSettings: { zh: '高级设置', en: 'Advanced Settings' },

      // Prompt Variant
      systemPromptVariant: { zh: 'System Prompt 变体', en: 'System Prompt Variant' },
      systemPromptVariantDesc: { zh: '选择 Chaos 交易的风格变体 (Primary/Entry)', en: 'Select trading style variant (Primary/Entry)' },
      default: { zh: 'Default (默认)', en: 'Default' },
      none: { zh: 'None (无/自定义)', en: 'None (Custom)' },
      s1: { zh: 'S1 (主力波段 | 1h/15m)', en: 'S1 (Swing Core | 1h/15m)' },
      t1: { zh: 'T1 (慢速趋势 | 4h/1h)', en: 'T1 (Trend Slow | 4h/1h)' },
      d1: { zh: 'D1 (日内波段 | 15m/5m)', en: 'D1 (Intraday | 15m/5m)' },
      r1: { zh: 'R1 (震荡防御 | 30m/5m)', en: 'R1 (Range Defensive | 30m/5m)' },
      x1: { zh: 'X1 (高频实验 | 5m/1m)', en: 'X1 (Scalp Experiment | 5m/1m)' },

      // Chaos Prompt
      chaosPromptDesc: { 
        zh: 'Chaos 策略的核心提示词配置 (JSON)', 
        en: 'Core prompt configuration for Chaos strategy (JSON)' 
      },
      
      // User Prompt Version
      userPromptVariant: { zh: 'User Prompt 变体', en: 'User Prompt Variant' },
      userPromptVariantDesc: { zh: '选择用户提示词的版本', en: 'Select user prompt version' },
      v1: { zh: 'v1 (旧版)', en: 'v1 (Legacy)' },
      v2: { zh: 'v2 (JSON格式)', en: 'v2 (JSON)' },
      v4: { zh: 'v4 (结构化事实)', en: 'v4 (Structural Facts)' },
    }
    return translations[key]?.[language] || key
  }

  const updateField = <K extends keyof ChaosStrategyConfig>(
    key: K,
    value: ChaosStrategyConfig[K]
  ) => {
    if (!disabled) {
      onChange({ ...config, [key]: value })
    }
  }

  const inputStyle = {
    background: '#1E2329',
    border: '1px solid #2B3139',
    color: '#EAECEF',
  }

  const sectionStyle = {
    background: '#0B0E11',
    border: '1px solid #2B3139',
  }

  return (
    <div className="space-y-6">
      {/* Chaos Parameters */}
      <div>
        <div className="flex items-center gap-2 mb-4">
          <Dna className="w-5 h-5" style={{ color: '#a855f7' }} />
          <h3 className="font-medium" style={{ color: '#EAECEF' }}>
            {t('chaosParameters')}
          </h3>
        </div>

        <div className="p-4 rounded-lg" style={sectionStyle}>
          <label className="block text-sm mb-1" style={{ color: '#EAECEF' }}>
            {t('systemPromptVariant')}
          </label>
          <p className="text-xs mb-2" style={{ color: '#848E9C' }}>
            {t('systemPromptVariantDesc')}
          </p>
          <select
            value={config.system_prompt_variant || 's1'}
            onChange={(e) => updateField('system_prompt_variant', e.target.value)}
            disabled={disabled}
            className="w-full px-3 py-2 rounded"
            style={inputStyle}
          >
            <option value="default">{t('default')}</option>
            <option value="none">{t('none')}</option>
            <option value="s1">{t('s1')}</option>
            <option value="t1">{t('t1')}</option>
            <option value="d1">{t('d1')}</option>
            <option value="r1">{t('r1')}</option>
            <option value="x1">{t('x1')}</option>
          </select>
        </div>
        <div className="p-4 rounded-lg mt-4" style={sectionStyle}>
          <label className="block text-sm mb-1" style={{ color: '#EAECEF' }}>
            {t('userPromptVariant')}
          </label>
          <p className="text-xs mb-2" style={{ color: '#848E9C' }}>
            {t('userPromptVariantDesc')}
          </p>
          <select
            value={config.user_prompt_version || 'v4'}
            onChange={(e) => updateField('user_prompt_version', e.target.value)}
            disabled={disabled}
            className="w-full px-3 py-2 rounded"
            style={inputStyle}
          >
            <option value="v1">{t('v1')}</option>
            <option value="v2">{t('v2')}</option>
            <option value="v4">{t('v4')}</option>
          </select>
        </div>
      </div>

      {/* Chaos Prompt */}
      <div>
        <div className="flex items-center gap-2 mb-4">
          <FileText className="w-5 h-5" style={{ color: '#a855f7' }} />
          <h3 className="font-medium" style={{ color: '#EAECEF' }}>
            {t('chaosPrompt')}
          </h3>
        </div>

        <div className="p-4 rounded-lg" style={sectionStyle}>
          <p className="text-xs mb-2" style={{ color: '#848E9C' }}>
            {t('chaosPromptDesc')}
          </p>
          <textarea
            value={config.chaos_prompt || ''}
            onChange={(e) => updateField('chaos_prompt', e.target.value)}
            onKeyDown={(e) => {
               if ((e.ctrlKey || e.metaKey) && (e.key === 'a' || e.key === 'A')) {
                 e.preventDefault()
                 e.currentTarget.select()
               }
             }}
            disabled={disabled}
            className="w-full h-96 px-3 py-2 rounded resize-none font-mono text-xs"
            style={{
              ...inputStyle,
              border: '1px solid #a855f7',
            }}
          />
        </div>
      </div>


    </div>
  )
}
