FROM golang:stretch AS build

RUN apt-get update && apt-get install -y ca-certificates && apt-get clean
RUN adduser --disabled-password --gecos '' gopher
ADD . /go/src/github.com/lifeonair/targetgroupcontroller
RUN cd /go/src/github.com/lifeonair/targetgroupcontroller && go get && go build

FROM scratch
COPY --from=build /etc/ssl/certs/ /etc/ssl/
COPY --from=build /go/bin/targetgroupcontroller /bin/targetgroupcontroller
COPY --from=build /etc/passwd /etc/passwd
USER gopher
ENTRYPOINT ["/bin/targetgroupcontroller"]
