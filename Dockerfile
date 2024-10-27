# Use the official Go image as a builder
FROM golang:1.23 AS builder

# Set the working directory
WORKDIR /app

# Copy the Go modules and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the Go application
RUN go build -o salin .
CMD [ "./salin" ]
