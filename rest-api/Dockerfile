FROM rook/toolbox:master
MAINTAINER Steve Sloka <steves@heptio.com>

RUN apt update \
    && apt upgrade -y \
    && apt install python python-dev python-pip -y \
    && pip install awscli

ADD _output/bin/rook-api /rook-api
ADD toolbox.sh /usr/local/bin/toolbox.sh

ENTRYPOINT ["/rook-api"]

