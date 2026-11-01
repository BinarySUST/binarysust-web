# Binary SUST API: a static Go binary on distroless (CA certificates included, no shell).
FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=build /out/server /server
EXPOSE 8080
# Render's free plan has no pre-deploy step, so migrations run at boot (advisory-locked).
ENTRYPOINT ["/server"]
CMD ["serve", "--migrate"]
