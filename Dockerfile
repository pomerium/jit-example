FROM golang:1.23.3-bookworm@sha256:3f3b9daa3de608f3e869cd2ff8baf21555cf0fca9fd34251b8f340f9b7c30ec5

COPY . /src/github.com/pomerium/jit-example
WORKDIR /src/github.com/pomerium/jit-example
RUN go build -o /bin/jit-example .

ENV PORT=8000
ENTRYPOINT [ "/bin/jit-example" ]
