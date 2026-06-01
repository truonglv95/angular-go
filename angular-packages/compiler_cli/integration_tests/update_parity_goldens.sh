#!/bin/bash
set -e

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
TESTDATA="$DIR/testdata/render3_ir"
NGC="$DIR/../../new-demo-app/node_modules/.bin/ngc"
ROOT_NODE_MODULES="$DIR/../../new-demo-app/node_modules"

if [ ! -f "$NGC" ]; then
    echo "ngc not found at $NGC"
    exit 1
fi

for CASE_DIR in "$TESTDATA"/*; do
    if [ -d "$CASE_DIR/project" ]; then
        echo "Compiling $CASE_DIR with ngtsc..."
        cd "$CASE_DIR/project"
        
        # Symlink node_modules
        rm -rf node_modules
        ln -s "$ROOT_NODE_MODULES" node_modules
        
        "$NGC" -p tsconfig.json || true
        
        # Move output to golden
        rm -rf "$CASE_DIR/golden/out"
        cp -r out "$CASE_DIR/golden/out"
        echo "Updated goldens for $(basename "$CASE_DIR")"
    fi
done
