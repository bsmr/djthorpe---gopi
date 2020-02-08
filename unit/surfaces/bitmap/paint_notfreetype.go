// +build rpi,!freetype

/*
  Go Language Raspberry Pi Interface
  (c) Copyright David Thorpe 2016-2020
  All Rights Reserved
  For Licensing and Usage information, please see LICENSE.md
*/

package bitmap

import (
	"image/color"

	// Frameworks
	gopi "github.com/djthorpe/gopi/v2"
)

////////////////////////////////////////////////////////////////////////////////
// RUNE

func (this *bitmap) PaintRune(color.Color, gopi.Point, rune, gopi.FontFace, gopi.FontSize) {
	this.Log.Error(gopi.ErrNotImplemented.WithPrefix("PaintRune"))
}
