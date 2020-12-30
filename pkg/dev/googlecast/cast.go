package googlecast

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/djthorpe/gopi/v3"
	"github.com/hashicorp/go-multierror"
)

////////////////////////////////////////////////////////////////////////////////
// TYPES

type Cast struct {
	sync.RWMutex
	connection

	// Data about the cast device
	id, fn string
	md, rs string
	st     uint
	ips    []net.IP
	port   uint16

	// State information
	volume *Volume
	app    *App
}

////////////////////////////////////////////////////////////////////////////////
// GLOBALS

////////////////////////////////////////////////////////////////////////////////
// LIFECYCLE

func NewCastFromRecord(r gopi.ServiceRecord) *Cast {
	this := new(Cast)

	// Set addr and port
	if port := r.Port(); port == 0 {
		return nil
	} else {
		this.port = port
	}
	if ips := r.Addrs(); len(ips) == 0 {
		return nil
	} else {
		this.ips = ips
	}

	// Set properties
	tuples := txtToMap(r.Txt())
	if id, exists := tuples["id"]; exists && id != "" {
		this.id = id
	} else {
		return nil
	}
	if fn, exists := tuples["fn"]; exists && fn != "" {
		this.fn = fn
	} else {
		this.fn = this.id
	}
	if md, exists := tuples["md"]; exists {
		this.md = md
	}
	if rs, exists := tuples["rs"]; exists {
		this.rs = rs
	}
	if st, exists := tuples["st"]; exists {
		if st, err := strconv.ParseUint(st, 0, 64); err == nil {
			this.st = uint(st)
		}
	}

	return this
}

func (this *Cast) ConnectWithTimeout(timeout time.Duration, errs chan<- error, state chan<- state) error {
	// Get an address to connect to
	if len(this.ips) == 0 {
		return gopi.ErrNotFound.WithPrefix("ConnectWithTimeout", "No Address")
	}

	// Use first IP
	// TODO: Use a random IP and retry with other IP's if not working
	addr := fmt.Sprintf("%v:%v", this.ips[0], this.port)
	if err := this.connection.Connect(this.Id(), addr, timeout, errs, state); err != nil {
		return err
	}

	// Lock for setting state
	this.RWMutex.Lock()
	defer this.RWMutex.Unlock()

	// Set state
	this.volume = nil
	this.app = nil

	// Return success
	return nil
}

func (this *Cast) Disconnect() error {
	this.RWMutex.Lock()
	defer this.RWMutex.Unlock()

	var result error

	// Disconnect
	if err := this.connection.Disconnect(); err != nil {
		result = multierror.Append(result, err)
	}

	// Set state
	this.volume = nil
	this.app = nil

	return result
}

////////////////////////////////////////////////////////////////////////////////
// PROPERTIES

// Id returns the identifier for a chromecast
func (this *Cast) Id() string {
	this.RWMutex.RLock()
	defer this.RWMutex.RUnlock()
	return this.id
}

// Name returns the readable name for a chromecast
func (this *Cast) Name() string {
	this.RWMutex.RLock()
	defer this.RWMutex.RUnlock()
	return this.fn
}

// Model returns the reported model information
func (this *Cast) Model() string {
	this.RWMutex.RLock()
	defer this.RWMutex.RUnlock()
	return this.md
}

// Service returns the currently running service
func (this *Cast) Service() string {
	this.RWMutex.RLock()
	defer this.RWMutex.RUnlock()
	if this.app != nil && this.app.DisplayName != "" {
		return this.app.DisplayName
	} else {
		return this.rs
	}
}

// State returns 0 if backdrop, else returns 1
func (this *Cast) State() uint {
	this.RWMutex.RLock()
	defer this.RWMutex.RUnlock()
	return this.st
}

////////////////////////////////////////////////////////////////////////////////
// STATE

func (this *Cast) UpdateStatus() error {
	this.RWMutex.RLock()
	defer this.RWMutex.RUnlock()

	if this.volume == nil || this.app == nil {
		if _, data, err := this.channel.GetStatus(); err != nil {
			return err
		} else if err := this.send(data); err != nil {
			return err
		}
	}

	// Return success
	return nil
}

