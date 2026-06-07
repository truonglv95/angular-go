import { afterEach } from 'vitest';
import { cleanupTestbed } from '../testbed/render.js';

export function setupAngularGoTestbed() {
  afterEach(() => {
    cleanupTestbed();
  });
}
