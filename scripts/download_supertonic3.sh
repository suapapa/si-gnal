#!/bin/bash
set -euo pipefail

# 설정
HF_PRJ="https://huggingface.co/Supertone/supertonic-3"
TG_DIR="assets/supertonic3"
REPO_ID="${HF_PRJ#https://huggingface.co/}"

echo "Downloading Supertonic-3 models from Hugging Face to $TG_DIR..."

if ! command -v hf >/dev/null 2>&1; then
    echo "Error: 'hf' CLI not found. Install with: pip install -U huggingface_hub[cli]"
    exit 1
fi

if [ ! -d "$TG_DIR" ]; then
    echo "Downloading repository via hf..."
    hf download "$REPO_ID" --local-dir "$TG_DIR"
    echo "Download completed."
else
    echo "Directory $TG_DIR already exists."
    echo "To refresh the model, remove $TG_DIR and re-run this script,"
    echo "or run: hf download $REPO_ID --local-dir $TG_DIR"
fi
