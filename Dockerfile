FROM node:25-alpine@sha256:bdf2cca6fe3dabd014ea60163eca3f0f7015fbd5c7ee1b0e9ccb4ced6eb02ef4 AS ui-build
WORKDIR /app
COPY ui/package*.json ./ui/
RUN cd ui && npm ci
COPY ui/ ./ui/
# Create output directory and build - vite outputs directly to pkg/github/ui_dist/
RUN mkdir -p ./pkg/github/ui_dist && \
    cd ui && npm run build

FROM golang:1.26.2-alpine@sha256:f85330846cde1e57ca9ec309382da3b8e6ae3ab943d2739500e08c86393a21b1 AS build
ARG VERSION="dev"

# Set the working directory
WORKDIR /build

# Install git
RUN --mount=type=cache,target=/var/cache/apk \
    apk add git

# Copy source code (including ui_dist placeholder)
COPY . .

# Copy built UI assets over the placeholder
COPY --from=ui-build /app/pkg/github/ui_dist/* ./pkg/github/ui_dist/

# Build the server
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=${VERSION} -X main.commit=$(git rev-parse HEAD) -X main.date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -o /bin/omnigit-mcp ./cmd/omnigit-mcp

# Make a stage to run the app
FROM gcr.io/distroless/base-debian12@sha256:9dce90e688a57e59ce473ff7bc4c80bc8fe52d2303b4d99b44f297310bbd2210

# Add required MCP server annotation
LABEL io.modelcontextprotocol.server.name="io.github.aifity/omnigit-mcp"

# Set the working directory
WORKDIR /server
# Copy the binary from the build stage
COPY --from=build /bin/omnigit-mcp .
# Expose the default port
EXPOSE 8082
# Set the entrypoint to the server binary
ENTRYPOINT ["/server/omnigit-mcp"]
# Default arguments for ENTRYPOINT
CMD ["stdio"]
