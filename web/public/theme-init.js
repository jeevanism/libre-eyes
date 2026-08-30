(() => {
  try {
    const stored = globalThis.localStorage.getItem('libreeyes-theme')
    const preference = stored === 'light' || stored === 'dark' ? stored : 'system'
    const resolved = preference === 'system' && globalThis.matchMedia('(prefers-color-scheme: dark)').matches
      ? 'dark'
      : preference === 'dark' ? 'dark' : 'light'
    globalThis.document.documentElement.dataset.theme = resolved
    globalThis.document.documentElement.style.colorScheme = resolved
    globalThis.document.querySelector('meta[name="theme-color"]')?.setAttribute(
      'content', resolved === 'dark' ? '#12191b' : '#116466',
    )
  } catch {
    globalThis.document.documentElement.dataset.theme = 'light'
  }
})()
