FROM amd64/alpine:3.16
COPY /build/monaco-linux-amd64 /usr/local/bin/monaco
RUN chmod +x /usr/local/bin/monaco
ENTRYPOINT ["/usr/local/bin/monaco"]
CMD ["--help"]
