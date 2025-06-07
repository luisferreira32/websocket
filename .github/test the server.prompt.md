The goal is to test the library server implementation.

Change directory to `cmd/autobahn` to run the server with `go run . -m server`.
Then run in parallel the client from the docker image `crossbario/autobahn-testsuite` with the command:
```
docker run -it --rm \
    --add-host=host.docker.internal:host-gateway \
    -v ${PWD}/config:/config \
    -v ${PWD}/reports:/reports \
    crossbario/autobahn-testsuite \
    wstest -m fuzzingclient -s /config/fuzzingclient.json
```

The report can be found under `reports/servers`, it is expected that the index.json file is populated with test cases with passing.
