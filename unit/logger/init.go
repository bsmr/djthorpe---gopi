/*
  Go Language Raspberry Pi Interface
  (c) Copyright David Thorpe 2016-2020
  All Rights Reserved
  For Licensing and Usage information, please see LICENSE.md
*/

package logger

import (
	"fmt"

	// Frameworks
	gopi "github.com/djthorpe/gopi/v2"
)

func init() {
	gopi.UnitRegister(gopi.UnitConfig{
		Name: "gopi/logger",
		Type: gopi.UNIT_LOGGER,
		Config: func(app gopi.App) error {
			fmt.Println("TODO: Logger App Config")
			return nil
		},
	})
}
