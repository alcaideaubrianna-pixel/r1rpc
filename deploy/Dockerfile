# 多阶段构建：前端已预构建并 go:embed 进二进制，这里只编译 Go。
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
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
