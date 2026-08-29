<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from 'vue'
import { Calculator, Check, Copy, RotateCcw, ShieldCheck, WalletCards } from 'lucide-vue-next'
import { calculatePricing, calculateQuota, calculateSimulation } from './pricing'

type QuotaUnit = 'M' | '$' | '万' | '亿'
type CopyState = 'idle' | 'success' | 'error'

const cost = ref(158)
const quota = ref(100)
const rate = ref(1)
const loss = ref(3)
const targetProfit = ref(20)
const simulationMultiplier = ref(0.3)
const quotaPercent = ref(3)
const quotaAmount = ref(6.5)
const quotaUnit = ref<QuotaUnit>('M')
const copyState = ref<CopyState>('idle')
let copyTimer: number | undefined

const ratePresets = [1, 7.19, 7.2, 7.5]
const percentPresets = [1, 3, 5, 10, 25, 50]
const quotaUnits: QuotaUnit[] = ['M', '$', '万', '亿']

const pricing = computed(() => calculatePricing({
  cost: cost.value,
  quota: quota.value,
  rate: rate.value,
  lossPercent: loss.value,
  profitPercent: targetProfit.value,
}))
const pricingValid = computed(() => Number.isFinite(pricing.value.breakEven) && Number.isFinite(pricing.value.recommended))
const quotaCalculation = computed(() => calculateQuota(quotaPercent.value, quotaAmount.value))
const simulation = computed(() => calculateSimulation(pricing.value, simulationMultiplier.value))
const simulationMin = 0.05
const simulationMax = computed(() => {
  const target = Number.isFinite(pricing.value.recommended) ? pricing.value.recommended * 1.3 : 1.2
  return Math.max(1.2, Math.ceil(target * 100) / 100, simulationMultiplier.value)
})
const breakEvenPosition = computed(() => {
  if (!pricingValid.value) return 0
  return Math.min(100, Math.max(0, (pricing.value.breakEven - simulationMin) / (simulationMax.value - simulationMin) * 100))
})
const inputNotice = computed(() => {
  if (quota.value <= 0) return '账号额度必须大于 0。'
  if (rate.value <= 0) return '结算汇率必须大于 0。'
  if (cost.value < 0 || loss.value < 0 || targetProfit.value < 0) return '成本、损耗和目标利润不能为负数。'
  if (loss.value > 90) return '综合损耗最高按 90% 计算。'
  return ''
})

function format(value: number, digits = 2) {
  if (!Number.isFinite(value)) return '—'
  return new Intl.NumberFormat('zh-CN', {
    minimumFractionDigits: digits,
    maximumFractionDigits: digits,
  }).format(value)
}

function resetPricing() {
  cost.value = 158
  quota.value = 100
  rate.value = 1
  loss.value = 3
  targetProfit.value = 20
  simulationMultiplier.value = 0.3
  copyState.value = 'idle'
}

async function copyResult() {
  if (!pricingValid.value) return
  const text = [
    `保本倍率 ${format(pricing.value.breakEven, 4)}×`,
    `推荐倍率 ${format(pricing.value.recommended, 4)}×`,
    `保本单价 ¥${format(pricing.value.breakEvenPrice, 2)}/$`,
  ].join(' | ')
  try {
    await navigator.clipboard.writeText(text)
    copyState.value = 'success'
  } catch {
    copyState.value = 'error'
  }
  if (copyTimer !== undefined) window.clearTimeout(copyTimer)
  copyTimer = window.setTimeout(() => { copyState.value = 'idle' }, 2400)
}

onBeforeUnmount(() => {
  if (copyTimer !== undefined) window.clearTimeout(copyTimer)
})
</script>

