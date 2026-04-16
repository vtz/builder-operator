FROM golang:1.22 AS builder
WORKDIR /workspace
COPY go.mod go.sum ./
RUN go mod download
COPY . .

ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-s -w" -a -o manager main.go

RUN mkdir -p /cli && \
    CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64 go build -ldflags "-s -w -X main.version=${VERSION}" -o /cli/bob-darwin-amd64  ./cmd/bob && \
    CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build -ldflags "-s -w -X main.version=${VERSION}" -o /cli/bob-darwin-arm64  ./cmd/bob && \
    CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -ldflags "-s -w -X main.version=${VERSION}" -o /cli/bob-linux-amd64   ./cmd/bob && \
    CGO_ENABLED=0 GOOS=linux   GOARCH=arm64 go build -ldflags "-s -w -X main.version=${VERSION}" -o /cli/bob-linux-arm64   ./cmd/bob

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/manager .
COPY --from=builder /cli /cli
USER 65532:65532
ENTRYPOINT ["/manager"]
