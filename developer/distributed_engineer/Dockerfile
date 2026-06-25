# Consensus — distributed-engineer learning platform.
#
# IMPORTANT: the in-app code checker shells out to `go test`, so the RUNTIME image
# must contain the Go toolchain (not a scratch/distroless image). We therefore run
# from the golang image. Everything (templates, CSS, JS, migrations, lessons,
# interview questions, seed data) is embedded in the binary, so no extra files are
# copied into the runtime stage.

# ---- build stage ----
FROM golang:1.21-bookworm AS build
WORKDIR /src

# Cache module downloads.
COPY go.mod go.sum ./
RUN go mod download

# Build a static binary (pgx/echo/goldmark are pure Go; CGO not needed).
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/consensus ./cmd/server

# ---- runtime stage (keeps Go for the checker) ----
FROM golang:1.21-bookworm
WORKDIR /app

# Pre-warm the Go build cache so the FIRST checker run isn't a ~15s cold compile.
ENV GOCACHE=/root/.cache/go-build GOFLAGS=-mod=mod
RUN mkdir -p /tmp && cd /tmp && \
    printf 'package solution\nfunc Add(a, b int) int { return a + b }\n' > solution.go && \
    printf 'package solution\nimport "testing"\nfunc TestAdd(t *testing.T){ if Add(1,2)!=3 {t.Fatal("x")} }\n' > solution_test.go && \
    printf 'module solution\n\ngo 1.21\n' > go.mod && \
    GOPROXY=off go test ./... >/dev/null 2>&1 || true && \
    rm -f /tmp/solution.go /tmp/solution_test.go /tmp/go.mod

COPY --from=build /out/consensus /app/consensus

ENV APP_ENV=prod PORT=8080
EXPOSE 8080
CMD ["/app/consensus"]
