# Drift Detector Web Dashboard

React + Vite frontend for the Terraform drift detector platform.

## Development

Start the Go API server in one terminal:

```bash
make build
./bin/driftdetect serve --port 8080 --config ../configs/example.yaml
```

Start the Vite dev server in another terminal:

```bash
cd web
npm install
npm run dev
```

Open **http://localhost:5173**. API requests are proxied to `http://localhost:8080`.

## Production build

```bash
# From repository root
make build-web   # builds web/dist and copies to internal/api/webdist for Go embed
make build       # builds Go binary with embedded frontend
```

Or combine both:

```bash
make build-all
```

The production dashboard is served by the Go binary at `http://localhost:8080/`.

## Stack

- React 19
- TypeScript
- Vite 6
- Tailwind CSS 4
