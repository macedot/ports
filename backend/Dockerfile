# syntax=docker/dockerfile:1

# Stage 1: Build Vue.js frontend
FROM node:20-alpine AS frontend-build
WORKDIR /app
COPY frontend/package.json frontend/package-lock.json ./
RUN npm install --ignore-optional
COPY frontend/ .
RUN npm run build

# Stage 2: Build Go binary with embedded frontend
FROM golang:alpine AS backend-build
WORKDIR /app
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ .
# Copy frontend dist into the ui/dist directory for go:embed
COPY --from=frontend-build /app/dist ./ui/dist/
ENV CGO_ENABLED=0
RUN go build -ldflags="-w -s" -o /app/server .

# Stage 3: Minimal runtime
FROM gcr.io/distroless/static-debian12
ENV PORT=8080
EXPOSE 8080
COPY --from=backend-build /app/server /app/server
USER 65534:65534
ENTRYPOINT ["/app/server"]
