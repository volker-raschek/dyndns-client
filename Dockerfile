FROM scratch AS build

COPY dyndns-client-* /usr/bin/app

ENTRYPOINT [ "/usr/bin/dyndns-client" ]
