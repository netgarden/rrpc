import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [sveltekit()],
	server: {
		proxy: {
			'/Auth': 'http://localhost:8080',
			'/Pinger': 'http://localhost:8080',
			'/files': 'http://localhost:8080',
		}
	}
});
