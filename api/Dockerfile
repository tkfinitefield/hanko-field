# syntax=docker/dockerfile:1.7

FROM golang:1.21 as builder

WORKDIR /workspace

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGETOS=linux
ARG TARGETARCH=amd64

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -o /out/hanko-api ./cmd/api

FROM gcr.io/distroless/base-debian12:nonroot as runner

ENV PORT=8080

COPY --from=builder /out/hanko-api /usr/local/bin/hanko-api

EXPOSE 8080

USER nonroot

ENTRYPOINT ["/usr/local/bin/hanko-api"]
