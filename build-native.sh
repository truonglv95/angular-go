#!/bin/bash
set -e

cd angular-packages

# Bump versions
for pkg in angular-go angular-go-darwin-arm64 angular-go-darwin-x64 angular-go-linux-x64 angular-go-win32-x64; do
  sed -i '' 's/"version": "1.0.10"/"version": "1.0.11"/g' $pkg/package.json
  sed -i '' 's/"1.0.10"/"1.0.11"/g' $pkg/package.json
done

# Build binaries
cd compiler_cli

# Mac ARM64
GOOS=darwin GOARCH=arm64 go build -buildvcs=false -o ../angular-go-darwin-arm64/bin/go-ngc ./cmd/ngtsc
GOOS=darwin GOARCH=arm64 go build -buildvcs=false -o ../angular-go-darwin-arm64/bin/go-ngc-darwin-arm64 ./cmd/ngtsc
GOOS=darwin GOARCH=arm64 go build -buildvcs=false -o ../angular-go-darwin-arm64/bin/go-localize-darwin-arm64 ./cmd/localize/main.go
GOOS=darwin GOARCH=arm64 go build -buildvcs=false -o ../angular-go-darwin-arm64/bin/go-localize ./cmd/localize/main.go

# Mac x64
GOOS=darwin GOARCH=amd64 go build -buildvcs=false -o ../angular-go-darwin-x64/bin/go-ngc ./cmd/ngtsc
GOOS=darwin GOARCH=amd64 go build -buildvcs=false -o ../angular-go-darwin-x64/bin/go-ngc-darwin-x64 ./cmd/ngtsc
GOOS=darwin GOARCH=amd64 go build -buildvcs=false -o ../angular-go-darwin-x64/bin/go-localize-darwin-x64 ./cmd/localize/main.go
GOOS=darwin GOARCH=amd64 go build -buildvcs=false -o ../angular-go-darwin-x64/bin/go-localize ./cmd/localize/main.go

# Linux x64
GOOS=linux GOARCH=amd64 go build -buildvcs=false -o ../angular-go-linux-x64/bin/go-ngc ./cmd/ngtsc
GOOS=linux GOARCH=amd64 go build -buildvcs=false -o ../angular-go-linux-x64/bin/go-ngc-linux-x64 ./cmd/ngtsc
GOOS=linux GOARCH=amd64 go build -buildvcs=false -o ../angular-go-linux-x64/bin/go-localize-linux-x64 ./cmd/localize/main.go
GOOS=linux GOARCH=amd64 go build -buildvcs=false -o ../angular-go-linux-x64/bin/go-localize ./cmd/localize/main.go

# Win32 x64
GOOS=windows GOARCH=amd64 go build -buildvcs=false -o ../angular-go-win32-x64/bin/go-ngc.exe ./cmd/ngtsc
GOOS=windows GOARCH=amd64 go build -buildvcs=false -o ../angular-go-win32-x64/bin/go-ngc-win32-x64.exe ./cmd/ngtsc
GOOS=windows GOARCH=amd64 go build -buildvcs=false -o ../angular-go-win32-x64/bin/go-localize-win32-x64.exe ./cmd/localize/main.go
GOOS=windows GOARCH=amd64 go build -buildvcs=false -o ../angular-go-win32-x64/bin/go-localize.exe ./cmd/localize/main.go

echo "Done building all native binaries!"
