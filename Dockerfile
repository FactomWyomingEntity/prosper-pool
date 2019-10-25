
# Start from the latest golang base image
FROM golang:latest

# Set the Current Working Directory inside the container
RUN mkdir -p /go/src/prosper-pool
WORKDIR /go/src/prosper-pool

# Copy go mod and sum files
COPY go.mod go.sum wait-for-it.sh ./

COPY prosper-pool.toml.example /root/.prosper/prosper-pool.toml

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source from the current directory to the Working Directory inside the container
COPY . .

# Build the Go private-pool application
RUN go install .

# Expose port 1234 to the outside world
EXPOSE 1234

# Update permissions on the "wait for db" script
RUN ["chmod", "+x", "/go/src/prosper-pool/wait-for-it.sh"]