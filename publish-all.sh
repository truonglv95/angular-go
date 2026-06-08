#!/usr/bin/env bash
set -euo pipefail

# List of package directories to publish in order (native platform packages first, then main package)
packages=(
  "./angular-packages/angular-go-darwin-arm64"
  "./angular-packages/angular-go-darwin-x64"
  "./angular-packages/angular-go-linux-x64"
  "./angular-packages/angular-go-win32-x64"
  "./angular-packages/angular-go"
)

echo "=== Bắt đầu xuất bản toàn bộ các package ==="

for pkg in "${packages[@]}"; do
  echo "----------------------------------------"
  echo "Đang xuất bản: ${pkg}..."
  # Use --access public since these are scoped packages under @angular-go
  npm publish "${pkg}" --access public
done

echo "----------------------------------------"
echo "=== Tất cả các package đã được xuất bản thành công! ==="
