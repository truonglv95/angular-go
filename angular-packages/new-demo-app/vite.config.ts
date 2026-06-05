import { defineConfig } from 'vite';
import angularGo from 'vite-plugin-angular-go';

const enableAngularGoHmr = process.env.NG_GO_HMR === '1';

export default defineConfig({
  root: 'src',
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
      compilerPath: '../../go-ngc',
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
