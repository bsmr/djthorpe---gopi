// +build linux

/*
  Go Language Raspberry Pi Interface
  (c) Copyright David Thorpe 2016-2020
  All Rights Reserved
  For Licensing and Usage information, please see LICENSE.md
*/

package gpiosysfs

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	// Frameworks
	gopi "github.com/djthorpe/gopi/v2"
	base "github.com/djthorpe/gopi/v2/base"
)

type GPIO struct {
	FilePoll        gopi.FilePoll
	UnexportOnClose bool
}

type gpio struct {
	filepoll gopi.FilePoll
	watched  map[*os.File]gopi.GPIOPin
	exported []gopi.GPIOPin

	base.Unit
	sync.Mutex
}

////////////////////////////////////////////////////////////////////////////////
// CONSTANTS

const (
	GPIO_EXPORT   = "/sys/class/gpio/export"
	GPIO_UNEXPORT = "/sys/class/gpio/unexport"
	GPIO_PIN      = "/sys/class/gpio/gpio%v"
)

////////////////////////////////////////////////////////////////////////////////
// VARIABLES

////////////////////////////////////////////////////////////////////////////////
// IMPLEMENTATION gopi.Unit

func (GPIO) Name() string { return "gopi/gpio/sysfs" }

func (config GPIO) New(log gopi.Logger) (gopi.Unit, error) {
	this := new(gpio)
	if err := this.Unit.Init(log); err != nil {
		return nil, err
	}
	if err := this.Init(config); err != nil {
		return nil, err
	}
	return this, nil
}

////////////////////////////////////////////////////////////////////////////////
// INIT & CLOSE

func (this *gpio) Init(config GPIO) error {
	this.Lock()
	defer this.Unlock()

	if config.FilePoll == nil {
		return gopi.ErrBadParameter.WithPrefix("FilePoll")
	} else {
		this.filepoll = config.FilePoll
	}

	// Check for export and unexport paths
	if _, err := os.Stat(GPIO_EXPORT); os.IsNotExist(err) {
		return err
	}
	if _, err := os.Stat(GPIO_UNEXPORT); os.IsNotExist(err) {
		return err
	}

	// Create a map of watched pins
	this.watched = make(map[*os.File]gopi.GPIOPin, 0)

	// Create an array of exported pins
	if config.UnexportOnClose {
		this.exported = make([]gopi.GPIOPin, 0)
	}

	// Return success
	return nil
}

func (this *gpio) Close() error {
	this.Lock()
	defer this.Unlock()

	// Unwatch pins
	errs := gopi.NewCompoundError()
	for file, _ := range this.watched {
		errs.Add(this.filepoll.Unwatch(file.Fd()))
		errs.Add(file.Close())
	}
	if errs.ErrorOrSelf() != nil {
		return errs.ErrorOrSelf()
	}

	// Unexport pins
	for _, pin := range this.exported {
		if isExported(pin) {
			errs.Add(unexportPin(pin))
		}
	}
	if errs.ErrorOrSelf() != nil {
		return errs.ErrorOrSelf()
	}

	// Release resources
	this.exported = nil
	this.watched = nil
	this.filepoll = nil

	// Return success
	return this.Unit.Close()
}

////////////////////////////////////////////////////////////////////////////////
// STRINGIFY

func (this *gpio) String() string {
	str := "<" + this.Log.Name()
	if this.Unit.Closed {
		return str + ">"
	}
	if this.exported != nil {
		str += " exportedPins=" + fmt.Sprint(this.exported)
	}
	if len(this.watched) > 0 {
		str += " watchedPins=["
		for _, pin := range this.watched {
			str += " " + fmt.Sprint(pin)
		}
		str = str + " ]"
	}
	return str + ">"
}

////////////////////////////////////////////////////////////////////////////////
// PINS

// Return number of physical pins, or 0 if if cannot be returned
// or nothing is known about physical pins
func (this *gpio) NumberOfPhysicalPins() uint {
	return 0
}

// Return array of available logical pins or nil if nothing is
// known about pins
func (this *gpio) Pins() []gopi.GPIOPin {
	return nil
}

// Return logical pin for physical pin number. Returns
// GPIO_PIN_NONE where there is no logical pin at that position
// or we don't know about the physical pins
func (this *gpio) PhysicalPin(pin uint) gopi.GPIOPin {
	return gopi.GPIO_PIN_NONE
}

// Return physical pin number for logical pin. Returns 0 where there
// is no physical pin for this logical pin, or we don't know anything
// about the layout
func (this *gpio) PhysicalPinForPin(logical gopi.GPIOPin) uint {
	return 0
}

