.PHONY: frontend-build frontend-dev build

frontend-build:
	cd frontend && npm ci && npm run build
	rm -rf ui
	cp -R frontend/build ui

frontend-dev:
	cd frontend && npm run dev -- --host

build: frontend-build
	CGO_CFLAGS_ALLOW='-fno-strict-overflow' go build .
