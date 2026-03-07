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
          label: 'Start Here',
          items: [
            { label: 'Introduction', link: '/' },
            { label: 'Getting Started', link: '/guides/getting-started/' },
          ],
        },
        {
          label: 'Core Concepts',
          items: [
            { label: 'Agent', link: '/agent/' },
            { label: 'Stream', link: '/stream/' },
          ],
        },
        {
          label: 'Providers',
          items: [
            { label: 'Overview', link: '/provider/' },
            { label: 'Amazon Bedrock', link: '/provider/bedrock/' },
          ],
        },
        {
          label: 'Examples',
          items: [
            { label: 'All Examples', link: '/examples/' },
          ],
        },
      ],
      customCss: [
        './src/styles/custom.css',
      ],
    }),
  ],
});
