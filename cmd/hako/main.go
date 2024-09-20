package main

import (
	"github.com/hizkifw/hako/pkg/hako"
	"go.uber.org/fx"
)

func main() {
	fx.New(
		fx.Provide(hako.ConfigFromEnv),
		fx.Provide(hako.FxNewDB),
		fx.Provide(hako.FxNewLocalFS),
		fx.Provide(hako.FxNewGC),
		fx.Provide(hako.FxNewServer),
		fx.Invoke(func(db *hako.DB) {
			db.Migrate()
		}),
		fx.Invoke(func(*hako.Server, *hako.GC) {}),
	).Run()
}
