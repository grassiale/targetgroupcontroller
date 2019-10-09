FROM golang:1.13.1-alpine3.10 as build
RUN apk add -U --no-cache ca-certificates
RUN addgroup -S appgroup && adduser -S gopher -G appgroup
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY ./*.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -o targetgroupcontroller 

FROM scratch
COPY --from=build /app/targetgroupcontroller /
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /etc/passwd /etc/passwd
USER gopher
ENTRYPOINT ["/targetgroupcontroller"]
