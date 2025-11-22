#!/bin/bash

# --- CẤU HÌNH ---
# Bạn nhớ sửa cái VERSION này mỗi khi ra bản release mới nhé
VERSION="v0.0.0"
REPO="duongess/khoai-chain"
# ----------------

echo "🔍 Đang kiểm tra hệ điều hành..."

OS="$(uname -s)"
ARCH="$(uname -m)"
BINARY_NAME="khoai" # Tên file cuối cùng mà người dùng nhận được

# Xác định file cần tải từ GitHub Releases
case "${OS}" in
    Linux*)     
        FILE_ON_GITHUB="khoai-builder-linux"
        ;;
    Darwin*)    
        FILE_ON_GITHUB="khoai-builder-darwin"
        ;;
    CYGWIN*|MINGW*|MSYS*) 
        # Phát hiện Windows (khi chạy qua Git Bash)
        FILE_ON_GITHUB="khoai-builder-windows.exe"
        BINARY_NAME="khoai.exe" # Windows bắt buộc phải có đuôi exe
        ;;
    *)          
        echo "❌ Lỗi: Hệ điều hành ${OS} chưa được hỗ trợ!"
        exit 1
        ;;
esac

DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${FILE_ON_GITHUB}"

echo "⬇️  Đang tải ${FILE_ON_GITHUB}..."
echo "🔗 Link: ${DOWNLOAD_URL}"

# Tải về và đổi tên thành 'khoai' (hoặc khoai.exe) luôn
curl -L -o "${BINARY_NAME}" "${DOWNLOAD_URL}"

# Kiểm tra xem tải có thành công không
if [ $? -ne 0 ]; then
    echo "❌ Tải thất bại! Vui lòng kiểm tra lại kết nối mạng hoặc phiên bản release."
    exit 1
fi

# Cấp quyền thực thi (Chỉ cần thiết cho Linux/Mac)
if [[ "${OS}" != *"MINGW"* ]] && [[ "${OS}" != *"CYGWIN"* ]] && [[ "${OS}" != *"MSYS"* ]]; then
    chmod +x "${BINARY_NAME}"
fi

echo "--------------------------------------------------"
echo "✅ CÀI ĐẶT THÀNH CÔNG!"
echo "👉 Hãy gõ lệnh sau để bắt đầu:"
echo "   ./${BINARY_NAME}"
echo "--------------------------------------------------"