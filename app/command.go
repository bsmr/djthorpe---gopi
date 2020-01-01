/*
  Go Language Raspberry Pi Interface
  (c) Copyright David Thorpe 2016-2020
  All Rights Reserved
  For Licensing and Usage information, please see LICENSE.md
*/

package app

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	// Frameworks
	gopi "github.com/djthorpe/gopi/v2"
	base "github.com/djthorpe/gopi/v2/base"
)

////////////////////////////////////////////////////////////////////////////////
// INTERFACES

type command struct {
	main gopi.MainCommandFunc
	base.App
}

////////////////////////////////////////////////////////////////////////////////
// gopi.App implementation for command-line tool

func NewCommandLineTool(main gopi.MainCommandFunc, units ...string) (gopi.App, error) {
	this := new(command)

	// Name of command
	name := filepath.Base(os.Args[0])

	// Check parameters
	if main == nil {
		return nil, gopi.ErrBadParameter.WithPrefix("main")
	} else if err := this.App.Init(name, units); err != nil {
		return nil, err
	} else {
		this.main = main
	}

	// Success
	return this, nil
}

func (this *command) Run() int {
	if returnValue := this.App.Run(); returnValue != 0 {
		return returnValue
	}

	// Defer closing of instances to exit
	defer func() {
		if err := this.App.Close(); err != nil {
			fmt.Fprintln(os.Stderr, this.App.Flags().Name()+":", err)
		}
	}()

	// Run main function
	if err := this.main(this, this.Flags().Args()); errors.Is(err, gopi.ErrHelp) || errors.Is(err, flag.ErrHelp) {
		this.App.Flags().Usage(os.Stderr)
		return 0
	} else if err != nil {
		fmt.Fprintln(os.Stderr, this.App.Flags().Name()+":", err)
		return -1
	}

	// Success
	return 0
}
