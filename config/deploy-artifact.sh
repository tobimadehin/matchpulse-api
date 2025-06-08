#!/usr/bin/env bash
set -e

# Read the base version from version.txt
VERSION=$(cat version.txt | tr -d '[:space:]')
REGISTRY=ghcr.io/tobimadehin

echo "✅ Found BASE_VERSION $VERSION from version.txt"

# Look for existing versions
echo "🔍 Looking for existing versions with base $VERSION..."

# Create a temporary file to store found versions
TEMP_FILE=$(mktemp)

# Function to check existing versions
find_existing_versions() {
  # Look for all versions with this major.minor
  for i in {0..200}; do
    TEST_VERSION="$VERSION.$i"
    TEST_TAG="$TEST_VERSION"
    FULL_TAG="$REGISTRY/matchpulse:$TEST_TAG"
    
    # Use docker manifest inspect to check if the tag exists without pulling
    if docker manifest inspect "$FULL_TAG" > /dev/null 2>&1; then
      echo "$TEST_VERSION" >> "$TEMP_FILE"
    fi
  done
}

# Ensure CR_PAT is set
if [ -z "$CR_PAT" ]; then
  echo "❌ CR_PAT is not set. Please set the CR_PAT environment variable."
  exit 1
else
  echo "🔑 Using credentials for artifacts registry..."
fi

# Log in to Docker registry
echo "🔑 Logging into Docker registry..."
echo $CR_PAT | docker login ghcr.io -u USERNAME --password-stdin

# Find existing versions
find_existing_versions

# Determine if we have existing versions with this major.minor
if [ -s "$TEMP_FILE" ]; then
  # Get the highest version
  LATEST_VERSION=$(sort -V "$TEMP_FILE" | tail -1)
  echo "✅ Found existing versions with base $VERSION, latest is: $LATEST_VERSION"
  
  # Extract the patch number
  PATCH=$(echo "$LATEST_VERSION" | grep -oE '\.[0-9]+$' | grep -oE '[0-9]+')
  
  # Next patch is current + 1
  NEXT_PATCH=$((PATCH + 1))
  
  # Set the new version
  NEW_VERSION="$VERSION.$NEXT_PATCH"
else
  echo "🆕 No existing versions with base $VERSION found, starting from 0"
  NEW_VERSION="$VERSION.0"
fi

rm "$TEMP_FILE"

# Set the build tag
BUILD_TAG="$NEW_VERSION"
echo "✨ Using version: $BUILD_TAG"

# Build and push the Docker image
echo "🔨 Building Docker image..."
docker build -t matchpulse -f Dockerfile .

echo "🏷️ Tagging Docker image as $REGISTRY/matchpulse:$BUILD_TAG"
docker tag matchpulse $REGISTRY/matchpulse:$BUILD_TAG

echo "🚀 Pushing Docker image to registry..."
docker push $REGISTRY/matchpulse:$BUILD_TAG

# Tag and push latest if TAG_LATEST is true
if [ "$TAG_LATEST" = "true" ]; then
  echo "🏷️ Tagging Docker image as latest..."
  docker tag matchpulse $REGISTRY/matchpulse:latest
  echo "🚀 Pushing latest tag to registry..."
  docker push $REGISTRY/matchpulse:latest
  echo "✅ Latest tag pushed successfully"
fi

echo "✅ Done! Image pushed successfully: $REGISTRY/matchpulse:$BUILD_TAG"

# Export the version for use in subsequent scripts or steps
if [ -n "$GITHUB_OUTPUT" ]; then
  echo "build_tag=$BUILD_TAG" >> $GITHUB_OUTPUT
  echo "version=$NEW_VERSION" >> $GITHUB_OUTPUT
fi
