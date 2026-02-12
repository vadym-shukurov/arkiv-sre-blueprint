FROM golang:1.22-alpine AS build
WORKDIR /app
COPY go.mod main.go ./
RUN go mod download && go mod tidy
RUN CGO_ENABLED=0 go build -o faucet .

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
COPY --from=build /app/faucet /faucet
EXPOSE 8080
ENTRYPOINT ["/faucet"]
