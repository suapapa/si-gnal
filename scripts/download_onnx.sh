#!/bin/bash
set -euo pipefail

# download ONNX runtime release from
# https://github.com/microsoft/onnxruntime

VERSION="${VERSION:-1.27.1}"
PLATFORM="${PLATFORM:-linux-x64}"
TARGET_DIR="${TARGET_DIR:-assets/onnx}"

ARCHIVE="onnxruntime-${PLATFORM}-${VERSION}.tgz"
URL="https://github.com/microsoft/onnxruntime/releases/download/v${VERSION}/${ARCHIVE}"

mkdir -p "$TARGET_DIR"

echo "Downloading ${ARCHIVE} to ${TARGET_DIR}..."
echo "URL: ${URL}"

wget -O "${TARGET_DIR}/${ARCHIVE}" "${URL}"

echo "Download completed: ${TARGET_DIR}/${ARCHIVE}"
