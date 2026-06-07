import { afterEach } from 'vitest';
import { cleanupTestbed } from '../render.js';

export function setupAngularGoTestbed() {
  afterEach(() => {
    cleanupTestbed();
  });
}
