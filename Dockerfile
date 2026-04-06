FROM scratch
ARG BINARY=testmd
COPY ${BINARY} /testmd
ENTRYPOINT ["/testmd"]
