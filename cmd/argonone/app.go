package main

import (
	"context"
	"fmt"

	"github.com/djthorpe/gopi/v3"
	"github.com/djthorpe/gopi/v3/pkg/db/influxdb"
)

type app struct {
	gopi.Unit
	gopi.ArgonOne
	gopi.Command
	gopi.Publisher
	*influxdb.Writer
}

func (this *app) Define(cfg gopi.Config) error {
	cfg.Command("daemon", "Start daemon", this.Serve)
	return nil
}

func (this *app) New(cfg gopi.Config) error {
	if this.Command = cfg.GetCommand(cfg.Args()); this.Command == nil {
		return gopi.ErrHelp
	}
	return nil
}

func (this *app) Run(ctx context.Context) error {
	return this.Command.Run(ctx)
}

func (this *app) Serve(ctx context.Context) error {
	ch := this.Publisher.Subscribe()
	defer this.Publisher.Unsubscribe(ch)

	fmt.Println("Press CTRL+C to end")
	for {
		select {
		case evt := <-ch:
			fmt.Println(evt)
		case <-ctx.Done():
			return nil
		}
	}
}
