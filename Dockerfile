FROM dockerhub.deepglint.com/base/alpine:3.11

ARG path
ARG name

WORKDIR /$name

COPY $path /$name/

EXPOSE 4096
ENV ENVIRONMENT production
ENTRYPOINT ["./testTemplate"]
