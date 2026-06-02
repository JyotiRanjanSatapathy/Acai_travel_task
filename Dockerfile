# syntax=docker/dockerfile:1

# --- Build stage ---------------------------------------------------------
FROM golang:1.25 AS build

WORKDIR /src

# Download dependencies first to leverage Docker layer caching.
COPY go.mod go.sum ./
RUN go mod download

# Build a fully static binary so it runs in a minimal base image.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /bin/server ./cmd/server

# --- Runtime stage -------------------------------------------------------
FROM gcr.io/distroless/static-debian12:nonroot

# Distroless ships with CA certificates, needed for outbound HTTPS calls
# (OpenAI, weather API). Run as the unprivileged "nonroot" user.
COPY --from=build /bin/server /server

EXPOSE 8080

ENTRYPOINT ["/server"]
