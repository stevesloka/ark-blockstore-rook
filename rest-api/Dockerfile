FROM rook/toolbox:master
MAINTAINER Steve Sloka <steves@heptio.com>

ADD _output/bin/rook-api /rook-api
ADD toolbox.sh /usr/local/bin/toolbox.sh

ENTRYPOINT ["/rook-api"]

