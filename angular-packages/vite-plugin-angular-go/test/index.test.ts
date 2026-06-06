import { describe, expect, it } from 'vitest';
import angularGo, { angularGoCompile, angularGoLinker } from '../src/index';

describe('vite-plugin-angular-go exports', () => {
  it('creates split compile and linker plugins by default', () => {
    const plugins = angularGo();
    expect(plugins.map((plugin: any) => plugin.name)).toEqual([
      'vite-plugin-angular-go-compile',
      'vite-plugin-angular-go-linker',
    ]);
  });

  it('can create compile-only and linker-only plugin stacks', () => {
    expect(angularGo({ link: false }).map((plugin: any) => plugin.name)).toEqual([
      'vite-plugin-angular-go-compile',
    ]);
    expect(angularGo({ compile: false }).map((plugin: any) => plugin.name)).toEqual([
      'vite-plugin-angular-go-linker',
    ]);
  });

  it('keeps Angular HMR opt-in and disabled for production builds', () => {
    const devPlugin = angularGoCompile({ hmr: true });
    devPlugin.configResolved({ command: 'serve' } as any);
    expect(devPlugin.resolveId('/app/@ng/component?c=src%2Fapp%2Fapp.ts%40App')).toBe(
      '\0/app/@ng/component?c=src%2Fapp%2Fapp.ts%40App',
    );

    const prodPlugin = angularGoCompile({ hmr: true });
    prodPlugin.configResolved({ command: 'build' } as any);
    expect(prodPlugin.resolveId('/app/@ng/component?c=src%2Fapp%2Fapp.ts%40App')).toBeUndefined();

    const linkerPlugin = angularGoLinker();
    expect(linkerPlugin.name).toBe('vite-plugin-angular-go-linker');
  });
});
