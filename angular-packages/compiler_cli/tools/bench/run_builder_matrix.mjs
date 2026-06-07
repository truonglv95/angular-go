import { spawn, spawnSync } from 'node:child_process';
import fs from 'node:fs';
import path from 'node:path';
import { performance } from 'node:perf_hooks';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const ROOT = path.resolve(__dirname, '../../../../');
const DEFAULT_PROJECT = path.join(ROOT, 'angular-packages', 'new-demo-app');
const projectDir = path.resolve(process.env.GO_NGC_BENCH_PROJECT || DEFAULT_PROJECT);
const project = process.env.GO_NGC_BENCH_TSCONFIG || 'tsconfig.app.json';
const runs = Number.parseInt(process.env.GO_NGC_BENCH_RUNS || '3', 10);
const goNgc = process.env.GO_NGC || path.join(ROOT, 'go-ngc');
const outDir = process.env.GO_NGC_BENCH_OUT_DIR || path.join(projectDir, 'out-tsc');
const preserveImports = process.env.GO_NGC_BENCH_PRESERVE_IMPORTS !== '0';
const reportDir = path.join(ROOT, '.tmp', 'bench');

fs.mkdirSync(reportDir, { recursive: true });

function rm(pathname) {
  fs.rmSync(pathname, { recursive: true, force: true });
}

function runProcess(args, options = {}) {
  const started = performance.now();
  const result = spawnSync(goNgc, args, {
    cwd: projectDir,
    encoding: 'utf8',
    maxBuffer: 1024 * 1024 * 512,
    ...options,
  });
  const durationMs = performance.now() - started;
  if (result.status !== 0) {
    throw new Error(`go-ngc ${args.join(' ')} failed\n${result.stderr || result.stdout}`);
  }
  return { durationMs, stdout: result.stdout };
}

function compileArgs(extraArgs = []) {
  const args = ['-p', project, ...extraArgs];
  if (preserveImports && !args.includes('--preserve-imports')) {
    args.push('--preserve-imports');
  }
  return args;
}

function summarize(samples) {
  const sorted = [...samples].sort((a, b) => a - b);
  const sum = sorted.reduce((acc, value) => acc + value, 0);
  return {
    samplesMs: samples.map((value) => Number(value.toFixed(2))),
    avgMs: Number((sum / sorted.length).toFixed(2)),
    medianMs: Number(sorted[Math.floor(sorted.length / 2)].toFixed(2)),
    minMs: Number(sorted[0].toFixed(2)),
    maxMs: Number(sorted[sorted.length - 1].toFixed(2)),
  };
}

async function measure(name, fn, prepare = () => {}) {
  const samples = [];
  for (let i = 0; i < runs; i++) {
    prepare();
    const duration = await fn();
    samples.push(duration);
    console.log(`${name} run ${i + 1}: ${duration.toFixed(2)}ms`);
  }
  const summary = summarize(samples);
  console.log(`${name} median: ${summary.medianMs.toFixed(2)}ms\n`);
  return summary;
}

function startDaemon() {
  const proc = spawn(goNgc, ['--server'], {
    cwd: projectDir,
    stdio: ['pipe', 'pipe', 'pipe'],
  });
  let nextId = 1;
  let buffer = '';
  const pending = new Map();

  proc.stdout.setEncoding('utf8');
  proc.stdout.on('data', (chunk) => {
    buffer += chunk;
    let idx;
    while ((idx = buffer.indexOf('\n')) >= 0) {
      const line = buffer.slice(0, idx);
      buffer = buffer.slice(idx + 1);
      if (!line.trim()) continue;
      const message = JSON.parse(line);
      const item = pending.get(message.id);
      if (!item) continue;
      pending.delete(message.id);
      if (message.error) {
        item.reject(new Error(`${message.error.code}: ${message.error.message}`));
      } else {
        item.resolve(message.result);
      }
    }
  });

  proc.stderr.setEncoding('utf8');
  proc.stderr.on('data', (chunk) => {
    if (chunk.trim()) process.stderr.write(`[go-ngc] ${chunk}`);
  });

  function request(method, params = {}) {
    return new Promise((resolve, reject) => {
      const id = nextId++;
      pending.set(id, { resolve, reject });
      proc.stdin.write(`${JSON.stringify({ id, method, params })}\n`, (err) => {
        if (err) {
          pending.delete(id);
          reject(err);
        }
      });
    });
  }

  async function close() {
    try {
      await request('shutdown');
    } catch {
      proc.kill('SIGTERM');
    }
  }

  return { request, close };
}

