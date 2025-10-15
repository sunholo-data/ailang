// @ts-check
// `@type` JSDoc annotations allow editor autocompletion and type checking
// (when paired with `@ts-check`).
// There are various equivalent ways to declare your Docusaurus config.
// See: https://docusaurus.io/docs/api/docusaurus-config

import {themes as prismThemes} from 'prism-react-renderer';

/** @type {import('@docusaurus/types').Config} */
const config = {
  title: 'AILANG Documentation',
  tagline: 'AI-first programming language for AI-assisted development',
  favicon: 'img/favicon.ico',

  // Set the production url of your site here
  url: 'https://sunholo-data.github.io',
  // Set the /<baseUrl>/ pathname under which your site is served
  // For GitHub pages deployment, it is often '/<projectName>/'
  baseUrl: '/ailang/',

  // GitHub pages deployment config.
  // If you aren't using GitHub pages, you don't need these.
  organizationName: 'sunholo-data', // Usually your GitHub org/user name.
  projectName: 'ailang', // Usually your repo name.
  trailingSlash: false,

  onBrokenLinks: 'throw',
  onBrokenMarkdownLinks: 'warn',
  onBrokenAnchors: 'warn',

  // Static assets are not checked for broken links - they are copied as-is
  staticDirectories: ['static'],

  // Load Go's WebAssembly support for AILANG REPL
  scripts: [
    {
      src: '/ailang/wasm/wasm_exec.js',
      async: false,
    },
    {
      src: '/ailang/js/ailang-repl.js',
      async: false,
    },
  ],

  // Even if you don't use internationalization, you can use this field to set
  // useful metadata like html lang. For example, if your site is Chinese, you
  // may want to replace "en" with "zh-Hans".
  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
  },

  presets: [
    [
      'classic',
      /** @type {import('@docusaurus/preset-classic').Options} */
      ({
        docs: {
          sidebarPath: './sidebars.js',
          routeBasePath: '/docs', // Docs at /docs instead of root
          // Please change this to your repo.
          // Remove this to remove the "edit this page" links.
          editUrl:
            'https://github.com/sunholo-data/ailang/tree/main/docs/',
        },
        blog: {
          showReadingTime: true,
          routeBasePath: 'blog', // Keep blog at /blog
          feedOptions: {
            type: ['rss', 'atom'],
            xslt: true,
          },
          // Please change this to your repo.
          // Remove this to remove the "edit this page" links.
          editUrl:
            'https://github.com/sunholo-data/ailang/tree/main/docs/',
          // Useful options to enforce blogging best practices
          onInlineTags: 'warn',
          onInlineAuthors: 'warn',
          onUntruncatedBlogPosts: 'warn',
        },
        theme: {
          customCss: './src/css/custom.css',
        },
      }),
    ],
  ],

  themeConfig:
    /** @type {import('@docusaurus/preset-classic').ThemeConfig} */
    ({
      // Replace with your project's social card
      image: 'img/ailang-social-card.jpg',
      navbar: {
        title: 'AILANG',
        logo: {
          alt: 'AILANG Logo',
          src: 'img/logo.png',
        },
        items: [
          {
            type: 'docSidebar',
            sidebarId: 'tutorialSidebar',
            position: 'left',
            label: 'Documentation',
          },
          {
            to: '/docs/playground',
            label: '🎮 Playground',
            position: 'left',
          },
          {
            href: 'https://github.com/sunholo-data/ailang/tree/main/examples',
            label: 'Examples',
            position: 'left',
          },
          {
            href: 'https://sunholo-data.github.io/ailang/llms.txt',
            label: 'llms.txt',
            position: 'right',
          },
          {
            href: 'https://github.com/sunholo-data/ailang',
            label: 'GitHub',
            position: 'right',
          },
        ],
      },
      footer: {
        style: 'dark',
        links: [
          {
            title: 'Docs',
            items: [
              {
                label: 'Getting Started',
                to: '/docs/guides/getting-started',
              },
              {
                label: 'Language Reference',
                to: '/docs/reference/language-syntax',
              },
              {
                label: 'AI Prompts',
                to: '/docs/prompts',
              },
            ],
          },
          {
            title: 'Community',
            items: [
              {
                label: 'GitHub',
                href: 'https://github.com/sunholo-data/ailang',
              },
              {
                label: 'Issues',
                href: 'https://github.com/sunholo-data/ailang/issues',
              },
            ],
          },
          {
            title: 'More',
            items: [
              {
                label: 'Changelog',
                href: 'https://github.com/sunholo-data/ailang/blob/main/CHANGELOG.md',
              },
              {
                label: 'llms.txt',
                href: 'https://sunholo-data.github.io/ailang/llms.txt',
              },
            ],
          },
        ],
        copyright: `Copyright © ${new Date().getFullYear()} Sunholo. Built with Docusaurus.`,
      },
      prism: {
        theme: prismThemes.github,
        darkTheme: prismThemes.dracula,
        additionalLanguages: ['bash', 'json', 'javascript', 'typescript', 'go'],
        // Note: AILANG syntax highlighting coming soon - use 'typescript' for now
      },
      colorMode: {
        defaultMode: 'dark',
        disableSwitch: false,
        respectPrefersColorScheme: false,
      },
    }),
};

export default config;