func (this *Cast) SetVolume(v Volume) (gopi.CastFlag, error) {
	this.RWMutex.Lock()
	defer this.RWMutex.Unlock()

	if this.volume == nil || this.volume.Equals(v) == false {
		this.volume = &v
		return gopi.CAST_FLAG_VOLUME, nil
	} else {
		return gopi.CAST_FLAG_NONE, nil
	}
}

func (this *Cast) SetApp(a App) (gopi.CastFlag, error) {
	this.RWMutex.Lock()
	defer this.RWMutex.Unlock()

	if this.app == nil || this.app.Equals(a) == false {
		this.app = &a
		return gopi.CAST_FLAG_APP, nil
	} else {
		return gopi.CAST_FLAG_NONE, nil
	}
}

func (this *Cast) SetMedia(m Media) (gopi.CastFlag, error) {
	this.RWMutex.Lock()
	defer this.RWMutex.Unlock()

	fmt.Println("TODO:MEDIA", m)
	return gopi.CAST_FLAG_NONE, nil
}

////////////////////////////////////////////////////////////////////////////////
// PUBLIC METHODS - REQUESTS

func (this *Cast) ReqLaunchAppWithId(appId string) error {
	this.RWMutex.RLock()
	defer this.RWMutex.RUnlock()

	if _, data, err := this.channel.LaunchAppWithId(appId); err != nil {
		return err
	} else if err := this.send(data); err != nil {
		return err
	}

	// Return success
	return nil
}

func (this *Cast) ReqVolumeLevel(level float32) error {
	this.RWMutex.RLock()
	defer this.RWMutex.RUnlock()

	// Clamp value between 0.0 and 1.0
	if level < 0.0 {
		level = 0.0
	} else if level > 1.0 {
		level = 1.0
	}
	v := Volume{level, false}
	if level == 0 {
		v = Volume{0, true}
	}

	if _, data, err := this.channel.SetVolume(v); err != nil {
		return err
	} else if err := this.send(data); err != nil {
		return err
	}

	// Return success
	return nil
}

func (this *Cast) ReqMuted(muted bool) error {
	this.RWMutex.RLock()
	defer this.RWMutex.RUnlock()

	if _, data, err := this.channel.SetMuted(muted); err != nil {
		return err
	} else if err := this.send(data); err != nil {
		return err
	}

	// Return success
	return nil
}

func (this *Cast) ReqLoadURL(url *url.URL, mimetype string, autoplay bool) error {
	// ConnectMedia
	if this.app == nil || this.app.TransportId == "" {
		return gopi.ErrOutOfOrder.WithPrefix("transportId")
	} else if _, data, err := this.channel.ConnectMedia(this.app.TransportId); err != nil {
		return err
	} else if err := this.send(data); err != nil {
		return err
	}

	// LoadUrl
	if _, data, err := this.channel.LoadUrl(this.app.TransportId, url.String(), mimetype, autoplay); err != nil {
		return err
	} else if err := this.send(data); err != nil {
		return err
	}

	// Return success
	return nil
}

////////////////////////////////////////////////////////////////////////////////
// STRINGIFY

func (this *Cast) String() string {
	this.RWMutex.RLock()
	defer this.RWMutex.RUnlock()

	str := "<cast.device"
	str += " id=" + this.Id()
	if name := this.Name(); name != "" {
		str += " name=" + strconv.Quote(name)
	}
	if model := this.Model(); model != "" {
		str += " model=" + strconv.Quote(model)
	}
	if service := this.Service(); service != "" {
		str += " service=" + strconv.Quote(service)
	}
	str += " state=" + fmt.Sprint(this.State())
	if this.volume != nil {
		str += " volume=" + fmt.Sprint(this.volume)
	}
	if this.app != nil {
		str += " app=" + fmt.Sprint(this.app)
	}
	return str + ">"
}

////////////////////////////////////////////////////////////////////////////////
// PRIVATE METHODS

func txtToMap(txt []string) map[string]string {
	result := make(map[string]string, len(txt))
	for _, r := range txt {
		if kv := strings.SplitN(r, "=", 2); len(kv) == 2 {
			result[kv[0]] = kv[1]
		} else if len(kv) == 1 {
			result[kv[0]] = ""
		}
	}
	return result
}
