import { defineConfig } from 'vite';
import angularGo from 'vite-plugin-angular-go';

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
      compilerPath: '../../go-ngc',
      project: 'tsconfig.app.json',
      outDir: 'out-tsc/app',
      compile: true,
      link: true,
      recompileOnChange: true
    })
  ],
  resolve: {
    mainFields: ['module']
  }
});
