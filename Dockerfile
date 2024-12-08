# Step 1: Use an official Go image to build the application
FROM golang:1.23-alpine as builder

# Step 2: Set the working directory in the container
WORKDIR /app

# Step 3: Copy the Go modules files to the container
COPY go.mod go.sum ./

# Step 4: Download Go dependencies
RUN go mod tidy

# Step 5: Copy the rest of the application source code
COPY . .

# Step 6: Build the Go binary
RUN GOOS=linux GOARCH=amd64 go build -o grpc-app api/cmd/main.go

# Step 7: Use a smaller base image to run the application
FROM alpine:3.18

# Step 8: Install dependencies required to run the app (if needed, e.g., certificates)
RUN apk add --no-cache ca-certificates

# Step 9: Set the working directory in the container
WORKDIR /root/

# Step 10: Copy the Go binary from the builder stage
COPY --from=builder /app/grpc-app .

# Step 11: Expose the necessary ports (e.g., gRPC and HTTP)
EXPOSE 9091
EXPOSE 8081

# Step 12: Run the Go application
CMD ["./grpc-app"]
