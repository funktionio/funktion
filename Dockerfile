FROM alpine

ADD ./out/funktion-linux-amd64 /bin/operator

ENTRYPOINT ["/bin/operator"]
