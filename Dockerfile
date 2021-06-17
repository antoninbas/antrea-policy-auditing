FROM golang:1.16
RUN mkdir /app
ADD . /app
WORKDIR /app
RUN apt-get update
RUN apt-get install git
RUN go build -o antrea-audit .
CMD ["/app/antrea-audit"]
