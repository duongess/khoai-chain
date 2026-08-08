#!/bin/bash

set -e # Exit immediately if a command exits with a non-zero status.

VERSION=$1
REPO="duongess/khoai-chain"
BUILD_DIR="build"

# Create build directory if it doesn't exist
mkdir -p "$BUILD_DIR"

if [ -z "$VERSION" ] || [ "$VERSION" == "latest" ]; then
    echo "Determining latest version..." >&2
    LATEST_RELEASE_URL="https://api.github.com/repos/${REPO}/releases/latest"
    VERSION=$(curl -s https://api.github.com/repos/${REPO}/releases/latest | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    if [ -z "$VERSION" ]; then
        echo "Error: Could not determine the latest version from GitHub." >&2
        exit 1
    fi
fi

# Write version to build/.version file
echo "Writing version ${VERSION} to ${BUILD_DIR}/.version" >&2
echo -n "${VERSION}" > "${BUILD_DIR}/.version"

FILE_NAME="khoai-src-${VERSION}.zip"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${FILE_NAME}"
OUTPUT_PATH="${BUILD_DIR}/${FILE_NAME}"

echo "Downloading ${FILE_NAME} to ${BUILD_DIR}/ ..." >&2
curl -L -o "${OUTPUT_PATH}" "${DOWNLOAD_URL}"

echo "Extracting source code from ${OUTPUT_PATH}..." >&2
unzip -o "${OUTPUT_PATH}" -d .

# Output the determined version to stdout for the Go program to capture
echo "${VERSION}"

