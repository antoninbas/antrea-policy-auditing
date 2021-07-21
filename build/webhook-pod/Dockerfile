FROM golang:1.16 as audit-build
WORKDIR /antrea
COPY . /antrea
RUN CGO_ENABLED=0 go build -o antrea-audit-webhook cmd/webhook/main.go

FROM scratch
LABEL maintainer="Antrea <projectantrea-dev@googlegroups.com>"
LABEL description="The docker image for the webhook of the auditing system"
ENV USER root
COPY --from=audit-build /antrea/antrea-audit-webhook /
ENTRYPOINT ["/antrea-audit-webhook", "-v", "2"]
