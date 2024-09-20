package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/hizkifw/hako/pkg/hako"
)

func main() {
	db, err := hako.NewDB(":memory:")
	if err != nil {
		panic(err)
	}

	err = db.Migrate()
	if err != nil {
		panic(err)
	}

	fs, err := hako.NewLocalFS("/tmp/hako")
	if err != nil {
		panic(err)
	}

	gc := hako.NewGC(db, fs)
	server := hako.NewServer(db, fs)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go server.Run(ctx)
	go gc.LoopForever(ctx)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	cancel()

	<-gc.Done()
	<-server.Done()
}
