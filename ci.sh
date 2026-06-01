#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PARITY_OUT="${ROOT_DIR}/.tmp/ngtsc-parity-ci"

cd "${ROOT_DIR}"

if [[ -s "${HOME}/.gvm/scripts/gvm" ]]; then
  # Keep local developer runs aligned with the Go version used for compiler work.
  # CI without gvm can rely on its configured go binary.
  # shellcheck source=/dev/null
  set +eu
  source "${HOME}/.gvm/scripts/gvm"
  gvm use go1.26.3 >/dev/null || true
  set -eu
fi

echo "Running Go unit/integration tests..."
go test ./angular-packages/compiler/template/pipeline/phases
go test ./angular-packages/compiler_cli/integration_tests
go test ./angular-packages/compiler_cli/linker

echo "Building go-ngc..."
go build -o go-ngc ./angular-packages/compiler_cli/cmd/ngtsc/main.go

echo "Running ngtsc parity gates..."
rm -rf "${PARITY_OUT}"
go run ./angular-packages/compiler_cli/tools/parity_compare --kind render3 --all --strict --out "${PARITY_OUT}"
go run ./angular-packages/compiler_cli/tools/parity_compare --kind linker --all --strict --out "${PARITY_OUT}"

echo "Building Vite plugin..."
npm run build --prefix angular-packages/vite-plugin-angular-go

echo "Building new-demo-app through Vite..."
(command cd angular-packages/new-demo-app && npx vite build --config vite.config.ts)

echo "Running Playwright smoke..."
(command cd angular-packages/new-demo-app && npx playwright test)

echo "CI checks passed."
