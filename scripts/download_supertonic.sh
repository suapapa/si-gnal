#!/bin/bash

# 설정
TARGET_DIR="assets/supertonic2"
REPO_URL="https://huggingface.co/Supertone/supertonic-2"

echo "Downloading Supertonic-2 models from Hugging Face to $TARGET_DIR..."

# 기존 폴더가 없을 경우에만 복제 (빠르고 안전한 방식)
if [ ! -d "$TARGET_DIR" ]; then
    echo "Cloning repository..."
    # GIT_LFS_SKIP_SMUDGE=1 없이 복제하여 대용량 파일까지 모두 받습니다.
    # 단, git-lfs가 설치되어 있어야 합니다 (mac: brew install git-lfs)
    git clone $REPO_URL "$TARGET_DIR"
    
    # 선택: 불필요한 .git 정리 (최신 파일만 유지하고 용량 확보 시)
    # rm -rf "$TARGET_DIR/.git"
    
    echo "Download completed."
else
    echo "Directory $TARGET_DIR already exists."
    echo "To update the model, navigate to $TARGET_DIR and run 'git pull'."
fi
