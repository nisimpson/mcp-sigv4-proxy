#!/bin/bash
set -e

# Generate changelog from conventional commits
# Usage: ./changelog.sh [--dry-run]

DRY_RUN=false
if [ "$1" = "--dry-run" ]; then
    DRY_RUN=true
fi

# Get the latest tag
LATEST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "")

# Get commits since last tag
if [ -z "$LATEST_TAG" ]; then
    COMMITS=$(git log --pretty=format:"- %s (%h)" --reverse)
    RANGE="All commits (no previous tag)"
else
    COMMITS=$(git log ${LATEST_TAG}..HEAD --pretty=format:"- %s (%h)" --reverse)
    RANGE="${LATEST_TAG}..HEAD"
fi

# Group commits by type
FEATURES=$(echo "$COMMITS" | grep -E "^- (feat|feature)" || true)
FIXES=$(echo "$COMMITS" | grep -E "^- fix" || true)
OTHERS=$(echo "$COMMITS" | grep -vE "^- (feat|feature|fix)" || true)

# Generate changelog
generate_changelog() {
    echo "## What's Changed"
    echo ""
    
    if [ ! -z "$FEATURES" ]; then
        echo "### Features"
        echo "$FEATURES"
        echo ""
    fi
    
    if [ ! -z "$FIXES" ]; then
        echo "### Bug Fixes"
        echo "$FIXES"
        echo ""
    fi
    
    if [ ! -z "$OTHERS" ]; then
        echo "### Other Changes"
        echo "$OTHERS"
        echo ""
    fi
}

if [ "$DRY_RUN" = true ]; then
    echo "=== Changelog Preview (Dry Run) ==="
    echo ""
    echo "Range: $RANGE"
    echo ""
    echo "---"
    echo ""
    generate_changelog
    echo "---"
    echo ""
else
    generate_changelog
fi
