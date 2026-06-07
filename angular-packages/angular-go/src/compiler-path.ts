import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const PLATFORM_PACKAGES: Record<string, string> = {
  'darwin-arm64': 'angular-go-darwin-arm64',
  'darwin-x64': 'angular-go-darwin-x64',
  'linux-x64': 'angular-go-linux-x64',
  'win32-x64': 'angular-go-win32-x64'
};

export function resolveGoNgcPath(projectRoot: string, compilerPath = 'go-ngc', fromUrl = import.meta.url): string {
  if (compilerPath.includes('/') || compilerPath.includes(path.sep)) {
    return path.isAbsolute(compilerPath) ? compilerPath : path.resolve(projectRoot, compilerPath);
  }

  if (compilerPath !== 'go-ngc') {
    return compilerPath;
  }

  const workspaceBinary = findWorkspaceBinary(projectRoot);
  if (workspaceBinary) {
    return workspaceBinary;
  }

  const packageName = PLATFORM_PACKAGES[`${process.platform}-${process.arch}`];
  if (!packageName) {
    return compilerPath;
  }

  const currentDir = path.dirname(fromUrl.startsWith('file:') ? fileURLToPath(fromUrl) : fromUrl);
  const packageJson = findPackageJson(currentDir, packageName) ?? findPackageJson(projectRoot, packageName);
  if (!packageJson) {
    return compilerPath;
  }

  const binaryName = process.platform === 'win32' ? 'go-ngc.exe' : 'go-ngc';
  const candidate = path.join(path.dirname(packageJson), 'bin', binaryName);
  return fs.existsSync(candidate) ? candidate : compilerPath;
}

function findWorkspaceBinary(startDir: string): string | null {
  let dir = startDir;
  while (true) {
    const candidate = path.join(dir, process.platform === 'win32' ? 'go-ngc.exe' : 'go-ngc');
    if (fs.existsSync(candidate)) {
      return candidate;
    }
    const parent = path.dirname(dir);
    if (parent === dir) {
      return null;
    }
    dir = parent;
  }
}

function findPackageJson(startDir: string, packageName: string): string | null {
  let dir = startDir;
  while (true) {
    const candidate = path.join(dir, 'node_modules', packageName, 'package.json');
    if (fs.existsSync(candidate)) {
      return candidate;
    }
    const parent = path.dirname(dir);
    if (parent === dir) {
      return null;
    }
    dir = parent;
  }
}

