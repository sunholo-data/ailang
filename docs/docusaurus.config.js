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

  markdown: {
    mermaid: true,
  },

  themes: ['@docusaurus/theme-mermaid'],

  // Set the production url of your site here
  url: 'https://ailang.sunholo.com',
  // Set the /<baseUrl>/ pathname under which your site is served
  // For GitHub pages deployment, it is often '/<projectName>/'
  baseUrl: '/',

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
      src: '/wasm/wasm_exec.js',
      async: false,
    },
    {
      src: '/js/ailang-repl.js',
      async: false,
    },
  ],

  // Google Fonts for Sunholo brand styling
  headTags: [
    {
      tagName: 'link',
      attributes: {
        rel: 'preconnect',
        href: 'https://fonts.googleapis.com',
      },
    },
    {
      tagName: 'link',
      attributes: {
        rel: 'preconnect',
        href: 'https://fonts.gstatic.com',
        crossorigin: 'anonymous',
      },
    },
    {
      tagName: 'link',
      attributes: {
        rel: 'stylesheet',
        href: 'https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&family=JetBrains+Mono:wght@400;500;600&family=Montserrat:wght@600;700;800&display=swap',
      },
    },
    {
      tagName: 'link',
      attributes: {
        rel: 'icon',
        type: 'image/svg+xml',
        href: '/img/ailang-logo.svg',
      },
    },
  ],

  // Even if you don't use internationalization, you can use this field to set
  // useful metadata like html lang. For example, if your site is Chinese, you
  // may want to replace "en" with "zh-Hans".
  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
  },

  plugins: [
    function webpackPlugin() {
      return {
        name: 'custom-webpack-plugin',
        configureWebpack() {
          return {
            module: {
              rules: [
                {
                  test: /\.ail$/,
                  use: 'raw-loader',
                },
              ],
            },
          };
        },
      };
    },
  ],

  presets: [
    [
      'classic',
      /** @type {import('@docusaurus/preset-classic').Options} */
      ({
        googleTagManager: {
          containerId: 'GTM-WLQZQF2P',
        },
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
          src: 'img/ailang-logo.svg',
        },
        items: [
          {
            to: '/docs/why-ailang',
            label: 'Why AILANG',
            position: 'left',
          },
          {
            to: '/docs/vision',
            label: 'Vision',
            position: 'left',
          },
          {
            type: 'docSidebar',
            sidebarId: 'tutorialSidebar',
            position: 'left',
            label: 'Documentation',
          },
          {
            to: '/docs/examples',
            label: 'Examples',
            position: 'left',
          },
          {
            to: '/docs/playground',
            label: 'Playground',
            position: 'left',
          },
          {
            to: '/docs/benchmarks/performance',
            label: 'Benchmarks',
            position: 'left',
          },
          {
            href: 'https://ailang.sunholo.com/llms.txt',
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
        logo: {
          alt: 'AILANG Logo',
          src: 'img/ailang-logo.svg',
          href: '/',
          width: 48,
          height: 48,
        },
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
              {
                label: 'Benchmarks',
                to: '/docs/benchmarks/performance',
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
                href: 'https://ailang.sunholo.com/llms.txt',
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
