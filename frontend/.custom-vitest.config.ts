import { defineConfig } from 'vitest/config'
import { resolve } from 'node:path'
import vue from '@vitejs/plugin-vue'

const frontendRoot = __dirname

export default defineConfig({
  root: frontendRoot,
  plugins: [vue()],
  resolve: {
    alias: [
      { find: /^@\/types\/payment$/, replacement: resolve(frontendRoot, 'src/custom/types/payment.ts') },
      { find: /^@\/types$/, replacement: resolve(frontendRoot, 'src/custom/types/index-fork.ts') },
      { find: '@', replacement: resolve(frontendRoot, 'src') },
      { find: 'vue-i18n', replacement: 'vue-i18n/dist/vue-i18n.runtime.esm-bundler.js' },
    ],
  },
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./src/__tests__/setup.ts', './src/custom/__tests__/setup.ts'],
    include: ['src/custom/**/*.{test,spec}.{js,ts,jsx,tsx}'],
    exclude: ['node_modules', 'dist'],
    fileParallelism: false,
    maxWorkers: 1,
    minWorkers: 1,
  },
})
