## Certificate Authority
openssl genrsa -out ca.key 4096

openssl req -new -key ca.key -x509 -days 365 -out ca.crt -subj "/C=US/ST=Kansas/O=localhost/CN=localhost"

## Server
openssl genrsa -out server.key 4096

openssl req -new -nodes -key server.key -out server.csr -subj "/C=US/ST=Kansas/O=localhost/CN=localhost"

openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt

## Client
openssl genrsa -out client.key 4096

openssl req -new -nodes -key client.key -out client.csr -subj "/C=US/ST=Kansas/O=samschurter@makeict.org"

openssl x509 -req -in client.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out client.crt
