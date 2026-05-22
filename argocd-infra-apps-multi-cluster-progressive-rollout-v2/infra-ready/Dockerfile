FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY main.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/infra-ready .

FROM gcr.io/distroless/static:nonroot
COPY --from=build /out/infra-ready /infra-ready
USER nonroot:nonroot
ENTRYPOINT ["/infra-ready"]
