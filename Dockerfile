FROM scratch
COPY endpoint-controller /usr/bin/endpoint-controller
USER 1000:1000
CMD ["/usr/bin/endpoint-controller"]

