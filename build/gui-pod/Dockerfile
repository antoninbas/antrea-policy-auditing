FROM alpine:3.14
RUN apk update
RUN apk add git
RUN apk add python3
RUN wget -O - https://raw.githubusercontent.com/alberthier/git-webui/master/install/installer.sh | sh
RUN ln -s /usr/bin/python3 /usr/bin/python
WORKDIR /home/download/network-policy-repository
CMD ["git", "webui"]
