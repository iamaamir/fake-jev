# Minimal container image for fake-jev (spec §24.2, §31 Phase 4, §32).
#
# Stage 1 (build) carries the Go toolchain and produces a statically linked
# binary. Stage 2 (runtime) is `scratch`: it holds the binary and the CA
# certificate bundle and nothing else — no Go toolchain, no package manager,
# no shell, no source tree. The image therefore runs with only the Docker
# daemon and this image present: no Go install, no model, no provider SDK.
#
# The final stage is built from scratch, so the go.mod toolchain version is
# pinned here (1.26, spec §17.1) rather than resolved at runtime.

FROM golang:1.26-bookworm AS build

# Target platform for the cross-compile. Docker supplies both arguments from
# `--platform`/buildx; the defaults keep a plain `docker build` working on
# linux/amd64.
ARG TARGETOS=linux
ARG TARGETARCH=amd64

WORKDIR /src

# Dependency layer first, so editing source does not refetch the module graph.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO_ENABLED=0 yields a static executable with no dynamic loader and no C
# library, which is what lets the scratch stage below run it (spec §32:
# "a standalone native binary"). -trimpath keeps the build reproducible.
RUN CGO_ENABLED=0 GOOS="${TARGETOS}" GOARCH="${TARGETARCH}" \
      go build -trimpath -ldflags="-s -w" -o /out/fake-jev ./cmd/fake-jev

FROM scratch

# The only OS files the runtime carries: a CA bundle (spec §24.2) and the
# binary. The server itself makes no outbound request (spec §19.2); the bundle
# exists so a container may be extended with tooling that expects it.
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /out/fake-jev /fake-jev

# Non-root, numeric because scratch has no /etc/passwd (65532:nonroot).
USER 65532:65532

# Informational; binding is controlled by the flags below.
EXPOSE 8787

ENTRYPOINT ["/fake-jev"]

# Container-friendly binding is an explicit choice here (spec §19.1, §24.2):
# the binary's own default stays 127.0.0.1, and only the image's default
# command opts into 0.0.0.0 so `docker run -p 8787:8787` is reachable.
# Passing arguments replaces this CMD, so a mounted config file works too:
#   docker run --rm -p 8787:8787 -v "$PWD/fake-jev.yaml:/etc/fake-jev.yaml" \
#     fake-jev serve --config /etc/fake-jev.yaml --host 0.0.0.0
CMD ["serve", "--host", "0.0.0.0", "--port", "8787"]