async function measureDaemonBuilds() {
  const initial = await measure('server initial build', async () => {
    const daemon = startDaemon();
    const contextId = `bench_initial_${Date.now()}_${Math.random().toString(36).slice(2)}`;
    try {
      await daemon.request('hello');
      await daemon.request('create_context', {
        contextId,
        project,
        compilationMode: process.env.GO_NGC_BENCH_COMPILATION_MODE || 'global',
        hmr: false,
        preserveImports,
      });
      rm(outDir);
      const started = performance.now();
      await daemon.request('build', {
        contextId,
        project,
        compilationMode: process.env.GO_NGC_BENCH_COMPILATION_MODE || 'global',
        hmr: false,
        preserveImports,
      });
      return performance.now() - started;
    } finally {
      await daemon.close();
    }
  });

  const daemon = startDaemon();
  const contextId = `bench_rebuild_${Date.now()}`;
  try {
    await daemon.request('hello');
    await daemon.request('create_context', {
      contextId,
      project,
      compilationMode: process.env.GO_NGC_BENCH_COMPILATION_MODE || 'global',
      hmr: false,
      preserveImports,
    });
    rm(outDir);
    await daemon.request('build', {
      contextId,
      project,
      compilationMode: process.env.GO_NGC_BENCH_COMPILATION_MODE || 'global',
      hmr: false,
      preserveImports,
    });
    const rebuild = await measure('server rebuild no-change', async () => {
      const started = performance.now();
      await daemon.request('build', {
        contextId,
        project,
        compilationMode: process.env.GO_NGC_BENCH_COMPILATION_MODE || 'global',
        hmr: false,
        preserveImports,
      });
      return performance.now() - started;
    });

    return { initial, rebuild };
  } finally {
    await daemon.close();
  }
}

const report = {
  timestamp: new Date().toISOString(),
  root: ROOT,
  projectDir,
  project,
  runs,
  benchmarks: {},
};

report.benchmarks.disk = await measure(
  'one-shot disk',
  async () => runProcess(compileArgs(['--write=true'])).durationMs,
  () => rm(outDir),
);

report.benchmarks.memory = await measure(
  'one-shot memory',
  async () => runProcess(compileArgs(['--write=false', '--format=json'])).durationMs,
  () => rm(outDir),
);

report.benchmarks.server = await measureDaemonBuilds();

const linkerFixture = path.join(projectDir, 'node_modules', 'primeng', 'fesm2022', 'primeng-button.mjs');
if (fs.existsSync(linkerFixture)) {
  report.benchmarks.linkerOneShot = await measure(
    'linker one-shot',
    async () => runProcess(['--link', linkerFixture]).durationMs,
  );

  const daemon = startDaemon();
  try {
    await daemon.request('hello');
    report.benchmarks.linkerServer = await measure('linker server', async () => {
      const started = performance.now();
      await daemon.request('link_file', { path: linkerFixture });
      return performance.now() - started;
    });
  } finally {
    await daemon.close();
  }
}

const reportPath = path.join(reportDir, 'builder-matrix-report.json');
fs.writeFileSync(reportPath, `${JSON.stringify(report, null, 2)}\n`);
console.log(`report: ${reportPath}`);
