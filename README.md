# 1. Build cho Windows (Tạo ra file .exe)
GOOS=windows GOARCH=amd64 go build -o dist/khoai-builder-windows.exe cmd/builder/main.go

# 2. Build cho Linux (Dùng cho máy chủ Ubuntu/CentOS...)
GOOS=linux GOARCH=amd64 go build -o dist/khoai-builder-linux cmd/builder/main.go

# 3. Build cho macOS (Dùng cho Macbook chip Intel và M1/M2)
GOOS=darwin GOARCH=amd64 go build -o dist/khoai-builder-darwin cmd/builder/main.go