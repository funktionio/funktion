FROM scratch

ADD funktion-linux-static /bin/operator

ENTRYPOINT ["/bin/operator", "operate"]