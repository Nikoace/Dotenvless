package privacy

import (
	"errors"
	"golang.org/x/sys/windows"
	"runtime"
	"time"
	"unsafe"
)

var user32 = windows.NewLazySystemDLL("user32.dll")
var kernel32 = windows.NewLazySystemDLL("kernel32.dll")
var openClip = user32.NewProc("OpenClipboard")
var closeClip = user32.NewProc("CloseClipboard")
var emptyClip = user32.NewProc("EmptyClipboard")
var setClip = user32.NewProc("SetClipboardData")
var registerClip = user32.NewProc("RegisterClipboardFormatW")
var clipSequence = user32.NewProc("GetClipboardSequenceNumber")
var globalAlloc = kernel32.NewProc("GlobalAlloc")
var globalLock = kernel32.NewProc("GlobalLock")
var globalUnlock = kernel32.NewProc("GlobalUnlock")
var globalFree = kernel32.NewProc("GlobalFree")
var moveMemory = kernel32.NewProc("RtlMoveMemory")

type nativeClipboard struct{ handle func() uintptr }

func NewNativeClipboard(handle func() uintptr) Clipboard { return &nativeClipboard{handle: handle} }
func openClipboard(owner uintptr) error {
	for n := 0; n < 5; n++ {
		if ok, _, _ := openClip.Call(owner); ok != 0 {
			return nil
		}
		time.Sleep(20 * time.Millisecond)
	}
	return errors.New("clipboard is busy; retry")
}
func putClipboard(format uintptr, data []byte) error {
	if len(data) == 0 {
		return errors.New("invalid clipboard data")
	}
	h, _, _ := globalAlloc.Call(0x0002, uintptr(len(data)))
	if h == 0 {
		return errors.New("cannot allocate clipboard data")
	}
	p, _, _ := globalLock.Call(h)
	if p == 0 {
		globalFree.Call(h)
		return errors.New("cannot prepare clipboard data")
	}
	moveMemory.Call(p, uintptr(unsafe.Pointer(&data[0])), uintptr(len(data)))
	globalUnlock.Call(h)
	if r, _, _ := setClip.Call(format, h); r == 0 {
		globalFree.Call(h)
		return errors.New("cannot set clipboard data")
	}
	return nil
}
func (c *nativeClipboard) SetSensitive(value []byte) (uint32, error) {
	owner := c.handle()
	if owner == 0 {
		return 0, errors.New("native window is unavailable")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if e := openClipboard(owner); e != nil {
		return 0, e
	}
	defer closeClip.Call()
	if r, _, _ := emptyClip.Call(); r == 0 {
		return 0, errors.New("cannot prepare clipboard")
	}
	success := false
	defer func() {
		if !success {
			emptyClip.Call()
		}
	}()
	for _, name := range []string{"CanIncludeInClipboardHistory", "CanUploadToCloudClipboard", "ExcludeClipboardContentFromMonitorProcessing"} {
		ptr, _ := windows.UTF16PtrFromString(name)
		format, _, _ := registerClip.Call(uintptr(unsafe.Pointer(ptr)))
		if format == 0 {
			return 0, errors.New("cannot protect clipboard history")
		}
		if e := putClipboard(format, []byte{0, 0, 0, 0}); e != nil {
			return 0, e
		}
	}
	text, e := windows.UTF16FromString(string(value))
	if e != nil {
		return 0, errors.New("invalid clipboard value")
	}
	defer clear(text)
	data := unsafe.Slice((*byte)(unsafe.Pointer(&text[0])), len(text)*2)
	if e = putClipboard(13, data); e != nil {
		return 0, e
	}
	seq, _, _ := clipSequence.Call()
	success = true
	return uint32(seq), nil
}
func (c *nativeClipboard) ClearOwned(seq uint32) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if e := openClipboard(0); e != nil {
		return e
	}
	defer closeClip.Call()
	current, _, _ := clipSequence.Call()
	if uint32(current) != seq {
		return nil
	}
	if r, _, _ := emptyClip.Call(); r == 0 {
		return errors.New("cannot clear sensitive clipboard")
	}
	return nil
}
