# Step 1: build
FROM golang:1.26.8-alpine3.24 AS build

WORKDIR /app

COPY go.mod .

RUN go mod download && go mod verify

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o app .

# Step 2: image
FROM scratch

WORKDIR /app

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build ./app .

ENTRYPOINT ["./app"]