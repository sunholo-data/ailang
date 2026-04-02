#!/bin/bash
# Sync package registry data and generate static MDX pages for Docusaurus.
# Fetches index.json from registry-validator API (or direct GCS), generates:
#   - docs/static/registry/index.json (snapshot)
#   - docs/docs/packages/{vendor}/index.mdx (vendor index per vendor)
#   - docs/docs/packages/{vendor}/{name}.mdx (package detail per package)
#   - docs/src/data/packages-sidebar.json (sidebar items for sidebars.js)
#
# Graceful failure: if registry is unavailable, exits 0 with a warning.
# The site build continues without generated package pages.

set -euo pipefail

cd "$(dirname "$0")/../.."

REGISTRY_API="${AILANG_REGISTRY_VALIDATOR:-${AILANG_REGISTRY_API:-https://registry.ailang.sunholo.com}}"
REGISTRY_GCS="${AILANG_REGISTRY:-https://storage.googleapis.com/ailang-registry}"
DOCS_DIR="docs"
STATIC_DIR="$DOCS_DIR/static/registry"
PACKAGES_DIR="$DOCS_DIR/docs/packages"
TEMPLATES_DIR="$DOCS_DIR/src/templates"
DATA_DIR="$DOCS_DIR/src/data"

echo "🔄 Syncing package registry data..."

# Clean previous generated files (keep hand-written files)
rm -rf "$STATIC_DIR"
mkdir -p "$STATIC_DIR"
mkdir -p "$DATA_DIR"

# Fetch index.json — try API first, fall back to direct GCS
INDEX_JSON=""
if curl -sf --max-time 10 "$REGISTRY_API/api/packages" -o "$STATIC_DIR/index.json" 2>/dev/null; then
  echo "  ✓ Fetched index from registry-validator API"
  INDEX_JSON="$STATIC_DIR/index.json"
elif curl -sf --max-time 10 "$REGISTRY_GCS/index.json" -o "$STATIC_DIR/index.json" 2>/dev/null; then
  echo "  ✓ Fetched index from GCS bucket"
  INDEX_JSON="$STATIC_DIR/index.json"
else
  echo "  ⚠ Could not fetch registry index — skipping package page generation"
  # Write empty sidebar so sidebars.js doesn't break
  echo "[]" > "$DATA_DIR/packages-sidebar.json"
  exit 0
fi

# Parse package count
PKG_COUNT=$(jq '.packages | length' "$INDEX_JSON")
echo "  Found $PKG_COUNT packages"

if [ "$PKG_COUNT" -eq 0 ]; then
  echo "  ⚠ No packages in registry — writing empty sidebar"
  echo "[]" > "$DATA_DIR/packages-sidebar.json"
  exit 0
fi

# Extract unique vendors
VENDORS=$(jq -r '.packages[].name' "$INDEX_JSON" | cut -d/ -f1 | sort -u)

# Generate per-vendor directories and pages
SIDEBAR_JSON="["
FIRST_VENDOR=true

for VENDOR in $VENDORS; do
  VENDOR_DIR="$PACKAGES_DIR/$VENDOR"
  mkdir -p "$VENDOR_DIR"

  # Get packages for this vendor
  VENDOR_PKGS=$(jq -c "[.packages[] | select(.name | startswith(\"$VENDOR/\"))]" "$INDEX_JSON")
  VENDOR_PKG_COUNT=$(echo "$VENDOR_PKGS" | jq 'length')

  echo "  Generating $VENDOR/ ($VENDOR_PKG_COUNT packages)"

  # --- Vendor index page ---
  VENDOR_CARDS_JSON=$(echo "$VENDOR_PKGS" | jq -c '[.[] | {name, latest, ai_summary, tags, effects, stability, exports: (.exports | length), last_updated, updated_by}]')

  cat > "$VENDOR_DIR/index.mdx" <<VENDOREOF
---
title: "${VENDOR} Packages"
description: "All AILANG packages published by ${VENDOR}"
sidebar_label: "${VENDOR}"
---

