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
    'why-ailang',
    'vision',
    'examples',
    'playground',
    {
      type: 'category',
      label: 'Getting Started',
      items: [
        'guides/ailang-vs-agents',
        'guides/getting-started',
        'guides/quick-start-examples',
        'guides/editor-setup',
        'guides/module_execution',
      ],
    },
    {
      type: 'category',
      label: 'Language Reference',
      items: [
        'reference/language-syntax',
        'reference/modules',
        'reference/effects',
        'reference/capability-budgets',
        'guides/contracts',
        'reference/arrays',
        'reference/no-loops',
        'reference/repl-commands',
      ],
    },
    {
      type: 'category',
      label: 'Agent Integration',
      items: [
        'guides/claude-code-integration',
        'guides/agent-integration',
        'guides/hooks-setup',
        'guides/agent-workflows',
        'guides/state-system-workflow',
        'guides/agent-messaging',
        'guides/cross-project-messaging',
        'guides/collaboration-hub',
      ],
    },
    {
      type: 'category',
      label: 'Semantic Features',
      items: [
        'guides/semantic-search',
        'guides/semantic-caching-how-to',
        'guides/semantic-caching-vs-vectordb',
      ],
    },
    {
      type: 'category',
      label: 'AI & Prompts',
      items: [
        'guides/ai-effect',
        'guides/ai-prompt-guide',
        'prompts/index',
        'prompts/current',
      ],
    },
    {
      type: 'category',
      label: 'Development',
      items: [
        'guides/development-workflow',
        'guides/development',
        'guides/testing',
        'guides/debugging',
        'guides/telemetry',
        'guides/go-interop',
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
        'benchmarks/codebase-stats',
      ],
    },
    {
      type: 'category',
      label: 'Architecture',
      collapsed: true,
      items: [
        'architecture/index',
        'architecture/types',
        'architecture/anf',
        'architecture/adding-operators',
        'architecture/debug-tools',
      ],
    },
    {
      type: 'category',
      label: 'References',
      collapsed: false,
      items: [
        'reference/implementation-status',
        'reference/limitations',
        {
          type: 'category',
          label: 'Academic Foundations',
          items: [
            'references/axioms',
            'references/design-lineage',
            'references/philosophical-foundations',
            'references/index',
          ],
        },
      ],
    },
    'roadmap/index',
    'design-docs',
  ],
};

export default sidebars;
