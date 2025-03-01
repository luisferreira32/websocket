package main

import "context"

func testServer(ctx context.Context) {
	<-ctx.Done()
}
