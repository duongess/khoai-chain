#!/bin/bash
set -e

VERSION=$1
TARGET_DIR=${2:-"build"} 
REPO="duongess/khoai-chain"

if [ -z "$VERSION" ] || [ "$VERSION" == "latest" ]; then
    echo "Determining latest version..." >&2
    VERSION=$(curl -s https://api.github.com/repos/${REPO}/releases/latest | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    if [ -z "$VERSION" ]; then
        echo "Error: Could not determine the latest version from GitHub." >&2
        exit 1
    fi
fi

VERSION_DIR="${TARGET_DIR}/dist"

rm -rf "${VERSION_DIR}"
mkdir -p "${VERSION_DIR}"

echo "Writing version ${VERSION} to ${TARGET_DIR}/.version" >&2
echo -n "${VERSION}" > "${TARGET_DIR}/.version"

FILE_NAME="khoai-src.zip"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${FILE_NAME}"

OUTPUT_PATH="${VERSION_DIR}/khoai-src-${VERSION}.zip"

echo "Downloading ${FILE_NAME} to ${OUTPUT_PATH} ..." >&2
curl -sSfL -o "${OUTPUT_PATH}" "${DOWNLOAD_URL}"

echo "${VERSION}"