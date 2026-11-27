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
    // Remote MCP server discovery — agents that scan rel="mcp" can pick up
    // mcp.ailang.sunholo.com as the canonical structured-data endpoint for
    // AILANG docs/stdlib/examples/benchmarks. See docs/guides/agent-mcp.md.
    {
      tagName: 'link',
      attributes: {
        rel: 'mcp',
        href: 'https://mcp.ailang.sunholo.com/mcp/',
        title: 'AILANG MCP — versioned docs, stdlib, examples, benchmarks',
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
    [
      '@docusaurus/plugin-client-redirects',
      {
        redirects: [
          // Merged guides & recipes
          {
            from: '/docs/guides/three-camps-self-audit',
            to: '/docs/guides/three-camps-comparison',
          },
          {
            from: '/docs/recipes/ai-tool-loop',
            to: '/docs/guides/ai-effect',
          },
          // Superseded historical prompt snapshots + active version alias -> /docs/prompts/current
          {
            from: [
              '/docs/prompts/v0.8.2',
              '/docs/prompts/v0.9.0',
              '/docs/prompts/v0.11.4',
              '/docs/prompts/v0.12.1',
              '/docs/prompts/v0.16.0',
              '/docs/prompts/v0.16.1',
              '/docs/prompts/v0.16.2',
              '/docs/prompts/v0.16.3',
              '/docs/prompts/v0.16.4',
              '/docs/prompts/v0.16.5',
              '/docs/prompts/v0.16.6',
              '/docs/prompts/v0.16.7',
              '/docs/prompts/v0.16.8',
              '/docs/prompts/v0.16.9',
              '/docs/prompts/v0.16.10',
              '/docs/prompts/v0.16.11',
              '/docs/prompts/v0.16.12',
              '/docs/prompts/v0.16.13',
              '/docs/prompts/v0.16.14',
              '/docs/prompts/v0.16.15',
              '/docs/prompts/v0.16.16',
              '/docs/prompts/v0.16.17',
              '/docs/prompts/v0.16.18',
              '/docs/prompts/v0.16.20',
            ],
            to: '/docs/prompts/current',
          },
          // Relocated internal operator / fleet runbooks (moved to docs/internal/runbooks/)
          {
            from: [
              '/docs/guides/mission-bootstrap',
              '/docs/guides/mission-model-fleet',
              '/docs/guides/mission-iteration',
              '/docs/guides/mission-role-dispatch',
              '/docs/guides/claude-api-credits',
              '/docs/guides/coordinator-workers',
              '/docs/guides/notify-daemon',
            ],
            to: '/docs/guides/coordinator',
          },
          {
            from: [
              '/docs/guides/maintenance-benchmark',
              '/docs/guides/database-architecture',
              '/docs/guides/evaluation/local-ollama',
            ],
            to: '/docs/guides/development-workflow',
          },
        ],
      },
    ],
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
            'https://github.com/sunholo-data/ailang/tree/dev/docs/',
        },
        blog: false,
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
            type: 'docSidebar',
            sidebarId: 'tutorialSidebar',
            position: 'left',
            label: 'Docs',
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
              {
                label: 'Feedback & Contributing',
                to: '/docs/feedback',
              },
            ],
          },
          {
            title: 'Try It',
            items: [
              {
                label: 'Live Demos',
                href: 'https://www.sunholo.com/ailang-demos/',
              },
              {
                label: 'Playground',
                to: '/docs/playground',
              },
              {
                label: 'Examples',
                to: '/docs/examples',
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
              {
                label: 'llms-full.txt',
                href: 'https://ailang.sunholo.com/llms-full.txt',
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
        // `ailang` is registered in src/theme/prism-include-languages.js (grammar: src/theme/prism-ailang.js).
        // Fence AILANG as ```ailang, never typescript: make check-doc-samples checks ailang fences.
      },
      colorMode: {
        defaultMode: 'dark',
        disableSwitch: false,
        respectPrefersColorScheme: false,
      },
    }),
};

export default config;