<template>
  <section class="page pricing-page">
    <div class="page-heading">
      <div>
        <h2>保本测算</h2>
        <p>计算账号保本倍率、目标利润售价和额度换算，数据仅在当前浏览器中处理。</p>
      </div>
      <div class="page-actions">
        <span class="local-state"><ShieldCheck :size="14" />本地计算</span>
        <button type="button" class="button ghost" @click="resetPricing"><RotateCcw :size="15" />重置</button>
      </div>
    </div>

    <section class="tool-section" aria-labelledby="pricing-title">
      <header class="tool-heading">
        <div class="tool-heading-title">
          <span class="tool-heading-icon"><Calculator :size="18" /></span>
          <div><h3 id="pricing-title">账号定价</h3><p>输入进货条件后实时计算出售倍率。</p></div>
        </div>
        <div class="copy-area">
          <span v-if="copyState === 'success'" class="action-feedback success" role="status">已复制</span>
          <span v-else-if="copyState === 'error'" class="action-feedback error" role="alert">复制失败</span>
          <button type="button" class="icon-button" title="复制计算结果" aria-label="复制计算结果" :disabled="!pricingValid" @click="copyResult">
            <Check v-if="copyState === 'success'" :size="16" /><Copy v-else :size="16" />
          </button>
        </div>
      </header>

      <div class="calculator-layout">
        <div class="input-panel">
          <div class="field-grid">
            <label class="number-field"><span>账号成本 <small>人民币</small></span><div class="input-affix"><b>¥</b><input v-model.number="cost" type="number" min="0" step="any" /></div></label>
            <label class="number-field"><span>账号额度 <small>美元面值</small></span><div class="input-affix"><b>$</b><input v-model.number="quota" type="number" min="0" step="any" /></div></label>
            <label class="number-field field-wide"><span>结算汇率 <small>人民币 / 美元</small></span><div class="input-affix"><b>¥/$</b><input v-model.number="rate" type="number" min="0" step="any" /></div></label>
            <div class="segmented field-wide" aria-label="常用结算汇率">
              <button v-for="preset in ratePresets" :key="preset" type="button" :class="{ active: rate === preset }" @click="rate = preset">{{ preset === 7.19 ? '7.19 官价' : format(preset, 2) }}</button>
            </div>
            <label class="number-field"><span>综合损耗 <small>最高 90%</small></span><div class="input-affix suffix"><input v-model.number="loss" type="number" min="0" max="90" step="any" /><b>%</b></div></label>
            <label class="number-field"><span>目标利润率 <small>不计复利</small></span><div class="input-affix suffix"><input v-model.number="targetProfit" type="number" min="0" step="any" /><b>%</b></div></label>
          </div>
          <p v-if="inputNotice" class="inline-message error" role="alert">{{ inputNotice }}</p>
        </div>

        <div class="result-panel" :class="{ invalid: !pricingValid }">
          <div class="result-summary">
            <div class="metric primary"><span>保本倍率</span><strong>{{ format(pricing.breakEven, 3) }}<small>×</small></strong><p>低于此倍率出售会亏损</p></div>
            <div class="metric"><span>推荐倍率</span><strong>{{ format(pricing.recommended, 3) }}<small>×</small></strong><p>包含 {{ format(pricing.profitPercent, 0) }}% 目标利润</p></div>
            <div class="metric"><span>保本单价</span><strong><small>¥</small>{{ format(pricing.breakEvenPrice, 2) }}</strong><p>每 1 美元可用额度</p></div>
            <div class="metric"><span>进货单价</span><strong><small>¥</small>{{ format(pricing.unitCost, 2) }}</strong><p>成本除以账号额度</p></div>
          </div>
          <p v-if="!pricingValid" class="result-empty">填写有效的账号额度和结算汇率后显示结果。</p>
        </div>
      </div>

      <div class="simulation">
        <div class="simulation-control">
          <div class="control-heading"><label for="sale-multiplier">售价倍率</label><div class="compact-number"><input id="sale-multiplier" v-model.number="simulationMultiplier" type="number" :min="simulationMin" step="0.005" /><span>×</span></div></div>
          <div class="range-wrap">
            <input v-model.number="simulationMultiplier" type="range" :min="simulationMin" :max="simulationMax" step="0.005" aria-label="调整售价倍率" />
            <span v-if="pricingValid" class="break-even-mark" :style="{ left: `${breakEvenPosition}%` }">保本</span>
          </div>
        </div>
        <div class="simulation-values">
          <div><span>售空营收</span><strong>¥{{ format(simulation.revenue) }}</strong></div>
          <div><span>净利润</span><strong :class="simulation.netProfit >= 0 ? 'positive' : 'negative'">{{ simulation.netProfit >= 0 ? '+' : '-' }}¥{{ format(Math.abs(simulation.netProfit)) }}</strong></div>
          <div><span>投资回报</span><strong :class="simulation.roi >= 0 ? 'positive' : 'negative'">{{ format(simulation.roi, 1) }}%</strong></div>
        </div>
      </div>
    </section>

    <section class="tool-section" aria-labelledby="quota-title">
      <header class="tool-heading">
        <div class="tool-heading-title"><span class="tool-heading-icon neutral"><WalletCards :size="18" /></span><div><h3 id="quota-title">额度反推</h3><p>根据已知比例和额度换算完整账号额度。</p></div></div>
      </header>
      <div class="quota-layout">
        <div class="quota-inputs">
          <label class="number-field"><span>已知占比 <small>1% 至 100%</small></span><div class="input-affix suffix"><input v-model.number="quotaPercent" type="number" min="0.01" max="100" step="any" /><b>%</b></div></label>
          <div class="segmented" aria-label="常用占比"><button v-for="preset in percentPresets" :key="preset" type="button" :class="{ active: quotaPercent === preset }" @click="quotaPercent = preset">{{ preset }}%</button></div>
          <label class="number-field"><span>对应额度 <small>使用右侧所选单位</small></span><div class="input-affix"><input v-model.number="quotaAmount" type="number" min="0" step="any" /></div></label>
          <div class="segmented" aria-label="额度单位"><button v-for="unit in quotaUnits" :key="unit" type="button" :class="{ active: quotaUnit === unit }" @click="quotaUnit = unit">{{ unit === 'M' ? 'M 百万' : unit }}</button></div>
        </div>
        <div class="quota-results">
          <div class="quota-total"><span>100% 总额度</span><strong>{{ format(quotaCalculation.total) }} <small>{{ quotaUnit }}</small></strong><p v-if="quotaCalculation.valid">{{ format(quotaCalculation.percent) }}% = {{ format(quotaCalculation.amount) }} {{ quotaUnit }}</p><p v-else>已知占比必须大于 0。</p></div>
          <dl><div><dt>1% 对应额度</dt><dd>{{ format(quotaCalculation.perOne) }} {{ quotaUnit }}</dd></div><div><dt>剩余额度</dt><dd>{{ format(quotaCalculation.rest) }} {{ quotaUnit }}</dd></div><div><dt>放大倍数</dt><dd>{{ format(quotaCalculation.multiplier) }}×</dd></div></dl>
        </div>
      </div>
    </section>

    <section class="formula-section" aria-labelledby="formula-title">
      <div><h3 id="formula-title">计算口径</h3><p>保本倍率 = 成本 ÷（额度 × 汇率 ×（1 − 综合损耗率））</p><p>推荐倍率 = 保本倍率 ×（1 + 目标利润率）</p></div>
      <div class="formula-warning"><strong>核对汇率方向</strong><span>汇率位于分母。使用“成本 ÷ 额度 × 汇率”会得到错误且偏高的保本线。</span></div>
    </section>
  </section>
