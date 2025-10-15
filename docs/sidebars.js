/**
 * Creating a sidebar enables you to:
 * - create an ordered group of docs
 * - render a sidebar for each doc of that group
 * - provide next/previous navigation
 *
 * The sidebars can be generated from the filesystem, or explicitly defined here.
 *
 * Create as many sidebars as you want.
 */

// @ts-check

/** @type {import('@docusaurus/plugin-content-docs').SidebarsConfig} */
const sidebars = {
  tutorialSidebar: [
    'intro',
    'playground',
    {
      type: 'category',
      label: 'Getting Started',
      items: [
        'guides/getting-started',
        'guides/module_execution',
      ],
    },
    {
      type: 'category',
      label: 'Language Reference',
      items: [
        'reference/language-syntax',
        'reference/implementation-status',
        'reference/repl-commands',
      ],
    },
    {
      type: 'category',
      label: 'AI & Prompts',
      items: [
        'guides/ai-prompt-guide',
        'prompts/index',
        'prompts/v0.3.6',
        'prompts/python',
      ],
    },
    {
      type: 'category',
      label: 'Development',
      items: [
        'guides/development',
        'guides/wasm-integration',
        'guides/benchmarking',
      ],
    },
    {
      type: 'category',
      label: 'Evaluation & Testing',
      collapsed: true,
      items: [
        'guides/evaluation/README',
        'guides/evaluation/architecture',
        'guides/evaluation/model-configuration',
        'guides/evaluation/eval-loop',
        'guides/evaluation/go-implementation',
        'guides/evaluation/baseline-tests',
        'guides/evaluation/migration-guide',
      ],
    },
    {
      type: 'category',
      label: 'Benchmarks',
      collapsed: true,
      items: [
        'benchmarks/performance',
      ],
    },
  ],
};

export default sidebars;
