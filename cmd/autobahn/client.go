package main

import "context"

func testClient(ctx context.Context) {
	<-ctx.Done()
}
