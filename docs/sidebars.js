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
    'vision',
    {
      type: 'category',
      label: 'Architecture',
      items: [
        'architecture/shared-semantic-state',
      ],
    },
    'examples',
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
      label: 'Agent Integration',
      items: [
        'guides/claude-code-integration',
        'guides/hooks-setup',
        'guides/agent-workflows',
        'guides/state-system-workflow',
        'guides/agent-integration',
        'guides/agent-messaging',
      ],
    },
    {
      type: 'category',
      label: 'Language Reference',
      items: [
        'reference/language-syntax',
        'reference/implementation-status',
        'reference/repl-commands',
        'reference/no-loops',
        'reference/limitations',
      ],
    },
    {
      type: 'category',
      label: 'AI & Prompts',
      items: [
        'guides/ai-prompt-guide',
        'prompts/index',
        'prompts/v0.4.4',
        'prompts/python',
      ],
    },
    {
      type: 'category',
      label: 'Development',
      items: [
        'guides/development',
        'guides/testing',
        'guides/debugging',
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
