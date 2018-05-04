/*
	Go Language Raspberry Pi Interface
	(c) Copyright David Thorpe 2016-2018
	All Rights Reserved
	Documentation http://djthorpe.github.io/gopi/
	For Licensing and Usage information, please see LICENSE.md
*/

package gopi

import (
	"time"
)

// Metrics returns various system and application
// metrics
type Metrics interface {
	Driver

	UptimeHost() time.Duration
	UptimeApp() time.Duration
}
