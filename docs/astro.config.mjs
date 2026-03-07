import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

export default defineConfig({
  base: '/go-ai/',
  integrations: [
    starlight({
      title: 'Go AI SDK',
      description: 'A Go SDK for building AI-powered applications',
      social: {
        github: 'https://github.com/iamanishx/go-ai',
      },
      sidebar: [
        {
          label: 'Getting Started',
          autogenerate: { directory: 'guides' },
        },
        {
          label: 'Guides',
          items: [
            { label: 'Agent', link: '/agent/' },
            { label: 'Provider', link: '/provider/' },
            { label: 'Stream', link: '/stream/' },
          ],
        },
        {
          label: 'Providers',
          autogenerate: { directory: 'provider' },
        },
        {
          label: 'Examples',
          autogenerate: { directory: 'examples' },
        },
      ],
      customCss: [
        './src/styles/custom.css',
      ],
    }),
  ],
});
