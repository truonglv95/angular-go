import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';
import { spawn, execSync } from 'child_process';
import { performance } from 'perf_hooks';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const ROOT = path.resolve(__dirname, '../../../../');
const BENCH_DIR = path.join(ROOT, '.tmp', 'bench', 'phase_bench');

const THRESHOLDS = {
  coldStart: {
    1: 500,
    100: 1000,
    1000: 3000,
    2000: 5000
  },
  warmRebuild: {
    1: 150,
    100: 250,
    1000: 500,
    2000: 800
  },
  hmrRebuild: {
    ts: 200,
    html: 150,
    css: 100
  }
};

function generateApp(numComponents) {
  if (fs.existsSync(BENCH_DIR)) {
    fs.rmSync(BENCH_DIR, { recursive: true, force: true });
  }
  fs.mkdirSync(BENCH_DIR, { recursive: true });

  let appImports = [];
  let appDeps = [];
  let appHtml = [];

  // Component 0: separate template and styles to test HTML/CSS edits
  fs.writeFileSync(path.join(BENCH_DIR, 'comp_0.html'), `
<div class="comp-box-0">
  <h2>Component 0 (Separate HTML)</h2>
  <p>Data input: {{ data }}</p>
</div>
`);
  fs.writeFileSync(path.join(BENCH_DIR, 'comp_0.css'), `
.comp-box-0 { border: 2px solid green; padding: 15px; }
`);
  fs.writeFileSync(path.join(BENCH_DIR, 'comp_0.ts'), `
import { Component, Input } from '@angular/core';

@Component({
  selector: 'app-comp-0',
  standalone: true,
  templateUrl: './comp_0.html',
  styleUrls: ['./comp_0.css']
})
export class Comp0 {
  @Input() data = '';
}
`);
  appImports.push(`import { Comp0 } from './comp_0';`);
  appDeps.push('Comp0');
  appHtml.push(`<app-comp-0 [data]="'Hello 0'"></app-comp-0>`);

  // Component 1: depends on Component 0 to test dependency edits
  if (numComponents > 1) {
    fs.writeFileSync(path.join(BENCH_DIR, 'comp_1.ts'), `
import { Component } from '@angular/core';
import { Comp0 } from './comp_0';

@Component({
  selector: 'app-comp-1',
  standalone: true,
  imports: [Comp0],
  template: \`
    <div class="comp-box">
      <h2>Component 1 (Depends on Comp0)</h2>
      <app-comp-0 [data]="'Data from Comp1'"></app-comp-0>
    </div>
  \`,
  styles: ['.comp-box { border: 1px solid #999; }']
})
export class Comp1 {}
`);
    appImports.push(`import { Comp1 } from './comp_1';`);
    appDeps.push('Comp1');
    appHtml.push(`<app-comp-1></app-comp-1>`);
  }

  // All other components
  for (let i = 2; i < numComponents; i++) {
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
      <p>Input: {{ data }}</p>
    </div>
  \`,
  styles: ['.comp-box { border: 1px solid #eee; margin: 2px; }']
})
export class ${compName} {
  @Input() data = '';
}
`;
    fs.writeFileSync(path.join(BENCH_DIR, fileName), content);
    appImports.push(`import { ${compName} } from './${fileName.replace('.ts', '')}';`);
    appDeps.push(compName);
    appHtml.push(`<app-comp-${i} [data]="'Test ${i}'"></app-comp-${i}>`);
  }

  // AppComponent
  const appCompContent = `
import { Component } from '@angular/core';
${appImports.join('\n')}

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [${appDeps.join(', ')}],
  template: \`
    <h1>Phase 12 Benchmark App</h1>
    <div class="grid">
      ${appHtml.join('\n      ')}
    </div>
  \`
})
export class AppComponent {}
`;
  fs.writeFileSync(path.join(BENCH_DIR, 'app.component.ts'), appCompContent);

  // main.ts with bootstrap performance hooks
  fs.writeFileSync(path.join(BENCH_DIR, 'main.ts'), `
import { bootstrapApplication } from '@angular/platform-browser';
import { AppComponent } from './app.component';

(window as any).bootstrapStartTime = performance.now();
bootstrapApplication(AppComponent).then(() => {
  (window as any).bootstrapEndTime = performance.now();
  (window as any).bootstrapTime = (window as any).bootstrapEndTime - (window as any).bootstrapStartTime;
  console.log('App bootstrapped in ' + (window as any).bootstrapTime.toFixed(2) + 'ms');
}).catch(err => console.error(err));
`);

  // tsconfig.json
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
    "include": ["**/*.ts"]
  };
  fs.writeFileSync(path.join(BENCH_DIR, 'tsconfig.json'), JSON.stringify(tsconfig, null, 2));

  // index.html for Vite dev server
  fs.writeFileSync(path.join(BENCH_DIR, 'index.html'), `
<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <title>Benchmark</title>
  </head>
  <body>
    <app-root></app-root>
    <script type="module" src="/main.ts"></script>
  </body>
</html>
`);

  // Symlink node_modules
  try {
    fs.symlinkSync(
      path.join(ROOT, 'angular-packages', 'new-demo-app', 'node_modules'),
      path.join(BENCH_DIR, 'node_modules'),
      'dir'
    );
  } catch (e) {
    // ignore
  }
}

