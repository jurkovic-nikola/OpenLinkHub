.PHONY: frontend-build frontend-dev build

frontend-build:
	cd frontend && npm ci && npm run build

frontend-dev:
	cd frontend && npm run dev

build: frontend-build
	CGO_CFLAGS_ALLOW='-fno-strict-overflow' go build .
