The goal is to test the library client implementation.

Change directory to `cmd/autobahn` to run the server from the docker image `crossbario/autobahn-testsuite` with the command:
```
docker run -it --rm \
    -v "${PWD}/config:/config" \
    -v "${PWD}/reports:/reports" \
    -p 9001:9001 \
    --name fuzzingserver \
    crossbario/autobahn-testsuite
```
Then run in parallel the client with `go run . -m client`.