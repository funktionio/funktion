FROM scratch

ADD funktion-operator-linux-static /bin/operator

ENTRYPOINT ["/bin/operator"]