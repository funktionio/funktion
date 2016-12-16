FROM scratch

ADD ./out/funktion-operator-linux-amd64 /bin/operator

ENTRYPOINT ["/bin/operator"]