// Read pin state
func (this *gpio) ReadPin(logical gopi.GPIOPin) gopi.GPIOState {
	this.Lock()
	defer this.Unlock()

	// Check for pin exported
	if err := this.exportPin(logical); err != nil {
		this.Log.Error(fmt.Errorf("Unable to export %v: %w", logical, err))
		return gopi.GPIO_LOW
	}

	// Read the pin
	if value, err := readPin(logical); err != nil {
		this.Log.Error(fmt.Errorf("Unable to read %v: %w", logical, err))
		return gopi.GPIO_LOW
	} else {
		switch value {
		case "0":
			return gopi.GPIO_LOW
		case "1":
			return gopi.GPIO_HIGH
		default:
			this.Log.Warn(fmt.Errorf("Invalid value for pin %v: %v", logical, value))
			return gopi.GPIO_HIGH
		}
	}
}

// Write pin state
func (this *gpio) WritePin(logical gopi.GPIOPin, state gopi.GPIOState) {
	this.Lock()
	defer this.Unlock()

	// Check for pin exported
	if err := this.exportPin(logical); err != nil {
		this.Log.Error(fmt.Errorf("Unable to export %v: %w", logical, err))
		return
	}

	// Do extra checks of output state when debugging is on
	if this.Log.IsDebug() {
		if direction, err := direction(logical); err != nil {
			this.Log.Warn(fmt.Errorf("Invalid direction for pin %v: %w", logical, err))
		} else if direction != "out" {
			this.Log.Warn(fmt.Errorf("Invalid direction for pin %v: %v", logical, direction))
		}
	}

	// Write pin
	switch state {
	case gopi.GPIO_LOW:
		if err := writePin(logical, "0"); err != nil {
			this.Log.Error(fmt.Errorf("Unable to write value '0' to %v: %w", logical, err))
		}
	case gopi.GPIO_HIGH:
		if err := writePin(logical, "1"); err != nil {
			this.Log.Error(fmt.Errorf("Unable to write value '1' to %v: %w", logical, err))
		}
	}
}

// Get pin mode
func (this *gpio) GetPinMode(logical gopi.GPIOPin) gopi.GPIOMode {
	this.Lock()
	defer this.Unlock()

	// Check for pin exported
	if err := this.exportPin(logical); err != nil {
		this.Log.Error(fmt.Errorf("Unable to export %v: %w", logical, err))
		return gopi.GPIO_NONE
	}

	// Read the pin
	if value, err := direction(logical); err != nil {
		this.Log.Error(fmt.Errorf("Unable to read direction %v: %w", logical, err))
		return gopi.GPIO_NONE
	} else {
		switch value {
		case "in":
			return gopi.GPIO_INPUT
		case "out":
			return gopi.GPIO_OUTPUT
		default:
			this.Log.Warn("Invalid direction for %v: %v", logical, value)
			return gopi.GPIO_NONE
		}
	}
}

// Set pin mode
func (this *gpio) SetPinMode(logical gopi.GPIOPin, mode gopi.GPIOMode) {
	this.Lock()
	defer this.Unlock()

	// Check for pin exported
	if err := this.exportPin(logical); err != nil {
		this.Log.Error(fmt.Errorf("Unable to export %v: %w", logical, err))
		return
	}

	// Write pin
	switch mode {
	case gopi.GPIO_INPUT:
		if err := setDirection(logical, "in"); err != nil {
			this.Log.Error(fmt.Errorf("Unable to write direction to %v: %w", logical, err))
		}
		if err := writeEdge(logical, "none"); err != nil {
			this.Log.Error(fmt.Errorf("Unable to write edge to %v: %w", logical, err))
		}
	case gopi.GPIO_OUTPUT:
		if err := setDirection(logical, "out"); err != nil {
			this.Log.Error(fmt.Errorf("Unable to write direction to %v: %w", logical, err))
		}
	default:
		this.Log.Error(fmt.Errorf("Invalid pin mode %v: %v", logical, mode))
	}
}

// SetPullMode is not implemented in the sysfs driver
func (this *gpio) SetPullMode(logical gopi.GPIOPin, pull gopi.GPIOPull) error {
	return gopi.ErrNotImplemented
}

