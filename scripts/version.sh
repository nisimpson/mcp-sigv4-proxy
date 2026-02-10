#!/bin/bash
set -e

# Calculate next semantic version based on conventional commits
# Usage: ./version.sh [--dry-run]

DRY_RUN=false
if [ "$1" = "--dry-run" ]; then
    DRY_RUN=true
fi

# Get the latest tag
LATEST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")

# Extract version numbers
VERSION=${LATEST_TAG#v}
IFS='.' read -r MAJOR MINOR PATCH <<< "$VERSION"

# Get commit messages since last tag
if [ "$LATEST_TAG" = "v0.0.0" ]; then
    COMMITS=$(git log --pretty=format:"%s" 2>/dev/null || echo "")
else
    COMMITS=$(git log ${LATEST_TAG}..HEAD --pretty=format:"%s" 2>/dev/null || echo "")
fi

# Determine version bump based on conventional commits
BUMP_TYPE="patch"
if echo "$COMMITS" | grep -qE "^(feat|feature)\(.*\)!:|^BREAKING CHANGE:"; then
    # Major version bump for breaking changes
    MAJOR=$((MAJOR + 1))
    MINOR=0
    PATCH=0
    BUMP_TYPE="major"
elif echo "$COMMITS" | grep -qE "^(feat|feature)(\(.*\))?:"; then
    # Minor version bump for new features
    MINOR=$((MINOR + 1))
    PATCH=0
    BUMP_TYPE="minor"
else
    # Patch version bump for fixes and other changes
    PATCH=$((PATCH + 1))
fi

NEW_VERSION="v${MAJOR}.${MINOR}.${PATCH}"

if [ "$DRY_RUN" = true ]; then
    echo "=== Version Calculation (Dry Run) ==="
    echo ""
    echo "Current tag: $LATEST_TAG"
    echo "Current version: ${LATEST_TAG#v}"
    echo ""
    echo "Commits since last tag:"
    if [ "$LATEST_TAG" = "v0.0.0" ]; then
        git log --pretty=format:"  %s (%h)" --reverse 2>/dev/null || echo "  (no commits)"
    else
        git log ${LATEST_TAG}..HEAD --pretty=format:"  %s (%h)" --reverse 2>/dev/null || echo "  (no commits)"
    fi
    echo ""
    echo ""
    echo "Version bump: ${BUMP_TYPE^^} ($(echo "$BUMP_TYPE" | sed 's/patch/fixes\/other changes/;s/minor/new features detected/;s/major/breaking changes detected/'))"
    echo ""
    echo "Next version: $NEW_VERSION"
    echo ""
else
    echo "$NEW_VERSION"
fi
