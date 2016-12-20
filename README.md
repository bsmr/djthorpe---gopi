
https://github.com/djthorpe/gopi/tree/master/etc/images/gopi-800x388.png "gopi"

This repository contains a golang framework for the Raspberry Pi, which
will allow you to develop applications which utilize the Graphics Processing
Unit (GPU) for image and video encoding/decoding and 2D and 3D graphics,
and various external hardware devices like mouse, keyboard, touchscreen,
GPIO, I2C and Camera.

More information on usage is available at http://djthorpe.github.io/gopi/

# Requirements

The tested requirements are currently:

  * Any Raspberry Pi
  * Raspian Jessie Lite 4.4 (other distributions may work, but not tested
  * Go 1.6

In order to use the library, you'll need to have a working version of Go on 
your Raspberry Pi, which you can [download](https://golang.org/dl/). Then 
retrieve the library on your device, using:

```
go get github.com/djthorpe/gopi
```

# License

```
Copyright 2016 David Thorpe All Rights Reserved

Redistribution and use in source and binary forms, with or without 
modification, are permitted with some conditions. 
```

This repository is released under the BSD License. Please see the file
[LICENSE](LICENSE.md) for a copy of this license and for a list of the
conditions for redistribution and use.