import VendorIndex from '@site/src/components/PackageExplorer/VendorIndex';

# ${VENDOR} Packages

${VENDOR_PKG_COUNT} packages published by **${VENDOR}**.

<VendorIndex
  vendor="${VENDOR}"
  packages={${VENDOR_CARDS_JSON}}
/>
VENDOREOF

  # --- Per-package detail pages ---
  SIDEBAR_ITEMS="["
  FIRST_PKG=true

  echo "$VENDOR_PKGS" | jq -c '.[]' | while IFS= read -r PKG_JSON; do
    FULL_NAME=$(echo "$PKG_JSON" | jq -r '.name')
    SHORT_NAME=$(echo "$FULL_NAME" | cut -d/ -f2)
    LATEST=$(echo "$PKG_JSON" | jq -r '.latest')
    AI_SUMMARY=$(echo "$PKG_JSON" | jq -r '.ai_summary // "No description"')
    STABILITY=$(echo "$PKG_JSON" | jq -r '.stability // "experimental"')
    EFFECTS_RAW=$(echo "$PKG_JSON" | jq -r '.effects // [] | if length == 0 then "Pure" else join(", ") end')
    EXPORTS_COUNT=$(echo "$PKG_JSON" | jq -r '.exports | length')
    LAST_UPDATED=$(echo "$PKG_JSON" | jq -r '.last_updated // "unknown"')
    STATIC_DATA=$(echo "$PKG_JSON" | jq -c '.')

    # Save per-package static data
    PKG_STATIC_DIR="$STATIC_DIR/$VENDOR/$SHORT_NAME"
    mkdir -p "$PKG_STATIC_DIR"
    echo "$PKG_JSON" > "$PKG_STATIC_DIR/index.json"

    cat > "$VENDOR_DIR/$SHORT_NAME.mdx" <<PKGEOF
---
title: "${FULL_NAME}"
description: "${AI_SUMMARY}"
sidebar_label: "${SHORT_NAME}"
---

import PackageDetail from '@site/src/components/PackageExplorer/PackageDetail';

# ${FULL_NAME}

> ${AI_SUMMARY}

| Field | Value |
|-------|-------|
| Latest | \`${LATEST}\` |
| Stability | ${STABILITY} |
| Effects | ${EFFECTS_RAW} |
| Exports | ${EXPORTS_COUNT} modules |
| Last Updated | ${LAST_UPDATED} |

<PackageDetail
  packageName="${FULL_NAME}"
  staticData={${STATIC_DATA}}
/>
PKGEOF
  done

  # Build sidebar items for this vendor
  VENDOR_SIDEBAR_ITEMS=$(echo "$VENDOR_PKGS" | jq -r '.[].name' | while IFS= read -r FULL_NAME; do
    SHORT_NAME=$(echo "$FULL_NAME" | cut -d/ -f2)
    echo "\"packages/$VENDOR/$SHORT_NAME\""
  done | paste -sd, -)

  if [ "$FIRST_VENDOR" = true ]; then
    FIRST_VENDOR=false
  else
    SIDEBAR_JSON="$SIDEBAR_JSON,"
  fi

  SIDEBAR_JSON="$SIDEBAR_JSON
  {
    \"type\": \"category\",
    \"label\": \"$VENDOR\",
    \"link\": {\"type\": \"doc\", \"id\": \"packages/$VENDOR/index\"},
    \"items\": [$VENDOR_SIDEBAR_ITEMS]
  }"
done

SIDEBAR_JSON="$SIDEBAR_JSON
]"

echo "$SIDEBAR_JSON" > "$DATA_DIR/packages-sidebar.json"

echo "✓ Generated $PKG_COUNT package pages across $(echo "$VENDORS" | wc -w | tr -d ' ') vendors"
echo "  Static data: $STATIC_DIR/"
echo "  Package pages: $PACKAGES_DIR/"
echo "  Sidebar: $DATA_DIR/packages-sidebar.json"