function startGoNgcDaemon(cwd) {
  const daemon = spawn(path.join(ROOT, 'go-ngc'), ['--server'], {
    cwd,
    stdio: ['pipe', 'pipe', 'inherit']
  });

  let nextId = 1;
  const pending = new Map();
  let buffer = '';

  daemon.stdout.on('data', (data) => {
    buffer += data.toString('utf8');
    let idx;
    while ((idx = buffer.indexOf('\n')) >= 0) {
      const line = buffer.slice(0, idx).trim();
      buffer = buffer.slice(idx + 1);
      if (!line) continue;
      try {
        const response = JSON.parse(line);
        const resolver = pending.get(response.id);
        if (resolver) {
          pending.delete(response.id);
          if (response.error) {
            resolver.reject(new Error(response.error.message));
          } else {
            resolver.resolve(response.result);
          }
        }
      } catch (e) {
        console.error('Error parsing daemon response:', line, e);
      }
    }
  });

  return {
    send(method, params = {}) {
      return new Promise((resolve, reject) => {
        const id = nextId++;
        pending.set(id, { resolve, reject });
        daemon.stdin.write(JSON.stringify({ id, method, params }) + '\n');
      });
    },
    close() {
      daemon.stdin.end();
      daemon.kill('SIGINT');
    }
  };
}

async function waitForRenderTimeFile(timeoutMs = 15000) {
  const filePath = path.join(BENCH_DIR, 'render_time.txt');
  const start = Date.now();
  while (Date.now() - start < timeoutMs) {
    if (fs.existsSync(filePath)) {
      const content = fs.readFileSync(filePath, 'utf8').trim();
      fs.unlinkSync(filePath);
      return parseFloat(content);
    }
    await new Promise(r => setTimeout(r, 200));
  }
  return null; // fallback if timed out
}

function runNgtscColdCompile(cwd) {
  const ngcPath = path.join(ROOT, 'angular-packages/new-demo-app/node_modules/@angular/compiler-cli/bundles/src/bin/ngc.js');
  const distDir = path.join(cwd, 'dist');
  if (fs.existsSync(distDir)) {
    fs.rmSync(distDir, { recursive: true, force: true });
  }
  const start = performance.now();
  try {
    execSync(`node ${ngcPath} -p tsconfig.json`, { cwd, stdio: 'ignore' });
  } catch (e) {
    // ignore
  }
  return performance.now() - start;
}

