# Autobahn test setup for websocket compliance

## Test client implementation

Start the server:

```
go run . -m server
```

Then run the test:

```
docker run -it --rm \
    -v ${PWD}/config:/config \
    -v ${PWD}/reports:/reports \
    crossbario/autobahn-testsuite \
    wstest -m fuzzingclient -s /config/fuzzingclient.json
```

## Test server implementation

Start the test server:

```
docker run -it --rm \
    -v "${PWD}/config:/config" \
    -v "${PWD}/reports:/reports" \
    -p 9001:9001 \
    --name fuzzingserver \
    crossbario/autobahn-testsuite
```

Then run the client:

```
go run . -m client
```
