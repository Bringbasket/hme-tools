import { describe, expect, it } from 'vitest'

// 最小冒烟测试：确认 Vitest 配置可用。
describe('smoke', () => {
  it('works', () => {
    expect(1 + 1).toBe(2)
  })
})
