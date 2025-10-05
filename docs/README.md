---
layout: page
title: Documentation README
nav_exclude: true
---

# AILANG Website Documentation

This directory contains the source for the AILANG documentation website hosted at [https://sunholo-data.github.io/ailang/](https://sunholo-data.github.io/ailang/)

## Quick Start

### Preview Locally

**First time setup:**
```bash
make docs-install
```

**Run the website locally:**
```bash
make docs-serve
```

Then open: [http://localhost:4000/ailang/](http://localhost:4000/ailang/)

**Regenerate docs and preview:**
```bash
make docs-preview
```

### Manual Jekyll Commands

If you prefer to use Jekyll directly:

```bash
cd docs

# Install dependencies (first time only)
bundle install

# Serve the site
bundle exec jekyll serve --baseurl /ailang

# Build static files
bundle exec jekyll build
```

## Directory Structure

```
docs/
├── _config.yml           # Jekyll configuration
├── index.md              # Homepage
├── guides/               # User guides
│   ├── getting-started.md
│   ├── development.md
│   ├── ai-prompt-guide.md
│   └── ...
├── reference/            # Reference documentation
│   ├── language-syntax.md
│   ├── implementation-status.md
│   └── repl-commands.md
├── prompts/              # AI teaching prompts (auto-synced)
│   ├── v0.3.0.md
│   ├── v0.2.0.md
│   └── python.md
├── llms.txt              # Consolidated docs for LLMs
└── assets/               # Images, CSS, etc.
```

## Theme

The site uses [Just the Docs](https://just-the-docs.github.io/just-the-docs/) theme for:
- Better navigation
- Search functionality
- Mobile responsiveness
- Heading anchors

## Automation

The following are automatically updated by CI/CD:

- `docs/prompts/` - Synced from `prompts/` on every push to dev
- `docs/llms.txt` - Regenerated from all docs on every push to dev

## Adding New Pages

1. Create a `.md` file in the appropriate directory
2. Add Jekyll front matter:

```yaml
---
layout: page
title: Your Page Title
parent: Parent Section (optional)
nav_order: 1
---
```

3. Write your content in Markdown
4. Preview locally with `make docs-serve`
5. Commit and push - GitHub Pages will rebuild automatically

## Front Matter Options

```yaml
layout: page           # Use 'page' for most docs
title: Page Title      # Shows in navigation
parent: Section Name   # Creates hierarchy
nav_order: 1          # Controls order in nav
nav_exclude: true     # Hide from navigation
has_children: true    # This page has children
```

## Troubleshooting

**Port already in use:**
```bash
# Kill existing Jekyll process
pkill -f jekyll
# Or use a different port
cd docs && bundle exec jekyll serve --port 4001 --baseurl /ailang
```

**Gems not installing:**
```bash
# Update bundler
gem install bundler
# Clean and reinstall
cd docs && rm -rf Gemfile.lock && bundle install
```

**Changes not showing:**
- Hard refresh: Cmd+Shift+R (Mac) or Ctrl+Shift+R (Windows)
- Clear Jekyll cache: `rm -rf docs/_site docs/.jekyll-cache`

## Resources

- [Just the Docs Documentation](https://just-the-docs.github.io/just-the-docs/)
- [Jekyll Documentation](https://jekyllrb.com/docs/)
- [GitHub Pages Documentation](https://docs.github.com/en/pages)
- [Markdown Guide](https://www.markdownguide.org/)
