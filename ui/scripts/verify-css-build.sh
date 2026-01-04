#!/bin/bash
# Verify essential CSS classes are included in the build output

set -e

CSS_FILE=$(ls dist/assets/index-*.css 2>/dev/null | head -1)

if [ -z "$CSS_FILE" ]; then
    echo "ERROR: No CSS file found in dist/assets/"
    exit 1
fi

# Essential classes that MUST be in the build
REQUIRED_CLASSES=(
    "nav-button"
    "app-header"
    "app-body"
    "docs-link"
    "version-tag"
)

MISSING=0
for class in "${REQUIRED_CLASSES[@]}"; do
    if ! grep -q "$class" "$CSS_FILE"; then
        echo "ERROR: Missing required CSS class: .$class"
        MISSING=1
    fi
done

if [ $MISSING -eq 1 ]; then
    echo ""
    echo "CSS build validation FAILED!"
    echo "This usually means a CSS file was imported incorrectly."
    echo "Check: Are you using 'import ./file.module.css' as side-effect?"
    echo "Fix: Either use 'import styles from' or rename to .css (not .module.css)"
    exit 1
fi

echo "CSS build validation passed - all required classes present"
