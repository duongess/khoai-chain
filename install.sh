#!/bin/bash

VERSION=$1
REPO="duongess/khoai-chain"

if [ -z "$VERSION" ] || [ "$VERSION" == "latest" ]; then
    VERSION=$(curl -s https://api.github.com/repos/${REPO}/releases/latest | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
fi

FILE_NAME="khoai-src-${VERSION}.zip"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${FILE_NAME}"

curl -L -o "${FILE_NAME}" "${DOWNLOAD_URL}"