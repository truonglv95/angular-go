#!/bin/bash
set -e
NAME=$1
DIR="testdata/render3_ir/$NAME"
mkdir -p "$DIR/project/app"
mkdir -p "$DIR/golden/app"

cat << 'JSON' > "$DIR/project/tsconfig.json"
{
  "compilerOptions": {
    "experimentalDecorators": true,
    "module": "preserve",
    "noEmitOnError": false,
    "outDir": "./out",
    "rootDir": ".",
    "skipLibCheck": true,
    "strict": true,
    "target": "ES2022",
    "types": []
  },
  "include": [
    "app/**/*.ts"
  ]
}
JSON
echo "Created $NAME"
