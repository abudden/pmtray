package ui

import (
	"math"
	"time"
	"unsafe"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver"
	"golang.org/x/sys/windows"
)

var (
	user32               = windows.NewLazySystemDLL("user32.dll")
	procGetCursorPos     = user32.NewProc("GetCursorPos")
	procMonitorFromPoint = user32.NewProc("MonitorFromPoint")
	procGetMonitorInfo   = user32.NewProc("GetMonitorInfoW")
	procGetWindowRect    = user32.NewProc("GetWindowRect")
	procSetWindowPos     = user32.NewProc("SetWindowPos")
	procSetForeground    = user32.NewProc("SetForegroundWindow")
	procGetDpiForWindow  = user32.NewProc("GetDpiForWindow")
)

type point struct{ X, Y int32 }

type rect struct{ Left, Top, Right, Bottom int32 }

type monitorInfo struct {
	Size    uint32
	Monitor rect
	Work    rect
	Flags   uint32
}

const (
	monitorDefaultToNearest = 2
	swpNoSize               = 0x0001
	swpNoZOrder             = 0x0004
	swpNoActivate           = 0x0010
)

func clamp(v, lo, hi int32) int32 {
	if v > hi {
		v = hi
	}
	if v < lo {
		v = lo
	}
	return v
}

// placeWindow moves the window next to the mouse pointer (which, after a
// click on the tray icon, is over the notification area), keeping it
// within the monitor's work area so it doesn't cover the taskbar.
func placeWindow(w fyne.Window) {
	nw, ok := w.(driver.NativeWindow)
	if !ok {
		return
	}
	nw.RunNative(func(ctx any) {
		wc, ok := ctx.(driver.WindowsWindowContext)
		if !ok || wc.HWND == 0 {
			return
		}
		var pt point
		if r, _, _ := procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt))); r == 0 {
			return
		}
		mon, _, _ := procMonitorFromPoint.Call(uintptr(*(*uint64)(unsafe.Pointer(&pt))), monitorDefaultToNearest)
		mi := monitorInfo{Size: uint32(unsafe.Sizeof(monitorInfo{}))}
		if r, _, _ := procGetMonitorInfo.Call(mon, uintptr(unsafe.Pointer(&mi))); r == 0 {
			return
		}
		var wr rect
		if r, _, _ := procGetWindowRect.Call(wc.HWND, uintptr(unsafe.Pointer(&wr))); r == 0 {
			return
		}
		width, height := wr.Right-wr.Left, wr.Bottom-wr.Top
		x := clamp(pt.X-width/2, mi.Work.Left, mi.Work.Right-width)
		y := clamp(pt.Y-height/2, mi.Work.Top, mi.Work.Bottom-height)
		procSetWindowPos.Call(wc.HWND, 0, uintptr(x), uintptr(y), 0, 0, swpNoSize|swpNoZOrder|swpNoActivate)
		procSetForeground.Call(wc.HWND)
	})
}

func hwnd(w fyne.Window) uintptr {
	var h uintptr
	if nw, ok := w.(driver.NativeWindow); ok {
		nw.RunNative(func(ctx any) {
			if wc, ok := ctx.(driver.WindowsWindowContext); ok {
				h = wc.HWND
			}
		})
	}
	return h
}

// fixScale works around Fyne keeping the scale of the monitor a window was
// created on after it has been moved to a monitor with a different Windows
// scaling setting (Fyne only re-checks when the physical DPI changes). If
// the window's scale doesn't match its monitor, re-applying the theme makes
// Fyne recalculate it; after is then called once the window has resized.
func fixScale(w fyne.Window, after func()) {
	h := hwnd(w)
	if h == 0 || procGetDpiForWindow.Find() != nil {
		return
	}
	dpi, _, _ := procGetDpiForWindow.Call(h)
	if dpi == 0 {
		return
	}
	want := float32(math.Round(float64(dpi)/96*10) / 10)
	if w.Canvas().Scale() == want {
		return
	}
	a := fyne.CurrentApp()
	a.Settings().SetTheme(a.Settings().Theme())
	if after != nil {
		time.AfterFunc(100*time.Millisecond, func() { fyne.Do(after) })
	}
}
