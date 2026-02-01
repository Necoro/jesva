//go:build eric

package eric

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"unsafe"
)

// #include <stdlib.h>
// #include "ericapi.h"
// #include "eric_fehlercodes.h"
// #cgo LDFLAGS: -Llib/ -lericapi -Wl,-rpath=${SRCDIR}/lib
// #cgo CPPFLAGS: -Iinclude
import "C"

type Art int

const (
	UStVA Art = iota
	UStE
)

func (a Art) name() string {
	switch a {
	case UStVA:
		return "UStVA"
	case UStE:
		return "UStE"
	}
	return ""
}

type Datenart struct {
	a      Art
	year   int
	string *C.char
}

func GetDatenart(a Art, year int) *Datenart {
	s := fmt.Sprintf("%s_%04d", a.name(), year)
	cs := C.CString(s)
	d := &Datenart{a, year, cs}

	runtime.AddCleanup(d, func(ptr unsafe.Pointer) {
		C.free(ptr)
	}, unsafe.Pointer(d.string))
	return d
}

func (d *Datenart) String() string {
	return C.GoString(d.string)
}

type ericBuffer struct {
	handle C.EricRueckgabepufferHandle
}

func (b *ericBuffer) init() {
	b.handle = C.EricRueckgabepufferErzeugen()
	if b.handle == nil {
		panic("Failed to create Eric return buffer")
	}
}

func (b *ericBuffer) close() {
	if b.handle != nil {
		C.EricRueckgabepufferFreigeben(b.handle)
		b.handle = nil
	}
}

func (b *ericBuffer) content() string {
	return C.GoString(C.EricRueckgabepufferInhalt(b.handle))
}

type Eric struct {
	buf *ericBuffer
}

func (e *Eric) Init() {
	// ignore errors: if removing fails, so be it
	_ = os.Remove("eric.log")

	cwd, err := os.Getwd()
	if err != nil {
		log.Panicf("getcwd: %v", err)
	}

	cwdC := C.CString(cwd)
	defer C.free(unsafe.Pointer(cwdC))
	if stat := C.EricInitialisiere(nil, cwdC); stat != C.ERIC_OK {
		panic("Init Eric")
	}

	e.buf = new(ericBuffer)
	e.buf.init()
}

func (e *Eric) SystemCheck() {
	C.EricSystemCheck()
}

func (e *Eric) CheckSteuerNr(steuerNr string) error {
	nr := C.CString(steuerNr)
	defer C.free(unsafe.Pointer(nr))

	status := int(C.EricPruefeSteuernummer(nr))
	if status == C.ERIC_GLOBAL_STEUERNUMMER_UNGUELTIG {
		return fmt.Errorf("Steuer-Nr. '%s' ungültig", steuerNr)
	} else if status != C.ERIC_OK {
		log.Panicf("Unexpected status %d", status)
	}

	return nil
}

func (e *Eric) CheckWID(wid string) error {
	widC := C.CString(wid)
	defer C.free(unsafe.Pointer(widC))

	status := int(C.EricPruefeWIdNr(widC))
	if status == C.ERIC_GLOBAL_IDNUMMER_UNGUELTIG {
		return fmt.Errorf("W-ID-Nr '%s' ungültig", wid)
	} else if status != C.ERIC_OK {
		log.Panicf("Unexpected status %d", status)
	}

	return nil
}

func (e *Eric) Close() {
	e.buf.close()
	C.EricBeende()
}
