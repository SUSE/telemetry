FROM registry.suse.com/bci/golang:1.21-openssl

WORKDIR /app
COPY . .

RUN mkdir -p /tmp/susetelemetry

ENTRYPOINT [ "go", "run", "cmd/faker/process/process_files.go" ]