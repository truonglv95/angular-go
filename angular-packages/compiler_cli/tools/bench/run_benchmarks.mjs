import { execSync } from 'child_process';
import { performance } from 'perf_hooks';
import * as fs from 'fs';
import * as path from 'path';
import { fileURLToPath } from 'url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const ROOT = path.resolve(__dirname, '../../../../');
const OUT_DIR = path.join(ROOT, '.tmp', 'bench');

// Ensure output directory exists
if (!fs.existsSync(OUT_DIR)) {
  fs.mkdirSync(OUT_DIR, { recursive: true });
}

const RUNS = 3;

function rm(pathname) {
  fs.rmSync(pathname, { recursive: true, force: true });
}

function runCmd(name, cmd, cwd = ROOT, prepare = () => {}) {
  const times = [];
  console.log(`Running benchmark: ${name} (${RUNS} runs)`);

  for (let i = 0; i < RUNS; i++) {
    prepare();

    const start = performance.now();
    execSync(cmd, { cwd, stdio: 'ignore' });
    const end = performance.now();
    times.push(end - start);
    console.log(`  Run ${i + 1}: ${(end - start).toFixed(2)}ms`);
  }

  const avg = times.reduce((a, b) => a + b, 0) / times.length;
  console.log(`  Average: ${avg.toFixed(2)}ms\n`);
  return { times, avg };
}

