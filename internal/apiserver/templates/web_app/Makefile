.PHONY: dev api ui build

# Start both API server and React dev server
dev:
	@echo "Starting AILANG API server and React dev server..."
	@make api & make ui & wait

# Start AILANG API server
api:
	ailang serve-api ./api/ --port 8080

# Start React dev server (proxies /api to port 8080)
ui:
	cd ui && npm run dev

# Build React frontend for production
build:
	cd ui && npm run build

# Serve with built frontend
serve:
	ailang serve-api ./api/ --port 8080 --static ./ui/dist

# Install UI dependencies
install:
	cd ui && npm install
