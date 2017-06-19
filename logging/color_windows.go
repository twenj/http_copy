package logging

import (
	"io"
	"fmt"
	"syscall"
)

type ColorType uint16

const (
	ColorRed     ColorType = 0x0004 | 0x0008
	ColorGreen   ColorType = 0x0002 | 0x0008
	ColorYellow  ColorType = 0x0004 | 0x0002 | 0x0008
	ColorBlue    ColorType = 0x0001 | 0x0008
	ColorMagents ColorType = 0x0001 | 0x0004 | 0x0008
	ColorCyan    ColorType = 0x0002 | 0x0001 | 0x0008
	ColorWhite   ColorType = 0x0004 | 0x0001 | 0x0002 | 0x0008
	ColorGray    ColorType = 0x0004 | 0x0002 | 0x0001
)

var (
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	proSetConsoleTextAttribute = kernel32.NewProc("SetConsoleTextAttribute")
)

func setConsoleTextAttribute(wAttributes uint16) bool {
	ret, _, _ := proSetConsoleTextAttribute.Call(
		uintptr(syscall.Stdout),
		uintptr(wAttributes),
	)
	return ret != 0
}

func FprintWithColor(w io.Writer, str string, code ColorType) (int, error) {
	if setConsoleTextAttribute(uint16(code)) {
		defer setConsoleTextAttribute(uint16(ColorGray))
	}
	return fmt.Fprint(w, str)
}