async function runAll() {
  const report = {
    timestamp: new Date().toISOString(),
    benchmarks: {}
  };

  const projectDir = path.join(ROOT, 'angular-packages', 'new-demo-app');
  const cleanAngularCompileOutputs = () => {
    rm(path.join(projectDir, 'out-tsc'));
  };
  const cleanViteOutputs = () => {
    rm(path.join(projectDir, '.angular/cache'));
    rm(path.join(projectDir, 'node_modules/.vite'));
    rm(path.join(ROOT, 'angular-packages', 'dist'));
    cleanAngularCompileOutputs();
  };

  // Cold Compile: Go-NGC
  report.benchmarks.coldCompileGo = runCmd(
    'Cold Compile: Go-NGC',
    `${path.join(ROOT, 'go-ngc')} -p tsconfig.app.json`,
    projectDir,
    cleanAngularCompileOutputs
  );

  // Cold Compile: NGTSC
  report.benchmarks.coldCompileNgtsc = runCmd(
    'Cold Compile: NGTSC',
    `node node_modules/@angular/compiler-cli/bundles/src/bin/ngc.js -p tsconfig.app.json`,
    projectDir,
    cleanAngularCompileOutputs
  );

  // Vite Production Build: Go-NGC
  report.benchmarks.viteProdGo = runCmd(
    'Vite Production Build: Go-NGC',
    `npx vite build --config vite.config.ts`,
    projectDir,
    cleanViteOutputs
  );

  // Linker Throughput: Go-NGC
  report.benchmarks.linkerGo = runCmd(
    'Linker: Go-NGC (primeng-button)',
    `${path.join(ROOT, 'go-ngc')} --link node_modules/primeng/fesm2022/primeng-button.mjs > /dev/null`,
    projectDir
  );

  // Linker Throughput: ngtsc (using babel transform directly)
  const benchLinkerJs = path.join(OUT_DIR, 'bench_linker_ngtsc.mjs');
  fs.writeFileSync(benchLinkerJs, `
import fs from 'node:fs';
import path from 'node:path';
import { createRequire } from 'node:module';
import { pathToFileURL } from 'node:url';

const requireFromCwd = createRequire(path.join(process.cwd(), 'package.json'));
const babel = requireFromCwd('@babel/core');
const linkerBabelPath = requireFromCwd.resolve('@angular/compiler-cli/linker/babel');
const { createEs2015LinkerPlugin } = await import(pathToFileURL(linkerBabelPath));

const inputPath = process.argv[2];
const absoluteInputPath = path.resolve(inputPath);
const code = fs.readFileSync(absoluteInputPath, 'utf8');

const plugin = createEs2015LinkerPlugin({
  fileSystem: {
    resolve: (filePath) => filePath,
    exists: (filePath) => fs.existsSync(filePath),
    dirname: (filePath) => path.dirname(filePath),
    relative: (from, to) => path.relative(from, to),
    readFile: (filePath) => fs.readFileSync(filePath, 'utf8'),
  },
  logger: { level: 0, debug() {}, info() {}, warn() {}, error() {} },
  options: { linkerJitMode: false },
});

const expectedAst = babel.parse(code, { filename: absoluteInputPath, parserOpts: { plugins: ['typescript'] } });
babel.transformFromAstSync(expectedAst, code, {
  filename: absoluteInputPath,
  plugins: [plugin],
  ast: false,
  code: true,
});
`);

  report.benchmarks.linkerNgtsc = runCmd(
    'Linker: NGTSC (primeng-button)',
    `node ${benchLinkerJs} node_modules/primeng/fesm2022/primeng-button.mjs`,
    projectDir
  );

  // Vite Dev Startup: Go-NGC
  async function runViteDevCmd(name, cmd, cwd) {
    console.log(`Running benchmark: ${name} (${RUNS} runs)`);
    const { spawn } = await import('child_process');
    const times = [];

    for (let i = 0; i < RUNS; i++) {
      cleanViteOutputs();

      const start = performance.now();
      await new Promise((resolve) => {
        const child = spawn('npx', ['vite', '--config', 'vite.config.ts', '--force'], { cwd, shell: true });

        let done = false;
        child.stdout.on('data', (data) => {
          const str = data.toString();
          const match = str.match(/ready in\s+([\d.]+)\s*ms/);
          if (match && !done) {
            done = true;
            times.push(parseFloat(match[1]));
            console.log(`  Run ${i + 1}: ${parseFloat(match[1]).toFixed(2)}ms`);
            child.kill('SIGINT');
            resolve();
          }
        });

        child.on('error', () => { if (!done) resolve(); });
        child.on('exit', (code) => {
          if (!done && code !== 0) {
            done = true;
            resolve();
          }
        });

        setTimeout(() => {
          if (!done) {
            done = true;
            child.kill('SIGINT');
            times.push(performance.now() - start);
            resolve();
          }
        }, 10000);
      });
      if (times.length <= i) {
        throw new Error(`${name} did not report ready time`);
      }
    }
    const avg = times.reduce((a, b) => a + b, 0) / times.length;
    console.log(`  Average: ${avg.toFixed(2)}ms\n`);
    return { times, avg };
  }

  report.benchmarks.viteDevGo = await runViteDevCmd(
    'Vite Dev Startup: Go-NGC',
    `npx vite --config vite.config.ts --force`,
    projectDir
  );

  const reportPath = path.join(OUT_DIR, 'report.json');
  fs.writeFileSync(reportPath, JSON.stringify(report, null, 2));
  console.log(`Benchmark report saved to ${reportPath}`);

  // Print summary comparison
  const goAvg = report.benchmarks.coldCompileGo.avg;
  const ngtscAvg = report.benchmarks.coldCompileNgtsc.avg;
  const speedup = ngtscAvg / goAvg;
  console.log(`=== Summary ===`);
  console.log(`Cold Compile Go:    ${goAvg.toFixed(2)}ms`);
  console.log(`Cold Compile ngtsc: ${ngtscAvg.toFixed(2)}ms`);
  console.log(`Speedup:            ${speedup.toFixed(2)}x faster\n`);

  const linkerGoAvg = report.benchmarks.linkerGo.avg;
  const linkerNgtscAvg = report.benchmarks.linkerNgtsc.avg;
  const linkerSpeedup = linkerNgtscAvg / linkerGoAvg;
  console.log(`Linker Go:          ${linkerGoAvg.toFixed(2)}ms`);
  console.log(`Linker ngtsc:       ${linkerNgtscAvg.toFixed(2)}ms`);
  console.log(`Linker Speedup:     ${linkerSpeedup.toFixed(2)}x faster\n`);

  console.log(`Vite Dev Startup Go: ${report.benchmarks.viteDevGo.avg.toFixed(2)}ms`);
  console.log(`Vite Prod Build Go:  ${report.benchmarks.viteProdGo.avg.toFixed(2)}ms`);
}

runAll();
