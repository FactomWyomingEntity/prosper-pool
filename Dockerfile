
# Start from the latest golang base image
FROM golang:latest

# Set the Current Working Directory inside the container
RUN mkdir -p /go/src/prosper-pool

# For UI
RUN mkdir -p /go/src/github.com/qor
RUN git clone https://github.com/qor/auth_themes.git
RUN mv auth_themes /go/src/github.com/qor

WORKDIR /go/src/prosper-pool

# Copy go mod and sum files
COPY go.mod go.sum wait-for-it.sh ./
COPY prosper-pool.toml.example /root/.prosper/prosper-pool.toml

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source from the current directory to the Working Directory inside the container
COPY . .

# Expose ports 1234 and 7070 to the outside world
EXPOSE 1234 7070

# Build the Go prosper-pool application
RUN ["/go/src/prosper-pool/build.sh"]
