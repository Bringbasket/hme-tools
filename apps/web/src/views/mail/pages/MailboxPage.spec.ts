import { createSSRApp } from 'vue'
import { renderToString } from 'vue/server-renderer'
import { afterEach, describe, expect, it, vi } from 'vitest'

vi.mock('../api', () => ({ mailAPI: {} }))

describe('收件箱分页栏', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('将每页条数放在列表底部而不是搜索工具栏', async () => {
    vi.stubGlobal('window', { location: { search: '' } })
    const { default: MailboxPage } = await import('./MailboxPage.vue')
    const html = await renderToString(createSSRApp(MailboxPage))
    const toolbarStart = html.indexOf('class="toolbar mailbox-toolbar"')
    const dataFrameStart = html.indexOf('class="data-frame"', toolbarStart)
    const paginationStart = html.indexOf('class="pagination-bar"', dataFrameStart)

    expect(toolbarStart).toBeGreaterThan(-1)
    expect(dataFrameStart).toBeGreaterThan(toolbarStart)
    expect(paginationStart).toBeGreaterThan(dataFrameStart)
    expect(html.slice(toolbarStart, dataFrameStart)).not.toContain('page-size')
    expect(html.slice(paginationStart)).toContain('class="page-size"')
    expect(html.slice(paginationStart)).toContain('每页')
  })
})
