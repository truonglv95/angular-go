import { performance } from 'perf_hooks';

export async function benchmark<T>(name: string, fn: () => Promise<T>): Promise<T> {
  const start = performance.now();
  try {
    return await fn();
  } finally {
    const end = performance.now();
    console.log(`[Benchmark] ${name} took ${(end - start).toFixed(2)}ms`);
  }
}
