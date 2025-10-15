# AILANG Documentation

This directory contains the AILANG documentation website built with [Docusaurus](https://docusaurus.io/).

**Live site:** [https://sunholo-data.github.io/ailang/](https://sunholo-data.github.io/ailang/)

## Quick Start

### Local Development

```bash
cd docs
npm install
npm start
```

This starts a local development server and opens a browser window. Most changes are reflected live without restarting the server.

**Local URL:** [http://localhost:3000/ailang/](http://localhost:3000/ailang/)

### Build

```bash
npm run build
```

Generates static content into the `build` directory.

### Test Production Build

```bash
npm run serve
```

Serves the production build locally.

## Directory Structure

```
docs/
├── docs/                  # Documentation content
│   ├── intro.md          # Homepage
│   ├── guides/           # Tutorials and guides
│   ├── reference/        # API and language reference
│   ├── prompts/          # AI teaching prompts
│   └── ...
├── blog/                 # Blog posts (release notes, updates)
├── src/                  # Custom React components
│   └── css/             # Custom styles
├── static/               # Static assets
│   ├── img/             # Images
│   └── llms.txt         # AI-readable documentation
├── docusaurus.config.js  # Docusaurus configuration
├── sidebars.js          # Sidebar structure (auto-generated)
└── package.json         # Dependencies
```

## Commands

- `npm start` - Start local development server
- `npm run build` - Build production site
- `npm run serve` - Serve built site locally
- `npm run clear` - Clear Docusaurus cache

## Deployment

Documentation is automatically deployed to GitHub Pages when changes are pushed to `main` or `dev` branches via `.github/workflows/jekyll-gh-pages.yml`.

## Adding New Pages

1. Create a `.md` or `.mdx` file in `docs/docs/`
2. Optional front matter:

```yaml
---
sidebar_position: 1
title: Your Page Title
---
```

3. Write your content in Markdown or MDX
4. Sidebar is auto-generated from file structure
5. Preview with `npm start`
6. Commit and push - CI will deploy automatically

## Migrated from Jekyll (October 2025)

This site was previously built with Jekyll. Benefits of Docusaurus:
- Better developer experience (hot reload, fast builds)
- React-based - can add interactive components
- Better search (Algolia integration available)
- Versioned docs support
- Active development and maintenance
- Modern UI with dark mode by default

## Troubleshooting

**Port already in use:**
```bash
npm start -- --port 3001
```

**Build errors:**
```bash
npm run clear
rm -rf node_modules package-lock.json
npm install
npm run build
```

**Broken links:**
Links to files outside `docs/` will show warnings. Use absolute GitHub URLs for external files like CHANGELOG.md.

## Resources

- [Docusaurus Documentation](https://docusaurus.io/docs)
- [Markdown Features](https://docusaurus.io/docs/markdown-features)
- [MDX and React](https://docusaurus.io/docs/markdown-features/react)
