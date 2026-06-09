import { defineConfig } from 'vite';
import angularGo from '@angular-go/build/vite';

const enableAngularGoHmr = process.env.NG_GO_HMR === '1';

export default defineConfig({
  root: 'src',
  define: {
    'process.env.NG_GO_VERSION': JSON.stringify('1.0.0')
  },
  server: {
    port: 4200,
  },
  build: {
    outDir: '../dist',
    emptyOutDir: true
  },
  plugins: [
    angularGo({
      mode: 'server',
      project: 'tsconfig.app.json',
      outDir: 'out-tsc/app',
      compile: true,
      link: true,
      hmr: enableAngularGoHmr,
      recompileOnChange: true
    })
  ],
  resolve: {
    mainFields: ['module']
  }
});
