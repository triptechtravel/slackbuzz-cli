import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

export default defineConfig({
  site: 'https://triptechtravel.github.io',
  base: '/slackbuzz-cli',
  legacy: {
    collections: true,
  },
  integrations: [
    starlight({
      title: 'slackbuzz CLI',
      logo: {
        src: './src/assets/logo.svg',
        alt: 'slackbuzz CLI',
        replacesTitle: false,
      },
      favicon: '/favicon.svg',
      customCss: ['./src/styles/custom.css'],
      head: [
        { tag: 'meta', attrs: { property: 'og:image', content: 'https://triptechtravel.github.io/slackbuzz-cli/og-image.png' } },
        { tag: 'meta', attrs: { property: 'og:image:width', content: '1200' } },
        { tag: 'meta', attrs: { property: 'og:image:height', content: '630' } },
        { tag: 'meta', attrs: { property: 'og:description', content: 'A command-line tool for working with Slack messages, channels, and cross-tool workflows -- designed for developers who live in the terminal.' } },
        { tag: 'meta', attrs: { name: 'twitter:card', content: 'summary_large_image' } },
      ],
      social: {
        github: 'https://github.com/triptechtravel/slackbuzz-cli',
      },
      sidebar: [
        {
          label: 'Getting Started',
          items: [
            { label: 'Installation', slug: 'installation' },
            { label: 'Getting Started', slug: 'getting-started' },
            { label: 'Team Setup', slug: 'team-setup' },
          ],
        },
        {
          label: 'Usage',
          items: [
            { label: 'Commands', slug: 'commands' },
            { label: 'Cross-Tool Integration', slug: 'cross-tool-integration' },
          ],
        },
        {
          label: 'Integrations',
          items: [
            { label: 'CI Usage', slug: 'ci-usage' },
            { label: 'AI Agents', slug: 'ai-agents' },
          ],
        },
        {
          label: 'Reference',
          collapsed: true,
          autogenerate: { directory: 'reference' },
        },
        {
          label: 'Project',
          items: [
            { label: 'Contributing', slug: 'contributing' },
            { label: 'Security', slug: 'security' },
          ],
        },
      ],
    }),
  ],
});
