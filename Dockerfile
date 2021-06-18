FROM golang:1.16
RUN mkdir /app
ADD . /app
WORKDIR /app
LABEL maintainer="Antrea <projectantrea-dev@googlegroups.com>"
LABEL description="The docker image for the auditing system"
RUN go build -o antrea-audit .
CMD ["/app/antrea-audit"]
