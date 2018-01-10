// +build rpi

/*
  Go Language Raspberry Pi Interface
  (c) Copyright David Thorpe 2016-2017
  All Rights Reserved

  Documentation http://djthorpe.github.io/gopi/
  For Licensing and Usage information, please see LICENSE.md
*/

package rpi

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/djthorpe/gopi"
)

////////////////////////////////////////////////////////////////////////////////
// TYPES

type EGL struct {
	Display gopi.Display
}

type egl struct {
	log          gopi.Logger
	display      gopi.Display
	handle       eglDisplay
	update       dxUpdateHandle
	lock         sync.Mutex
	major, minor int
}

type surface struct {
	handle eglSurface
}

// Raspberry-pi specific interface for SurfaceManager
type SurfaceManager interface {
	gopi.SurfaceManager

	// Return a list of extensions the GPU provides
	Extensions() []string
}

////////////////////////////////////////////////////////////////////////////////
// GLOBAL VARIABLES

var (
	// Map API names to eglAPI
	eglStringTypeMap = map[string]gopi.SurfaceType{
		"OpenGL_ES": gopi.SURFACE_TYPE_OPENGL_ES,
		"OpenVG":    gopi.SURFACE_TYPE_OPENVG,
	}
	// Map eglAPI to EGL_RENDERABLE_TYPE
	eglRenderableTypeMap = map[eglAPI]eglRenderableType{
		EGL_OPENGL_API:    EGL_OPENGL_BIT,
		EGL_OPENVG_API:    EGL_OPENVG_BIT,
		EGL_OPENGL_ES_API: EGL_OPENGL_ES_BIT,
	}
)

////////////////////////////////////////////////////////////////////////////////
// OPEN AND CLOSE

func (config EGL) Open(log gopi.Logger) (gopi.Driver, error) {
	log.Debug("<sys.surface.rpi.SurfaceManager.Open>{ Display=%v }", config.Display)
	this := new(egl)
	this.log = log

	// Check display
	this.display = config.Display
	if this.display == nil {
		return nil, gopi.ErrBadParameter
	}

	// Initialize EGL
	n := to_eglNativeDisplayType(this.display.Display())
	if handle, err := eglGetDisplay(n); err != EGL_SUCCESS {
		return nil, os.NewSyscallError("eglGetDisplay", err)
	} else {
		this.handle = handle
	}
	if major, minor, err := eglInitialize(this.handle); err != EGL_SUCCESS {
		return nil, os.NewSyscallError("eglInitialize", err)
	} else {
		this.major = int(major)
		this.minor = int(minor)
	}

	/*
		// Get configurations
		if configs, err := eglGetConfigs(this.handle); err != EGL_SUCCESS {
			return nil, os.NewSyscallError("eglGetConfigs", err)
		} else {
			for i, config := range configs {
				if a, err := eglGetConfigAttribs(this.handle, config); err != EGL_SUCCESS {
					return nil, os.NewSyscallError("eglGetConfigAttribs", err)
				} else {
					fmt.Println(i, a)
				}
			}
		}
	*/

	this.getFrameBufferConfiguration()

	return this, nil
}