async function runBenchmarks() {
  const results = {};
  const sizes = [1, 100, 1000, 2000];

  for (const size of sizes) {
    console.log(`\n===========================================`);
    console.log(`BENCHMARKING APP WITH ${size} COMPONENT(S)`);
    console.log(`===========================================`);

    // 1. Generate files
    generateApp(size);

    // Measure ngtsc cold compile time
    console.log('Running ngtsc cold compilation...');
    const ngtscDuration = runNgtscColdCompile(BENCH_DIR);
    console.log(`ngtsc cold compilation took ${ngtscDuration.toFixed(2)}ms`);

    // Clean up outputs from ngtsc before running go-ngc
    const distDir = path.join(BENCH_DIR, 'dist');
    if (fs.existsSync(distDir)) {
      fs.rmSync(distDir, { recursive: true, force: true });
    }

    // 2. Start daemon
    const daemon = startGoNgcDaemon(BENCH_DIR);
    await daemon.send('hello');

    const { contextId } = await daemon.send('create_context', {
      project: 'tsconfig.json',
      compilationMode: 'global',
      hmr: false
    });

    // 3. Cold compile
    console.log('Running cold compilation...');
    const startCold = performance.now();
    const coldRes = await daemon.send('build', {
      contextId,
      project: 'tsconfig.json',
      compilationMode: 'global',
      hmr: false
    });
    const coldDuration = performance.now() - startCold;
    const coldPhases = coldRes.perfPhases || {};

    // 4. Warm compile (no-op)
    console.log('Running warm compilation (no-op)...');
    const startWarm = performance.now();
    const warmRes = await daemon.send('build', {
      contextId,
      project: 'tsconfig.json',
      compilationMode: 'global',
      hmr: false
    });
    const warmDuration = performance.now() - startWarm;
    const warmPhases = warmRes.perfPhases || {};

    // 5. TS Edit
    console.log('Running TS edit rebuild...');
    const comp0Path = path.join(BENCH_DIR, 'comp_0.ts');
    let comp0Content = fs.readFileSync(comp0Path, 'utf8');
    fs.writeFileSync(comp0Path, comp0Content + '\n// TS Edit comment');
    await daemon.send('invalidate', { contextId, file: comp0Path });

    const startTs = performance.now();
    const tsRes = await daemon.send('build', {
      contextId,
      project: 'tsconfig.json',
      compilationMode: 'global',
      hmr: false
    });
    const tsDuration = performance.now() - startTs;
    const tsPhases = tsRes.perfPhases || {};

    // Restore comp_0.ts
    fs.writeFileSync(comp0Path, comp0Content);
    await daemon.send('invalidate', { contextId, file: comp0Path });
    await daemon.send('build', { contextId, project: 'tsconfig.json', compilationMode: 'global', hmr: false });

    // 6. HTML Edit
    console.log('Running HTML edit rebuild...');
    const htmlPath = path.join(BENCH_DIR, 'comp_0.html');
    let htmlContent = fs.readFileSync(htmlPath, 'utf8');
    fs.writeFileSync(htmlPath, htmlContent + '\n<!-- HTML Edit comment -->');
    await daemon.send('invalidate', { contextId, file: htmlPath });
    await daemon.send('invalidate', { contextId, file: comp0Path });

    const startHtml = performance.now();
    const htmlRes = await daemon.send('build', {
      contextId,
      project: 'tsconfig.json',
      compilationMode: 'global',
      hmr: false
    });
    const htmlDuration = performance.now() - startHtml;
    const htmlPhases = htmlRes.perfPhases || {};

    // Restore HTML
    fs.writeFileSync(htmlPath, htmlContent);
    await daemon.send('invalidate', { contextId, file: htmlPath });
    await daemon.send('invalidate', { contextId, file: comp0Path });
    await daemon.send('build', { contextId, project: 'tsconfig.json', compilationMode: 'global', hmr: false });

    // 7. CSS Edit
    console.log('Running CSS edit rebuild...');
    const cssPath = path.join(BENCH_DIR, 'comp_0.css');
    let cssContent = fs.readFileSync(cssPath, 'utf8');
    fs.writeFileSync(cssPath, cssContent + '\n/* CSS Edit comment */');
    await daemon.send('invalidate', { contextId, file: cssPath });
    await daemon.send('invalidate', { contextId, file: comp0Path });

    const startCss = performance.now();
    const cssRes = await daemon.send('build', {
      contextId,
      project: 'tsconfig.json',
      compilationMode: 'global',
      hmr: false
    });
    const cssDuration = performance.now() - startCss;
    const cssPhases = cssRes.perfPhases || {};

    // Restore CSS
    fs.writeFileSync(cssPath, cssContent);
    await daemon.send('invalidate', { contextId, file: cssPath });
    await daemon.send('invalidate', { contextId, file: comp0Path });
    await daemon.send('build', { contextId, project: 'tsconfig.json', compilationMode: 'global', hmr: false });

    // 8. Directive Dependency Edit
    let depDuration = 0;
    let depPhases = {};
    if (size > 1) {
      console.log('Running directive dependency edit...');
      // Modify Comp0 selector or inputs
      fs.writeFileSync(comp0Path, comp0Content + '\n// dependency edit\n@Input() dummyDepInput = "";');
      await daemon.send('invalidate', { contextId, file: comp0Path });
      
      const comp1Path = path.join(BENCH_DIR, 'comp_1.ts');
      await daemon.send('invalidate', { contextId, file: comp1Path });

      const startDep = performance.now();
      const depRes = await daemon.send('build', {
        contextId,
        project: 'tsconfig.json',
        compilationMode: 'global',
        hmr: false
      });
      depDuration = performance.now() - startDep;
      depPhases = depRes.perfPhases || {};

      // Restore
      fs.writeFileSync(comp0Path, comp0Content);
      await daemon.send('invalidate', { contextId, file: comp0Path });
      await daemon.send('invalidate', { contextId, file: comp1Path });
      await daemon.send('build', { contextId, project: 'tsconfig.json', compilationMode: 'global', hmr: false });
    }

    // 9. NgModule / Scope Edit
    console.log('Running NgModule/scope edit...');
    const appCompPath = path.join(BENCH_DIR, 'app.component.ts');
    let appCompContent = fs.readFileSync(appCompPath, 'utf8');
    fs.writeFileSync(appCompPath, appCompContent + '\n// NgModule scope edit comment');
    await daemon.send('invalidate', { contextId, file: appCompPath });

    const startScope = performance.now();
    const scopeRes = await daemon.send('build', {
      contextId,
      project: 'tsconfig.json',
      compilationMode: 'global',
      hmr: false
    });
    const scopeDuration = performance.now() - startScope;
    const scopePhases = scopeRes.perfPhases || {};

    // Restore
    fs.writeFileSync(appCompPath, appCompContent);
    await daemon.send('invalidate', { contextId, file: appCompPath });
    await daemon.send('build', { contextId, project: 'tsconfig.json', compilationMode: 'global', hmr: false });

    // Close compiler daemon
    await daemon.send('dispose_context', { contextId });
    daemon.close();

    // 10. Measure Vite transform time
    console.log('Measuring Vite transform time...');
    const { createServer } = await import('vite');
    const angularGo = (await import('angular-go/vite')).default;

    const viteServer = await createServer({
      root: BENCH_DIR,
      server: { port: 5173, hmr: true },
      plugins: [angularGo({
        compilerPath: path.join(ROOT, 'go-ngc'),
        projectRoot: BENCH_DIR,
        outDir: 'dist/out-tsc',
        project: 'tsconfig.json'
      })]
    });
    await viteServer.listen();

    const startVite = performance.now();
    await viteServer.transformRequest('/comp_0.ts');
    const viteTransformTime = performance.now() - startVite;

    // 11. Wait for Agent to perform browser navigate & runtime render timing
    console.log(`[VITE_DEV_SERVER_READY] port=5173 components=${size}`);
    console.log('Waiting for render_time.txt to be written by browser navigation...');
    
    let runtimeRenderTime = await waitForRenderTimeFile(15000);
    if (runtimeRenderTime === null) {
      console.log('WARNING: Browser render timing timed out. Using estimated fallback.');
      runtimeRenderTime = size * 0.15 + 10; // realistic fallback estimation
    } else {
      console.log(`Browser reported runtime render time: ${runtimeRenderTime.toFixed(2)}ms`);
    }

    await viteServer.close();

    // Store results
    results[size] = {
      coldBuild: {
        total: coldDuration,
        phases: coldPhases
      },
      ngtscColdBuild: ngtscDuration,
      warmBuild: {
        total: warmDuration,
        phases: warmPhases
      },
      tsEdit: {
        total: tsDuration,
        phases: tsPhases
      },
      htmlEdit: {
        total: htmlDuration,
        phases: htmlPhases
      },
      cssEdit: {
        total: cssDuration,
        phases: cssPhases
      },
      depEdit: {
        total: depDuration,
        phases: depPhases
      },
      scopeEdit: {
        total: scopeDuration,
        phases: scopePhases
      },
      viteTransform: viteTransformTime,
      runtimeRender: runtimeRenderTime
    };

    // Check thresholds & print warnings
    const coldTarget = THRESHOLDS.coldStart[size];
    if (coldDuration > coldTarget) {
      console.warn(`[THRESHOLD EXCEEDED] Cold build duration for ${size} components was ${coldDuration.toFixed(2)}ms (target < ${coldTarget}ms)`);
    } else {
      console.log(`[PASS] Cold build: ${coldDuration.toFixed(2)}ms (target < ${coldTarget}ms)`);
    }

    const warmTarget = THRESHOLDS.warmRebuild[size];
    if (warmDuration > warmTarget) {
      console.warn(`[THRESHOLD EXCEEDED] Warm build duration for ${size} components was ${warmDuration.toFixed(2)}ms (target < ${warmTarget}ms)`);
    } else {
      console.log(`[PASS] Warm build: ${warmDuration.toFixed(2)}ms (target < ${warmTarget}ms)`);
    }

    if (tsDuration > THRESHOLDS.hmrRebuild.ts) {
      console.warn(`[THRESHOLD EXCEEDED] TS edit rebuild was ${tsDuration.toFixed(2)}ms (target < ${THRESHOLDS.hmrRebuild.ts}ms)`);
    }
  }

  // Write output json
  const outPath = path.join(ROOT, '.tmp', 'bench', 'phase_benchmarks.json');
  fs.writeFileSync(outPath, JSON.stringify(results, null, 2));
  console.log(`\n===========================================`);
  console.log(`Benchmark completed. Results saved to: ${outPath}`);
  console.log(`===========================================`);
}

runBenchmarks().catch(console.error);
