import { Dna, Zap, FileText } from 'lucide-react'
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
  prompt_variant: 's1',
  fault_injection_rate: 0,
  data_noise_level: 0,
  stress_test_mode: false,
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
      chaosParameters: { zh: 'Chaos 参数', en: 'Chaos Parameters' },
      chaosPrompt: { zh: 'Chaos Prompt', en: 'Chaos Prompt' },
      riskControl: { zh: '独立风控', en: 'Independent Risk Control' },
      advancedSettings: { zh: '高级设置', en: 'Advanced Settings' },

      // Prompt Variant
      promptVariant: { zh: 'Prompt 变体', en: 'Prompt Variant' },
      promptVariantDesc: { zh: '选择 Chaos 交易的风格变体', en: 'Select trading style variant' },
      default: { zh: 'Default (默认)', en: 'Default' },
      none: { zh: 'None (无/自定义)', en: 'None (Custom)' },
      s1: { zh: 'S1 (主力/基线)', en: 'S1 (SWING_CORE)' },
      t1: { zh: 'T1 (慢趋势)', en: 'T1 (TREND_FOLLOW_SLOW)' },
      d1: { zh: 'D1 (日内波段)', en: 'D1 (INTRADAY_SWING)' },
      r1: { zh: 'R1 (震荡防御)', en: 'R1 (RANGE_DEFENSIVE)' },
      x1: { zh: 'X1 (实验/微结构)', en: 'X1 (SCALP_EXPERIMENT)' },

      // Chaos Prompt
      chaosPromptDesc: { 
        zh: 'Chaos 策略的核心提示词配置 (JSON)', 
        en: 'Core prompt configuration for Chaos strategy (JSON)' 
      },

      // Advanced Settings
      faultInjection: { zh: '故障注入率', en: 'Fault Injection Rate' },
      faultInjectionDesc: { zh: '模拟系统故障的概率 (0-1)', en: 'Probability of simulated faults (0-1)' },
      dataNoise: { zh: '数据噪声水平', en: 'Data Noise Level' },
      dataNoiseDesc: { zh: '注入市场数据的噪声强度 (0-1)', en: 'Noise intensity injected into market data (0-1)' },
      stressTest: { zh: '压力测试模式', en: 'Stress Test Mode' },
      stressTestDesc: { zh: '启用极限市场条件模拟', en: 'Enable extreme market condition simulation' },
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
            {t('promptVariant')}
          </label>
          <p className="text-xs mb-2" style={{ color: '#848E9C' }}>
            {t('promptVariantDesc')}
          </p>
          <select
            value={config.prompt_variant || 's1'}
            onChange={(e) => updateField('prompt_variant', e.target.value)}
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
            disabled={disabled}
            className="w-full h-96 px-3 py-2 rounded resize-none font-mono text-xs"
            style={{
              ...inputStyle,
              border: '1px solid #a855f7',
            }}
          />
        </div>
      </div>

      {/* Advanced Settings */}
      <div>
        <div className="flex items-center gap-2 mb-4">
          <Zap className="w-5 h-5" style={{ color: '#F0B90B' }} />
          <h3 className="font-medium" style={{ color: '#EAECEF' }}>
            {t('advancedSettings')}
          </h3>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div className="p-4 rounded-lg" style={sectionStyle}>
            <label className="block text-sm mb-1" style={{ color: '#EAECEF' }}>
              {t('faultInjection')}
            </label>
            <p className="text-xs mb-2" style={{ color: '#848E9C' }}>
              {t('faultInjectionDesc')}
            </p>
            <div className="flex items-center gap-2">
                <input
                    type="range"
                    value={(config.fault_injection_rate || 0) * 100}
                    onChange={(e) => updateField('fault_injection_rate', parseInt(e.target.value) / 100)}
                    disabled={disabled}
                    min={0}
                    max={100}
                    className="flex-1 accent-purple-500"
                />
                <span className="w-12 text-center font-mono" style={{ color: '#a855f7' }}>
                    {Math.round((config.fault_injection_rate || 0) * 100)}%
                </span>
            </div>
          </div>

          <div className="p-4 rounded-lg" style={sectionStyle}>
            <label className="block text-sm mb-1" style={{ color: '#EAECEF' }}>
              {t('dataNoise')}
            </label>
            <p className="text-xs mb-2" style={{ color: '#848E9C' }}>
              {t('dataNoiseDesc')}
            </p>
            <div className="flex items-center gap-2">
                <input
                    type="range"
                    value={(config.data_noise_level || 0) * 100}
                    onChange={(e) => updateField('data_noise_level', parseInt(e.target.value) / 100)}
                    disabled={disabled}
                    min={0}
                    max={100}
                    className="flex-1 accent-purple-500"
                />
                <span className="w-12 text-center font-mono" style={{ color: '#a855f7' }}>
                    {Math.round((config.data_noise_level || 0) * 100)}%
                </span>
            </div>
          </div>
          
           <div className="p-4 rounded-lg col-span-1 md:col-span-2" style={sectionStyle}>
              <div className="flex items-center justify-between">
                <div>
                  <label className="block text-sm" style={{ color: '#EAECEF' }}>
                    {t('stressTest')}
                  </label>
                  <p className="text-xs" style={{ color: '#848E9C' }}>
                    {t('stressTestDesc')}
                  </p>
                </div>
                <label className="relative inline-flex items-center cursor-pointer">
                  <input
                    type="checkbox"
                    checked={config.stress_test_mode || false}
                    onChange={(e) => updateField('stress_test_mode', e.target.checked)}
                    disabled={disabled}
                    className="sr-only peer"
                  />
                  <div className="w-11 h-6 bg-gray-600 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full rtl:peer-checked:after:-translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-purple-600"></div>
                </label>
              </div>
            </div>
        </div>
      </div>
    </div>
  )
}
