FROM alpine:3.6
RUN mkdir /plugins
ADD PLUGIN_NAME /plugins
USER nobody:nobody
ENTRYPOINT ["/bin/ash", "-c", "cp -a /plugins/* /target/."]
