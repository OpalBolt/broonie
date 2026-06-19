FROM golang:1.22-alpine AS builder

WORKDIR /go/src/app

COPY . .

RUN CGO_ENABLED=0 go build -o broonie .

FROM gcr.io/distroless/static

COPY --from=builder /go/src/app/broonie /broonie

VOLUME ["/data", "/sessions", "/workspaces"]

ENTRYPOINT ["/broonie"]
