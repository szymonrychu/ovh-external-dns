FROM golang:1.16.3 as build

# Set the Current Working Directory inside the container
WORKDIR /app

RUN export GO111MODULE=on

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /go/bin/main 

FROM scratch

COPY --from=build /go/bin/main /go/bin/main
COPY --from=alpine:latest /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Expose port 9000 to the outside world
EXPOSE 8090

# Command to run the executable
ENTRYPOINT ["/go/bin/main"]