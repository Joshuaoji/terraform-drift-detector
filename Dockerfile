FROM node:22-alpine AS web
WORKDIR /app/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.25-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /app/web/dist ./internal/api/webdist
RUN CGO_ENABLED=0 go build -o /driftdetect ./cmd/driftdetect

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /driftdetect /usr/local/bin/driftdetect
COPY configs/ /app/configs/
COPY testdata/ /app/testdata/
EXPOSE 8080
ENTRYPOINT ["driftdetect"]
