import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';
import { execSync } from 'child_process';
import { performance } from 'perf_hooks';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const ROOT = path.resolve(__dirname, '../../../../');
const LARGE_APP_DIR = path.join(ROOT, '.tmp', 'bench', 'large_app');

const NUM_COMPONENTS = Number.parseInt(process.env.NG_BENCH_COMPONENTS || '2000', 10);

console.log(`Generating ${NUM_COMPONENTS} components in ${LARGE_APP_DIR}...`);

if (fs.existsSync(LARGE_APP_DIR)) {
  fs.rmSync(LARGE_APP_DIR, { recursive: true, force: true });
}
fs.mkdirSync(LARGE_APP_DIR, { recursive: true });

// Create components
let appImports = [];
let appDeps = [];
let appHtml = [];

for (let i = 0; i < NUM_COMPONENTS; i++) {
  const compName = `Comp${i}`;
  const fileName = `comp_${i}.ts`;

  const content = `
import { Component, Input } from '@angular/core';

@Component({
  selector: 'app-comp-${i}',
  standalone: true,
  template: \`
    <div class="comp-box">
      <h2>Component ${i}</h2>
      <p>This is a generated component with input {{ data }}</p>
      <button (click)="onClick()">Click Me</button>
    </div>
  \`,
  styles: [\`
    .comp-box { border: 1px solid #ccc; padding: 10px; margin: 5px; }
    h2 { color: blue; }
  \`]
})
export class ${compName} {
  @Input() data = '';
  onClick() {
    console.log('Clicked ${i}');
  }
}
`;

  fs.writeFileSync(path.join(LARGE_APP_DIR, fileName), content);

  appImports.push(`import { ${compName} } from './${fileName.replace('.ts', '')}';`);
  appDeps.push(compName);
  appHtml.push(`<app-comp-${i} [data]="'Test ${i}'"></app-comp-${i}>`);
}

// Create app.component.ts to import them all
const appCompContent = `
import { Component } from '@angular/core';
${appImports.join('\n')}

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [${appDeps.join(', ')}],
  template: \`
    <h1>Large App (${NUM_COMPONENTS} components)</h1>
    <div class="grid">
      ${appHtml.join('\n      ')}
    </div>
  \`
})
export class AppComponent {
}
`;
fs.writeFileSync(path.join(LARGE_APP_DIR, 'app.component.ts'), appCompContent);

// Create dummy module and main just in case
fs.writeFileSync(path.join(LARGE_APP_DIR, 'main.ts'), `
import { bootstrapApplication } from '@angular/platform-browser';
import { AppComponent } from './app.component';

bootstrapApplication(AppComponent).catch(err => console.error(err));
`);

// Create tsconfig.json
const tsconfig = {
  "compileOnSave": false,
  "compilerOptions": {
    "outDir": "./dist/out-tsc",
    "strict": true,
    "noImplicitOverride": true,
    "noPropertyAccessFromIndexSignature": true,
    "noImplicitReturns": true,
    "noFallthroughCasesInSwitch": true,
    "skipLibCheck": true,
    "isolatedModules": true,
    "esModuleInterop": true,
    "experimentalDecorators": true,
    "moduleResolution": "bundler",
    "importHelpers": true,
    "target": "ES2022",
    "module": "ES2022",
    "types": []
  },
  "angularCompilerOptions": {
    "enableI18nLegacyMessageIdFormat": false,
    "strictInjectionParameters": true,
    "strictInputAccessModifiers": true,
    "strictTemplates": true,
    "compilationMode": "full"
  },
  "include": [
    "**/*.ts"
  ]
};
fs.writeFileSync(path.join(LARGE_APP_DIR, 'tsconfig.json'), JSON.stringify(tsconfig, null, 2));

try {
  fs.symlinkSync(path.join(ROOT, 'angular-packages', 'new-demo-app', 'node_modules'), path.join(LARGE_APP_DIR, 'node_modules'), 'dir');
} catch (e) {
  // might exist
}

console.log('Done generating files.');

// Run benchmark
const RUNS = 1; // Since it's huge, 1 run is enough

function rm(pathname) {
  fs.rmSync(pathname, { recursive: true, force: true });
}

const runCmd = (name, cmd) => {
  console.log(`Running benchmark: ${name}`);
  rm(path.join(LARGE_APP_DIR, 'dist'));
  const start = performance.now();
  execSync(cmd, { cwd: LARGE_APP_DIR, stdio: 'inherit' });
  const end = performance.now();
  const time = end - start;
  console.log(`  Time: ${time.toFixed(2)}ms`);
  return time;
};

const goGlobalTime = runCmd('Cold Compile: Go-NGC global (Large App)', `${path.join(ROOT, 'go-ngc')} -p tsconfig.json`);
const goLocalTime = runCmd('Cold Compile: Go-NGC local (Large App)', `${path.join(ROOT, 'go-ngc')} -p tsconfig.json --compilationMode=local`);
const ngtscTime = runCmd('Cold Compile: NGTSC (Large App)', `node ${path.join(ROOT, 'angular-packages/new-demo-app/node_modules/@angular/compiler-cli/bundles/src/bin/ngc.js')} -p tsconfig.json`);

console.log(`\n=== LARGE APP SUMMARY (${NUM_COMPONENTS} components) ===`);
console.log(`Cold Compile Go global: ${goGlobalTime.toFixed(2)}ms`);
console.log(`Cold Compile Go local:  ${goLocalTime.toFixed(2)}ms`);
console.log(`Cold Compile ngtsc:     ${ngtscTime.toFixed(2)}ms`);
console.log(`Global speedup:         ${(ngtscTime / goGlobalTime).toFixed(2)}x faster`);
console.log(`Local speedup:          ${(ngtscTime / goLocalTime).toFixed(2)}x faster`);
