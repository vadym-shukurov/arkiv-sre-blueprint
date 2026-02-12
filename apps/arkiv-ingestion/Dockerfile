FROM golang:1.22-alpine AS build
WORKDIR /app
COPY go.mod ./
COPY adapter.go fetcher_synthetic.go ingester_postgres.go main.go ./
RUN go mod download && go mod tidy && CGO_ENABLED=0 go build -o arkiv-ingestion .

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
COPY --from=build /app/arkiv-ingestion /arkiv-ingestion
EXPOSE 8080
ENTRYPOINT ["/arkiv-ingestion"]