func (this *egl) Close() error {
	this.log.Debug("<sys.surface.rpi.SurfaceManager.Close>{ Display=%v }", this.display)
	if this.display == nil {
		return nil
	}
	if err := eglTerminate(this.handle); err != EGL_SUCCESS {
		return os.NewSyscallError("Close", err)
	} else {
		this.display = nil
		this.handle = eglDisplay(EGL_NO_DISPLAY)
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////
// DO

func (this *egl) Do(callback gopi.SurfaceManagerCallback) error {
	// check parameters
	if this.handle == eglDisplay(EGL_NO_DISPLAY) {
		return ErrInvalidDisplay
	}
	// create update
	if err := this.doUpdateStart(); err != nil {
		return err
	}
	// callback
	cb_err := callback(this)
	// end update
	if err := this.doUpdateEnd(); err != nil {
		this.log.Error("doUpdateEnd: %v", err)
	}
	// return callback error
	return cb_err
}

func (this *egl) doUpdateStart() error {
	this.lock.Lock()
	defer this.lock.Unlock()
	if this.update != dxUpdateHandle(DX_NO_UPDATE) {
		return ErrInvalidState
	}
	if update, err := dxUpdateStart(DX_UPDATE_PRIORITY_DEFAULT); err != DX_SUCCESS {
		return os.NewSyscallError("dxUpdateStart", err)
	} else {
		this.update = update
		return nil
	}
}

func (this *egl) doUpdateEnd() error {
	this.lock.Lock()
	defer this.lock.Unlock()
	if this.update == dxUpdateHandle(DX_NO_UPDATE) {
		return ErrInvalidState
	}
	if err := dxUpdateSubmitSync(this.update); err != DX_SUCCESS {
		return os.NewSyscallError("doUpdateEnd", err)
	} else {
		this.update = update
		return nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// SURFACE

func (this *egl) CreateSurface(api gopi.SurfaceType, flags gopi.SurfaceFlags, opacity float32, layer uint16, origin gopi.Point, size gopi.Size) (gopi.Surface, error) {
	// Currently we only support RGBA32 surfaces
	if api != gopi.SURFACE_TYPE_RGBA32 {
		return nil, gopi.ErrNotImplemented
	}

	// Create bitmap (in the case of RGBA32)
	if bitmap, err := this.CreateBitmap(api,size); err != nil {
		return nil, err
	} else if element, err := dxElementAdd(this.update,this.display,layer,&dest_rect,bitmap.Resource(),&src_rect,DX_PROTECTION_NONE,alpha,transform) {
		this.DestroyBitmap(bitmap)
		return nil, err
	} else surface, err := gopi.Open(Element{}, this.log); err != nil {
		this.DestroyBitmap(bitmap)
		return nil, err		
	} else {
		return element.(gopi.Surface), nil
	}
}

func (this *egl) CreateSurfaceWithBitmap(gopi.Bitmap, flags gopi.SurfaceFlags, opacity float32, layer uint16, origin gopi.Point, size gopi.Size) (gopi.Surface, error) {

}

func (this *egl) DestroySurface(surface gopi.Surface) error {
	return surface.Close()
}

// SetLayer changes a surface layer (except if it's a background or cursor). Currently
// the flags argument is ignored
func (this *egl) SetLayer(surface gopi.Surface,flags gopi.SurfaceFlags, layer uint16) error {	
	if layer == gopi.SURFACE_LAYER_BACKGROUND || layer > gopi.SURFACE_LAYER_MAX {
		return gopi.ErrBadParameter
	}
	if err := dxElementChangeLayer(this.update,surface.(rpi.Surface).Handle(),int32(layer); err != DX_SUCCESS {
		return os.NewSyscallError("dxElementChangeLayer",err)
	} else {
		return nil
	}
}

// SetOrigin moves the surface. Currently the flags argument is ignored
func (this *egl) SetOrigin(surface gopi.Surface,flags gopi.SurfaceFlags,origin gopi.Point) error {
	
}

//SetOpacity(Surface, SurfaceFlags, float32)


////////////////////////////////////////////////////////////////////////////////
// BITMAP

func (this *egl) CreateBitmap(api gopi.SurfaceType,size gopi.Size) (gopi.Bitmap, error) {
	// Currently we only support RGBA32 surfaces
	if api != gopi.SURFACE_TYPE_RGBA32 {
		return nil, gopi.ErrNotImplemented
	}

	if bitmap, err := gopi.Open(Resource{
		ImageType: DX_IMAGETYPE_RGBA32,
		Width:     uint32(size.W),
		Height:    uint32(size.H),
	}, this.log); err != nil {
		return nil, err
	} else {
		return bitmap.(gopi.Bitmap), nil
	}
}

func (this *egl) DestroyBitmap(bitmap gopi.Bitmap) error {
	return bitmap.Close()
}

////////////////////////////////////////////////////////////////////////////////
// STRINGIFY

func (this *egl) String() string {
	if this.display == nil {
		return fmt.Sprintf("<sys.surface.rpi.SurfaceManager>{ nil }")
	} else {
		return fmt.Sprintf("<sys.surface.rpi.SurfaceManager>{ handle=%v name=%v version={ %v,%v } types=%v extensions=%v display=%v }", this.handle, this.Name(), this.major, this.minor, this.Types(), this.Extensions(), this.display)
	}
}

////////////////////////////////////////////////////////////////////////////////
// INTERFACE

func (this *egl) Display() gopi.Display {
	return this.display
}

func (this *egl) Name() string {
	return fmt.Sprintf("%v %v", eglQueryString(this.handle, EGL_VENDOR), eglQueryString(this.handle, EGL_VERSION))
}

func (this *egl) Extensions() []string {
	return strings.Split(eglQueryString(this.handle, EGL_EXTENSIONS), " ")
}

// Return capabilities for the GPU
func (this *egl) Types() []gopi.SurfaceType {
	types := strings.Split(eglQueryString(this.handle, EGL_CLIENT_APIS), " ")
	surface_types := make([]gopi.SurfaceType, 0, 3)
	for _, t := range types {
		if t2, ok := eglStringTypeMap[t]; ok {
			surface_types = append(surface_types, t2)
		}
	}
	// always include RGBA32
	return append(surface_types, gopi.SURFACE_TYPE_RGBA32)
}

////////////////////////////////////////////////////////////////////////////////

func (this *egl) getFrameBufferConfiguration() (eglConfig, error) {
	attribute_list := map[eglConfigAttrib]eglInt{
		EGL_RED_SIZE:     eglInt(8),
		EGL_BLUE_SIZE:    eglInt(8),
		EGL_GREEN_SIZE:   eglInt(8),
		EGL_ALPHA_SIZE:   eglInt(8),
		EGL_SURFACE_TYPE: eglInt(EGL_WINDOW_BIT),
	}

	// RENDERABLE_TYPE
	api := eglQueryAPI()
	if rednerable_type, ok := eglRenderableTypeMap[api]; ok {
		attribute_list[EGL_RENDERABLE_TYPE] = eglInt(rednerable_type)
	}

	// Configs
	if configs, err := eglChooseConfig(this.handle, attribute_list); err != EGL_SUCCESS {
		return EGL_NO_CONFIG, os.NewSyscallError("eglChooseConfig", err)
	} else if len(configs) == 0 {
		return EGL_NO_CONFIG, errors.New("Matches several configurations")
	} else {
		this.log.Info("Configs = %v", configs)
		return configs[0], nil
	}
}
