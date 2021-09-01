
## Start Job
curl -k -H "Content-Type: application/json" -X POST --cert certs/client.crt --key certs/client.key --cacert certs/ca.crt  --tlsv1.2 https://localhost:4430/start -d '{"job":"sleep", "args":["1"]}'

curl -k -H "Content-Type: application/json" -X POST --cert certs/client.crt --key certs/client.key --cacert certs/ca.crt  --tlsv1.2 https://localhost:4430/start -d '{"job":"sleep", "args":["foo"]}'

## Status
curl -k --cert certs/client.crt --key certs/client.key --cacert certs/ca.crt  --tlsv1.2 https://localhost:4430/status/6016d308-26cd-4377-ab8f-de49ffe6e8b7

## Stop
curl -k -X POST --cert certs/client.crt --key certs/client.key --cacert certs/ca.crt  --tlsv1.2 https://localhost:4430/stop/1582c14b-38a3-4a5d-8bd7-e98a16af39f8

## Stdout
curl -k --cert certs/client.crt --key certs/client.key --cacert certs/ca.crt  --tlsv1.2 https://localhost:4430/stdout/ID

## Stderr
curl -k --cert certs/client.crt --key certs/client.key --cacert certs/ca.crt  --tlsv1.2 https://localhost:4430/stderr/942f0ac8-1ec4-4730-b02f-ee4fe8ae521f