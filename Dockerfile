FROM node:25-alpine@sha256:ad82ecad30371c43f4057aaa4800a8ed88f9446553a2d21323710c7b937177fc AS ui-build
WORKDIR /app
COPY ui/package*.json ./ui/
RUN cd ui && npm ci
COPY ui/ ./ui/
# Create output directory and build - vite outputs directly to pkg/github/ui_dist/
RUN mkdir -p ./pkg/github/ui_dist && \
    cd ui && npm run build

FROM golang:1.26.2-alpine@sha256:c2a1f7b2095d046ae14b286b18413a05bb82c9bca9b25fe7ff5efef0f0826166 AS build
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
