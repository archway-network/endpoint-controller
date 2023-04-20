# Build stage
FROM golang:1.20-alpine AS build
WORKDIR /app
COPY . .
RUN go build -o endpoint-controller .

# Final stage
FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /app/endpoint-controller /endpoint-controller
USER 1000:1000
CMD ["/endpoint-controller"]

