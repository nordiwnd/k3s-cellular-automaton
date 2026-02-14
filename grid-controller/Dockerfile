# syntax=docker/dockerfile:1.4
FROM golang:alpine AS builder

WORKDIR /app
COPY go.mod ./
# COPY go.sum ./ # No dependencies yet
RUN go mod download

COPY . .
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags="-w -s" -o controller .

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /app/controller .
USER 65532:65532
ENTRYPOINT ["/controller"]
