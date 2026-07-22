# OpenLinkHub WebUI

SvelteKit 2 + Svelte 5 + Tailwind 4 + [Skeleton](https://www.skeleton.dev/) SPA.

## Develop

```bash
npm install
npm run dev
```

Vite proxies `/api` (including WebSocket `/api/ws`) to `http://127.0.0.1:27003`.

## Production build

From the repo root:

```bash
make frontend-build
```

This runs `npm ci && npm run build` and copies `frontend/build` → `ui/`, which the Go server serves when `frontend` is enabled in `config.json`.