</template>

<style scoped>
.pricing-page { padding-bottom: 12px; }
.local-state { display: inline-flex; min-height: 36px; align-items: center; gap: 6px; padding: 0 10px; color: var(--primary-text); background: var(--primary-soft); border: 1px solid color-mix(in srgb, var(--primary) 20%, var(--border)); border-radius: 6px; font-size: 12px; white-space: nowrap; }
.tool-section { min-width: 0; margin-bottom: 16px; overflow: hidden; background: var(--surface); border: 1px solid var(--border); border-radius: 8px; }
.tool-heading { display: flex; min-height: 72px; align-items: center; justify-content: space-between; gap: 16px; padding: 14px 18px; border-bottom: 1px solid var(--border-soft); }
.tool-heading-title { display: flex; min-width: 0; align-items: center; gap: 11px; }
.tool-heading-icon { display: grid; flex: 0 0 36px; width: 36px; height: 36px; color: var(--primary-text); background: var(--primary-soft); border-radius: 6px; place-items: center; }
.tool-heading-icon.neutral { color: var(--text-secondary); background: var(--surface-hover); }
.tool-heading h3, .formula-section h3 { margin: 0; color: var(--text); font-size: 15px; font-weight: 700; }
.tool-heading p { margin: 3px 0 0; color: var(--muted); font-size: 11px; line-height: 1.4; }
.copy-area { display: flex; min-width: 92px; align-items: center; justify-content: flex-end; gap: 8px; }
.action-feedback { font-size: 11px; white-space: nowrap; }
.action-feedback.success { color: #047857; }
.action-feedback.error { color: var(--danger); }
.calculator-layout { display: grid; grid-template-columns: minmax(330px, .82fr) minmax(0, 1.18fr); }
.input-panel { min-width: 0; padding: 20px; border-right: 1px solid var(--border-soft); }
.field-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 14px 12px; }
.field-wide { grid-column: 1 / -1; }
.number-field { display: grid; min-width: 0; gap: 7px; color: var(--text); font-size: 12px; font-weight: 650; }
.number-field > span { display: flex; align-items: center; justify-content: space-between; gap: 8px; }
.number-field small { color: var(--muted); font-size: 10px; font-weight: 500; }
.input-affix { display: flex; height: 38px; min-width: 0; align-items: center; overflow: hidden; background: var(--surface-subtle); border: 1px solid var(--border); border-radius: 6px; transition: border-color 140ms ease, box-shadow 140ms ease; }
.input-affix:focus-within { background: var(--surface); border-color: var(--primary); box-shadow: 0 0 0 3px color-mix(in srgb, var(--primary) 14%, transparent); }
.input-affix b { flex: 0 0 auto; padding-left: 10px; color: var(--muted); font-size: 11px; font-weight: 600; }
.input-affix input { width: 100%; min-width: 0; height: 36px; padding: 0 10px 0 7px; color: var(--text); background: transparent; border: 0; outline: 0; font-family: ui-monospace, SFMono-Regular, Consolas, monospace; font-size: 13px; }
.input-affix.suffix input { padding-right: 6px; padding-left: 10px; }
.input-affix.suffix b { padding-right: 10px; padding-left: 0; }
.segmented { display: flex; min-width: 0; flex-wrap: wrap; gap: 5px; }
.segmented button { min-height: 28px; padding: 0 9px; color: var(--muted); background: var(--surface-subtle); border: 1px solid var(--border); border-radius: 5px; font-size: 10px; }
.segmented button:hover, .segmented button.active { color: var(--primary-text); background: var(--primary-soft); border-color: color-mix(in srgb, var(--primary) 32%, var(--border)); }
.inline-message { margin: 14px 0 0; padding: 9px 10px; border-radius: 6px; font-size: 11px; line-height: 1.5; }
.inline-message.error { color: var(--danger); background: var(--danger-soft); }
.result-panel { position: relative; min-width: 0; padding: 20px; background: var(--surface-subtle); }
.result-panel.invalid .result-summary { opacity: .36; }
.result-summary { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); border-top: 1px solid var(--border); border-left: 1px solid var(--border); transition: opacity 140ms ease; }
.metric { display: flex; min-width: 0; min-height: 118px; justify-content: center; flex-direction: column; padding: 14px 16px; background: var(--surface); border-right: 1px solid var(--border); border-bottom: 1px solid var(--border); }
.metric > span { color: var(--muted); font-size: 11px; }
.metric strong { margin-top: 7px; color: var(--text); font-family: ui-monospace, SFMono-Regular, Consolas, monospace; font-size: 25px; line-height: 1.15; overflow-wrap: anywhere; }
.metric strong small { margin: 0 3px; font-size: 13px; font-weight: 600; }
.metric p { margin: 5px 0 0; color: var(--muted); font-size: 10px; line-height: 1.4; }
.metric.primary { box-shadow: inset 3px 0 0 var(--primary); }
.metric.primary strong { color: var(--primary-text); font-size: 30px; }
.result-empty { position: absolute; right: 20px; bottom: 20px; left: 20px; margin: 0; padding: 10px; color: var(--danger); background: var(--danger-soft); border: 1px solid color-mix(in srgb, var(--danger) 20%, transparent); border-radius: 6px; font-size: 11px; text-align: center; }
.simulation { display: grid; grid-template-columns: minmax(280px, .8fr) minmax(0, 1.2fr); gap: 24px; padding: 18px 20px 20px; border-top: 1px solid var(--border-soft); }
.simulation-control { min-width: 0; }
.control-heading { display: flex; height: 34px; align-items: center; justify-content: space-between; gap: 12px; }
.control-heading label { color: var(--text); font-size: 12px; font-weight: 650; }
.compact-number { display: flex; width: 110px; height: 32px; align-items: center; overflow: hidden; border: 1px solid var(--border); border-radius: 5px; }
.compact-number input { min-width: 0; width: 100%; height: 30px; padding: 0 4px 0 8px; color: var(--text); background: var(--surface-subtle); border: 0; outline: 0; font-family: ui-monospace, monospace; font-size: 11px; }
.compact-number span { padding-right: 8px; color: var(--muted); font-size: 11px; }
.range-wrap { position: relative; height: 42px; padding-top: 17px; }
.range-wrap input { width: 100%; accent-color: var(--primary); }
.break-even-mark { position: absolute; top: 0; color: var(--danger); font-size: 9px; font-weight: 650; transform: translateX(-50%); }
.simulation-values { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); border-left: 1px solid var(--border-soft); }
.simulation-values > div { display: flex; min-width: 0; justify-content: center; flex-direction: column; gap: 5px; padding: 8px 14px; border-right: 1px solid var(--border-soft); }
.simulation-values span { color: var(--muted); font-size: 10px; }
.simulation-values strong { font-family: ui-monospace, monospace; font-size: 14px; overflow-wrap: anywhere; }
.positive { color: #047857; }
.negative { color: var(--danger); }
.quota-layout { display: grid; grid-template-columns: minmax(280px, .68fr) minmax(0, 1.32fr); }
.quota-inputs { display: grid; min-width: 0; align-content: start; gap: 9px; padding: 20px; border-right: 1px solid var(--border-soft); }
.quota-inputs .segmented + .number-field { margin-top: 7px; }
.quota-results { display: grid; min-width: 0; grid-template-columns: minmax(0, 1fr) minmax(0, 1.2fr); align-items: stretch; padding: 20px; background: var(--surface-subtle); }
.quota-total { display: flex; min-width: 0; justify-content: center; flex-direction: column; padding: 12px 18px; background: var(--surface); border: 1px solid var(--border); border-left: 3px solid var(--primary); }
.quota-total > span, .quota-total p { color: var(--muted); font-size: 10px; }
.quota-total strong { margin: 7px 0 3px; color: var(--primary-text); font-family: ui-monospace, monospace; font-size: 26px; overflow-wrap: anywhere; }
.quota-total strong small { font-size: 12px; }
.quota-total p { margin: 0; }
.quota-results dl { display: grid; margin: 0; border-top: 1px solid var(--border); }
.quota-results dl > div { display: grid; min-width: 0; grid-template-columns: minmax(100px, .8fr) minmax(0, 1.2fr); align-items: center; gap: 10px; padding: 10px 14px; background: var(--surface); border-right: 1px solid var(--border); border-bottom: 1px solid var(--border); }
.quota-results dt { color: var(--muted); font-size: 10px; }
.quota-results dd { min-width: 0; margin: 0; color: var(--text); font-family: ui-monospace, monospace; font-size: 12px; text-align: right; overflow-wrap: anywhere; }
.formula-section { display: grid; grid-template-columns: minmax(0, 1fr) minmax(300px, .7fr); gap: 24px; padding: 18px 20px; background: var(--surface); border: 1px solid var(--border); border-radius: 8px; }
.formula-section p { margin: 8px 0 0; color: var(--text-secondary); font-family: ui-monospace, SFMono-Regular, Consolas, monospace; font-size: 11px; line-height: 1.55; overflow-wrap: anywhere; }
.formula-warning { display: flex; justify-content: center; flex-direction: column; gap: 5px; padding-left: 18px; border-left: 3px solid var(--warning); }
.formula-warning strong { color: var(--warning); font-size: 12px; }
.formula-warning span { color: var(--muted); font-size: 11px; line-height: 1.5; }
@media (max-width: 1023px) { .calculator-layout, .quota-layout { grid-template-columns: minmax(0, 1fr); } .input-panel, .quota-inputs { border-right: 0; border-bottom: 1px solid var(--border-soft); } .simulation { grid-template-columns: minmax(0, 1fr); } .formula-section { grid-template-columns: minmax(0, 1fr); } .formula-warning { padding: 0 0 0 14px; } }
@media (max-width: 620px) { .page-actions { justify-content: space-between; } .tool-heading { align-items: flex-start; padding: 13px 14px; } .tool-heading p { max-width: 220px; } .input-panel, .result-panel, .quota-inputs, .quota-results { padding: 14px; } .field-grid, .result-summary, .quota-results { grid-template-columns: minmax(0, 1fr); } .field-wide { grid-column: auto; } .metric { min-height: 96px; } .simulation { gap: 14px; padding: 14px; } .simulation-values { grid-template-columns: minmax(0, 1fr); border-top: 1px solid var(--border-soft); } .simulation-values > div { min-height: 58px; border-bottom: 1px solid var(--border-soft); } .quota-total { min-height: 112px; } .formula-section { padding: 16px; } }
@media (max-width: 360px) { .local-state { display: none; } .tool-heading-title { align-items: flex-start; } .tool-heading-icon { flex-basis: 32px; width: 32px; height: 32px; } .copy-area { min-width: 36px; } .action-feedback { display: none; } }
</style>
