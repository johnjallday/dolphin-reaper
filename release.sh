#!/bin/bash

# Release script for dolphin-reaper plugin
# Creates a GitHub release with the built .so file

set -e # Exit on any error

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

PLUGIN_NAME="reascript_launcher"
OUTPUT_NAME="reascript_launcher.so"

# Function to show usage
show_usage() {
    echo "Usage: $0 [VERSION_TYPE] [VERSION]"
    echo ""
    echo "VERSION_TYPE:"
    echo "  patch    - Increment patch version (0.0.1 -> 0.0.2)"
    echo "  minor    - Increment minor version (0.0.1 -> 0.1.0)"  
    echo "  major    - Increment major version (0.0.1 -> 1.0.0)"
    echo "  custom   - Set specific version (requires VERSION argument)"
    echo "  current  - Use current VERSION file without bumping"
    echo ""
    echo "Examples:"
    echo "  $0 patch              # Auto-increment patch version"
    echo "  $0 minor              # Auto-increment minor version"
    echo "  $0 custom 1.2.3       # Set version to 1.2.3"
    echo "  $0 current            # Use current version"
    echo "  $0                    # Interactive mode"
    exit 1
}

# Function to increment version
increment_version() {
    local version=$1
    local type=$2
    
    IFS='.' read -r major minor patch <<< "$version"
    
    case $type in
        patch)
            patch=$((patch + 1))
            ;;
        minor)
            minor=$((minor + 1))
            patch=0
            ;;
        major)
            major=$((major + 1))
            minor=0
            patch=0
            ;;
        *)
            echo -e "${RED}❌ Invalid version type: $type${NC}"
            exit 1
            ;;
    esac
    
    echo "$major.$minor.$patch"
}

# Get current version
CURRENT_VERSION=$(cat VERSION 2>/dev/null || echo "0.0.0")
echo -e "${BLUE}📋 Current version: $CURRENT_VERSION${NC}"

# Handle version argument
if [ $# -eq 0 ]; then
    # Interactive mode
    echo ""
    echo "Select version action:"
    echo "1) Patch (bug fixes)     - $CURRENT_VERSION -> $(increment_version $CURRENT_VERSION patch)"
    echo "2) Minor (new features)  - $CURRENT_VERSION -> $(increment_version $CURRENT_VERSION minor)"
    echo "3) Major (breaking)      - $CURRENT_VERSION -> $(increment_version $CURRENT_VERSION major)"
    echo "4) Custom version"
    echo "5) Use current version ($CURRENT_VERSION)"
    echo "6) Exit"
    
    read -p "Enter choice [1-6]: " choice
    
    case $choice in
        1) VERSION_TYPE="patch" ;;
        2) VERSION_TYPE="minor" ;;
        3) VERSION_TYPE="major" ;;
        4) 
            read -p "Enter custom version (e.g., 1.2.3): " CUSTOM_VERSION
            if [[ ! $CUSTOM_VERSION =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
                echo -e "${RED}❌ Invalid version format. Use semantic versioning (x.y.z)${NC}"
                exit 1
            fi
            VERSION_TYPE="custom"
            ;;
        5) VERSION_TYPE="current" ;;
        6) echo "Cancelled."; exit 0 ;;
        *) echo -e "${RED}❌ Invalid choice${NC}"; exit 1 ;;
    esac
elif [ "$1" = "current" ]; then
    VERSION_TYPE="current"
elif [ $# -eq 1 ]; then
    VERSION_TYPE=$1
    if [ "$VERSION_TYPE" = "custom" ]; then
        echo -e "${RED}❌ Custom version type requires a version number${NC}"
        show_usage
    fi
elif [ $# -eq 2 ]; then
    VERSION_TYPE=$1
    CUSTOM_VERSION=$2
    if [ "$VERSION_TYPE" != "custom" ]; then
        echo -e "${RED}❌ Version number only allowed with 'custom' type${NC}"
        show_usage
    fi
else
    show_usage
fi

# Calculate version to use
if [ "$VERSION_TYPE" = "custom" ]; then
    VERSION=$CUSTOM_VERSION
    echo "$VERSION" > VERSION
    echo -e "${GREEN}✅ VERSION file updated to $VERSION${NC}"
elif [ "$VERSION_TYPE" = "current" ]; then
    VERSION=$CURRENT_VERSION
    echo -e "${BLUE}📋 Using current version: $VERSION${NC}"
else
    VERSION=$(increment_version $CURRENT_VERSION $VERSION_TYPE)
    echo "$VERSION" > VERSION
    echo -e "${GREEN}✅ VERSION file updated to $VERSION${NC}"
fi

TAG="v$VERSION"

echo -e "${BLUE}Creating release for dolphin-reaper $TAG...${NC}"

# Check if we're in a git repository
if ! git rev-parse --git-dir >/dev/null 2>&1; then
  echo -e "${RED}✗ Not in a git repository${NC}"
  exit 1
fi

# Check if there are uncommitted changes
if ! git diff-index --quiet HEAD --; then
  echo -e "${RED}✗ There are uncommitted changes. Please commit or stash them first.${NC}"
  exit 1
fi

# Build the plugin
echo -e "${YELLOW}Building plugin...${NC}"
if ! ./build.sh; then
  echo -e "${RED}✗ Build failed${NC}"
  exit 1
fi

# Verify the .so file exists
if [ ! -f "$OUTPUT_NAME" ]; then
  echo -e "${RED}✗ Built plugin file $OUTPUT_NAME not found${NC}"
  exit 1
fi

# Check if tag already exists
if git rev-parse "$TAG" >/dev/null 2>&1; then
  echo -e "${YELLOW}⚠ Tag $TAG already exists. Deleting existing tag and release...${NC}"
  git tag -d "$TAG" || true
  git push origin ":refs/tags/$TAG" || true
  gh release delete "$TAG" --yes || true
fi

# Create git tag
echo -e "${YELLOW}Creating git tag $TAG...${NC}"
git tag "$TAG"
git push origin "$TAG"

# Create GitHub release with the .so file
echo -e "${YELLOW}Creating GitHub release...${NC}"
gh release create "$TAG" \
  --title "Release $TAG" \
  --notes "Release $TAG of dolphin-reaper REAPER script launcher plugin

## Installation
Download the \`$OUTPUT_NAME\` file and place it in your dolphin-agent plugins directory.

## Changes
- Version $VERSION release" \
  "$OUTPUT_NAME"

echo -e "${GREEN}🎉 Release $TAG created successfully!${NC}"
echo -e "${BLUE}Plugin file $OUTPUT_NAME has been uploaded to the release.${NC}"
echo -e "${BLUE}Users can download it from: https://github.com/johnjallday/dolphin-reaper/releases/tag/$TAG${NC}"
