FROM registry.suse.com/bci/golang:1.21-openssl

ARG FILES_COUNT
ARG CUSTOMER_ID

ENV CONFIG_TEMPLATE="cmd/faker/config/client_template.yaml"
ENV FILES_COUNT=${FILES_COUNT}
ENV CUSTOMER_ID=${CUSTOMER_ID}

WORKDIR /app
COPY . .

RUN mkdir -p /tmp/susetelemetry
RUN cd cmd/faker; go run . --count=${FILES_COUNT}
RUN chmod +x "cmd/faker/scripts/process_files.sh"

ENTRYPOINT [ "/bin/sh", "-c", "cmd/faker/scripts/process_files.sh $FILES_COUNT $CUSTOMER_ID" ]