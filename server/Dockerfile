# Stage 1: 빌드
FROM golang:1.25-alpine AS build
WORKDIR /src

# 의존성만 먼저 받아 두면 소스가 바뀌어도 캐시가 유지된다
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

# Stage 2: 실행. 단일 실행 파일만 담는다 (마이그레이션 SQL은 실행 파일에 포함돼 있다)
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/server /server
EXPOSE 4000
ENTRYPOINT ["/server"]
