import { fileURLToPath, URL } from 'node:url';
import { defineConfig } from 'vite';
import vue from '@vitejs/plugin-vue';
import tailwindcss from '@tailwindcss/vite';

export default defineConfig({
    plugins: [vue(), tailwindcss()],
    resolve: {
        alias: {
            '@': fileURLToPath(new URL('./src', import.meta.url))
        }
    },
    base: '/',
    build: {
        outDir: '../internal/chatlog/http/static',
        emptyOutDir: true,
        target: 'es2020',
        cssCodeSplit: false,
        chunkSizeWarningLimit: 1024,
        rollupOptions: {
            output: {
                entryFileNames: 'assets/[name]-[hash].js',
                chunkFileNames: 'assets/[name]-[hash].js',
                assetFileNames: 'assets/[name]-[hash][extname]'
            }
        }
    },
    server: {
        port: 5173,
        proxy: {
            '/api': 'http://127.0.0.1:5030',
            '/health': 'http://127.0.0.1:5030',
            '/image': 'http://127.0.0.1:5030',
            '/video': 'http://127.0.0.1:5030',
            '/voice': 'http://127.0.0.1:5030',
            '/file': 'http://127.0.0.1:5030',
            '/data': 'http://127.0.0.1:5030',
            '/mcp': 'http://127.0.0.1:5030',
            '/sse': 'http://127.0.0.1:5030'
        }
    }
});
