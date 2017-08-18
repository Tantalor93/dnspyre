FROM scratch

ARG BUILD_DATE
ARG VCS_REF
ARG VERSION

LABEL org.label-schema.build-date=$BUILD_DATE \
      org.label-schema.name="DNStrace" \
      org.label-schema.description="Command-line DNS benchmark tool built to stress test and measure the performance of DNS servers" \
      org.label-schema.vcs-ref=$VCS_REF \
      org.label-schema.vcs-url="https://github.com/redsift/dnstrace" \
      org.label-schema.vendor="Redsift Limited." \
      org.label-schema.version=$VERSION \
      org.label-schema.schema-version="1.0"

ADD dnstrace /

ENTRYPOINT ["/dnstrace"]