#!/bin/bash

# 设置错误即停止
set -e

# 设置输出目录和文件
BIN_DIR="./bin"
TARGET_NAME="antigravity-decryptor"
OUTPUT_PATH="${BIN_DIR}/${TARGET_NAME}"

# 确保 bin 目录存在
mkdir -p "$BIN_DIR"

echo "Building ${TARGET_NAME}..."

# 执行编译
go build -o "$OUTPUT_PATH" ./cmd/antigravity-decryptor/

# 赋予执行权限
chmod +x "$OUTPUT_PATH"

echo "Build successful! Binary location: ${OUTPUT_PATH}"
./bin/antigravity-decryptor --version || echo "Note: --version flag not implemented, but binary is ready."
