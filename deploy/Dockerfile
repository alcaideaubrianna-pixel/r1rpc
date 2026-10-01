# Git 部署必须在镜像内构建前端，不能依赖仓库中已有的嵌入产物。
FROM node:22-alpine AS web-build
WORKDIR /src
COPY web/package.json web/package-lock.json ./web/
RUN npm ci --prefix web
COPY web ./web
COPY internal/web/ui ./internal/web/ui
RUN npm run build --prefix web

FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web-build /src/internal/web/ui ./internal/web/ui
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags "-s -w" -o /out/r1rpc-server ./cmd/server

FROM debian:bookworm-slim
RUN apt-get update \
  && apt-get install -y --no-install-recommends ca-certificates tzdata wget tesseract-ocr tesseract-ocr-chi-sim tesseract-ocr-eng \
  && tesseract --list-langs | grep -q chi_sim \
  && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=build /out/r1rpc-server /app/r1rpc-server
EXPOSE 9876
HEALTHCHECK --interval=10s --timeout=3s --start-period=10s --retries=12 \
  CMD wget -q -O - http://127.0.0.1:9876/healthz || exit 1
ENTRYPOINT ["/app/r1rpc-server"]