// Start watching for rising and/or falling edge,
// or stop watching when GPIO_EDGE_NONE is passed.
// Will return ErrNotImplemented if not supported
func (this *gpio) Watch(logical gopi.GPIOPin, edge gopi.GPIOEdge) error {
	this.Lock()
	defer this.Unlock()

	// Check for pin exported
	if err := this.exportPin(logical); err != nil {
		return fmt.Errorf("Unable to export %v: %w", logical, err)
	}

	// Do extra checks of output state when debugging is on
	if this.Log.IsDebug() {
		if direction, err := direction(logical); err != nil {
			this.Log.Warn(fmt.Errorf("Invalid direction for pin %v: %w", logical, err))
		} else if direction != "out" {
			this.Log.Warn(fmt.Errorf("Invalid direction for pin %v: %v", logical, direction))
		}
	}
	/*
		// Set rising, falling, both or none
		edge_write := ""
		switch edge {
		case gopi.GPIO_EDGE_NONE:
			if err := writeEdge(logical, "none"); err != nil {
				return fmt.Errorf("Unable to write edge for %v: %w", pin, err)
			} else if file, exists := this.watched[pin]; exists == false {
				// IGNORE UNWATCHED PINS
			} else if err := this.filepoll.Unwatch(file.Fd()); err != nil {
				this.log.Error("%v: %v", pin, err)
				file.Close()
			} else if err := file.Close(); err != nil {
				return err
			} else {
				// Remove from list of watched pins
				delete(this.watched, pin)
			}
		case gopi.GPIO_EDGE_RISING:
			edge_write = "rising"
		case gopi.GPIO_EDGE_FALLING:
			edge_write = "falling"
		case gopi.GPIO_EDGE_BOTH:
			edge_write = "both"
		default:
			return errors.New("Watch: Invalid edge value")
		}

		if edge_write != "" {
			if err := writeEdge(pin, edge_write); err != nil {
				this.log.Error("Watch: Unable to write edge for %v: %v", pin, err)
				return err
			} else if _, exists := this.watched[pin]; exists {
				// IGNORE ALREADY WATCHED PINS
			} else if file, err := watchValue(pin); err != nil {
				this.log.Error("Watch: Unable to watch %v: %v", pin, err)
				return err
			} else if err := this.filepoll.Watch(file, filepoll.FILEPOLL_MODE_EDGE, func(handle *os.File, mode filepoll.FilePollMode) {
				if err := this.handleEdge(handle, pin); err != nil {
					this.log.Warn("Watch: %v: %v", pin, err)
				}
			}); err != nil {
				this.log.Error("Watch: %v: %v", pin, err)
				file.Close()
				return err
			} else {
				this.watched[pin] = file
			}
		}
	*/

	return nil
}

////////////////////////////////////////////////////////////////////////////////
// PRIVATE METHODS

func (this *gpio) exportPin(pin gopi.GPIOPin) error {
	if isExported(pin) == false {
		this.Log.Debug("Export Pin", pin)
		if err := exportPin(pin); err != nil {
			return err
		}
	}
	if this.exported != nil {
		for _, exported := range this.exported {
			if pin == exported {
				return nil
			}
		}
		this.exported = append(this.exported, pin)
	}
	return nil
}

func filenameForPin(pin gopi.GPIOPin, filename string) string {
	return filepath.Join(fmt.Sprintf(GPIO_PIN, uint(pin)), filename)
}

func writeFile(filename string, value string) error {
	return ioutil.WriteFile(filename, []byte(value), 777)
}

func readFile(filename string) (string, error) {
	if bytes, err := ioutil.ReadFile(filename); err != nil {
		return "", err
	} else {
		return string(bytes), nil
	}
}

func isExported(pin gopi.GPIOPin) bool {
	if _, err := os.Stat(filenameForPin(pin, "")); os.IsNotExist(err) {
		return false
	} else if err != nil {
		return false
	} else {
		return true
	}
}

func exportPin(pin gopi.GPIOPin) error {
	if err := writeFile(GPIO_EXPORT, strconv.FormatUint(uint64(pin), 10)+"\n"); err != nil {
		return err
	}
	// Update permissions on the pin
	if _, err := os.Stat(filenameForPin(pin, "")); os.IsNotExist(err) {
		return err
	}
	if err := os.Chmod(filenameForPin(pin, "direction"), 0770); err != nil {
		return err
	}
	if err := os.Chmod(filenameForPin(pin, "value"), 0770); err != nil {
		return err
	}
	if err := os.Chmod(filenameForPin(pin, "edge"), 0770); err != nil {
		return err
	}
	// Return success
	return nil
}

func unexportPin(pin gopi.GPIOPin) error {
	return writeFile(GPIO_UNEXPORT, strconv.FormatUint(uint64(pin), 10)+"\n")
}

func direction(pin gopi.GPIOPin) (string, error) {
	if value, err := readFile(filenameForPin(pin, "direction")); err != nil {
		return "", err
	} else {
		return strings.TrimSpace(value), nil
	}
}

func setDirection(pin gopi.GPIOPin, value string) error {
	return writeFile(filenameForPin(pin, "direction"), value+"\n")
}

func readPin(pin gopi.GPIOPin) (string, error) {
	if value, err := readFile(filenameForPin(pin, "value")); err != nil {
		return "", err
	} else {
		return strings.TrimSpace(value), nil
	}
}

func writePin(pin gopi.GPIOPin, value string) error {
	return writeFile(filenameForPin(pin, "value"), value+"\n")
}

func writeEdge(pin gopi.GPIOPin, edge string) error {
	return writeFile(filenameForPin(pin, "edge"), edge+"\n")
}

func readEdge(pin gopi.GPIOPin) (string, error) {
	if value, err := readFile(filenameForPin(pin, "edge")); err != nil {
		return "", err
	} else {
		return strings.TrimSpace(value), nil
	}
}

func watchValue(pin gopi.GPIOPin) (*os.File, error) {
	return os.OpenFile(filenameForPin(pin, "value"), os.O_RDONLY, 0)
}