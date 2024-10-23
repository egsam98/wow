FROM golang:1.23-alpine as BUILDER

COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /app/client apps/client/cmd/main.go

FROM scratch
COPY --from=BUILDER /app/client /app/client
ENTRYPOINT ["/app/client"]
