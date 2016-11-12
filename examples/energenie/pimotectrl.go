/*
	Go Language Raspberry Pi Interface
	(c) Copyright David Thorpe 2016
	All Rights Reserved

	For Licensing and Usage information, please see LICENSE.md
*/

// PIMOTECTRL
//
// This sample program shows how you can control an Energenie Pimote 4-gang
// socket. On the command line, the flags to control are either:
//
//   pimote_example -socket 1 -on
//   pimote_example -socket 1 -off
//
// or to switch all sockets at the same time omit the -socket flag. The sockets
// are numbered from 1 to 4.
//
package main

import (
	"flag"
	"fmt"
	"os"
)

import (
	gopi "github.com/djthorpe/gopi"
	app "github.com/djthorpe/gopi/app"
	energenie "github.com/djthorpe/gopi/device/energenie"
)

////////////////////////////////////////////////////////////////////////////////

func RunLoop(app *app.App) error {

	// Create the Pimote interface
	pimote, err := gopi.Open(energenie.Pimote{GPIO: app.GPIO}, app.Logger)
	if err != nil {
		return err
	}
	defer pimote.Close()

	// Get the socket and state (on or off)
	socket, _ := app.FlagSet.GetUint("socket")
	on, _ := app.FlagSet.GetBool("on")
	off, _ := app.FlagSet.GetBool("off")

	if on == off {
		return app.Logger.Error("Invalid flag combination, use either -on or -off")
	}

	if on {
		if socket != 0 {
			err = pimote.(*energenie.PimoteDriver).On(socket)
		} else {
			err = pimote.(*energenie.PimoteDriver).On()
		}
	} else {
		if socket != 0 {
			err = pimote.(*energenie.PimoteDriver).Off(socket)
		} else {
			err = pimote.(*energenie.PimoteDriver).Off()
		}
	}
	return err
}

////////////////////////////////////////////////////////////////////////////////

func main() {
	// Create the config
	config := app.Config(app.APP_GPIO)

	// Add on command-line flags
	config.FlagSet.FlagUint("socket", 0, "Socket number (1,2,3 or 4). If not specified, all sockets are controlled")
	config.FlagSet.FlagBool("on", false, "Switch socket on")
	config.FlagSet.FlagBool("off", false, "Switch socket off")

	// Create the application
	myapp, err := app.NewApp(config)
	if err == flag.ErrHelp {
		return
	} else if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		return
	}
	defer myapp.Close()

	// Run the application
	if err := myapp.Run(RunLoop); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		return
	}
}