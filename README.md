# Khoai chain

``` bash
GOOS=windows GOARCH=amd64 go build -o dist/khoai-builder-windows.exe builder.go
GOOS=linux GOARCH=amd64 go build -o dist/khoai-builder-linux builder.go
GOOS=darwin GOARCH=amd64 go build -o dist/khoai-builder-darwin builder.go
```