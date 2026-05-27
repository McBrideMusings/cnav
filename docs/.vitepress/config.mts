import { defineConfig } from 'vitepress'

export default defineConfig({
  title: 'cnav Docs',
  description: 'Jump between Claude Code projects and resume past sessions.',
  cleanUrls: true,
  themeConfig: {
    nav: [
      { text: 'Guide', link: '/guide/getting-started' },
      { text: 'API', link: '/api' },
      { text: 'Reference', link: '/PRD' },
    ],
    sidebar: {
      '/guide/': [
        {
          text: 'Guide',
          items: [
            { text: 'Getting Started', link: '/guide/getting-started' },
          ],
        },
      ],
      '/': [
        {
          text: 'Reference',
          items: [
            { text: 'CLI Reference', link: '/api' },
            { text: 'Product Spec (PRD)', link: '/PRD' },
            { text: 'Roadmap', link: '/roadmap' },
            { text: 'File Map', link: '/file-map' },
            { text: 'Glossary', link: '/CONTEXT' },
          ],
        },
        {
          text: 'ADRs',
          items: [
            { text: '0001 — stderr/stdout split', link: '/adr/0001-stderr-stdout-split' },
            { text: '0002 — Two-phase JSONL scan', link: '/adr/0002-two-phase-jsonl-scan' },
            { text: '0003 — Unified list view', link: '/adr/0003-unified-list-view' },
          ],
        },
      ],
    },
    search: { provider: 'local' },
    socialLinks: [{ icon: 'github', link: 'https://github.com/McBrideMusings/cnav' }],
    editLink: {
      pattern: 'https://github.com/McBrideMusings/cnav/edit/main/docs/:path',
      text: 'Edit this page on GitHub',
    },
  },
  vite: {
    server: { host: '0.0.0.0', port: 5193 },
  },
})
