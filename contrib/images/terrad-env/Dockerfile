FROM ubuntu:18.04

RUN apt-get update && \
  apt-get -y upgrade && \
  apt-get -y install curl jq file

ARG UID=1000
ARG GID=1000
ARG wrapper_script=wrapper.sh

USER ${UID}:${GID}
VOLUME /terrad
WORKDIR /terrad
EXPOSE 26656 26657
ENTRYPOINT ["/usr/bin/wrapper.sh"]
CMD ["start", "--log_format", "plain"]
STOPSIGNAL SIGTERM

COPY ${wrapper_script} /usr/bin/wrapper.sh
