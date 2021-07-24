package util

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"reflect"
	"strings"
)

var (
	Mask [4]uint32 = [4]uint32{
		0b1111_1111_1111_1111_1111_1111_0000_0000,
		0b1111_1111_1111_1111_0000_0000_1111_1111,
		0b1111_1111_0000_0000_1111_1111_1111_1111,
		0b0000_0000_1111_1111_1111_1111_1111_1111,
	}
)

func Contains(list interface{}, target interface{}) bool {
	if reflect.TypeOf(list).Kind() == reflect.Slice || reflect.TypeOf(list).Kind() == reflect.Array {
		listvalue := reflect.ValueOf(list)
		for i := 0; i < listvalue.Len(); i++ {
			if target == listvalue.Index(i).Interface() {
				return true
			}
		}
	}
	if reflect.TypeOf(target).Kind() == reflect.String && reflect.TypeOf(list).Kind() == reflect.String {
		return strings.Contains(list.(string), target.(string))
	}
	return false
}

// FormatSize convert 1024 into "1KB"
func FormatSize(s uint) string {
	const (
		_       = iota
		KB uint = 1 << (10 * iota)
		MB
		GB
	)

	switch {
	case s < KB:
		return fmt.Sprintf("%dB", s)
	case s < MB:
		return fmt.Sprintf("%dKB", s/KB)
	case s < GB:
		return fmt.Sprintf("%dMB", s/MB)
	default:
		return fmt.Sprintf("%dB", s)
	}
}

// BoolToInt converts boolean to int
func BoolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// BoolToU8 converts boolean to byte
func BoolToU8(b bool) byte {
	if b {
		return 1
	}
	return 0
}

// BoolToU32 converts boolean to uint32
func BoolToU32(b bool) uint32 {
	if b {
		return 1
	}
	return 0
}

// BoolToU32 converts boolean to uint16
func BoolToU16(b bool) uint16 {
	if b {
		return 1
	}
	return 0
}

// Bit check val's idx bit
func Bit(val interface{}, idx int) bool {
	switch val := val.(type) {
	case uint64:
		if idx < 0 || idx > 63 {
			return false
		}
		return (val & (1 << idx)) != 0

	case uint32:
		if idx < 0 || idx > 31 {
			return false
		}
		return (val & (1 << idx)) != 0

	case uint:
		if idx < 0 || idx > 31 {
			return false
		}
		return (val & (1 << idx)) != 0

	case uint16:
		if idx < 0 || idx > 15 {
			return false
		}
		return (val & (1 << idx)) != 0

	case byte:
		if idx < 0 || idx > 7 {
			return false
		}
		return (val & (1 << idx)) != 0

	case int64:
		if idx < 0 || idx > 63 {
			return false
		}
		return (val & (1 << idx)) != 0

	case int32:
		if idx < 0 || idx > 31 {
			return false
		}
		return (val & (1 << idx)) != 0

	case int:
		if idx < 0 || idx > 31 {
			return false
		}
		return (val & (1 << idx)) != 0

	case int16:
		if idx < 0 || idx > 15 {
			return false
		}
		return (val & (1 << idx)) != 0

	case int8:
		if idx < 0 || idx > 7 {
			return false
		}
		return (val & (1 << idx)) != 0
	}
	return false
}

func SetBit32(val uint32, idx int, b bool) uint32 {
	if b {
		val = val | (1 << idx)
	} else {
		val = val & ^(1 << idx)
	}
	return val
}
func SetBit16(val uint16, idx int, b bool) uint16 {
	if b {
		val = val | (1 << idx)
	} else {
		val = val & ^(1 << idx)
	}
	return val
}
func SetBit8(val byte, idx int, b bool) byte {
	if b {
		val = val | (1 << idx)
	} else {
		val = val & ^(1 << idx)
	}
	return val
}

func AddC(res uint64) bool { return res > 0xffffffff }
func SubC(res uint64) bool { return res < 0x100000000 }

func AddV(lhs, rhs, res uint32) bool {
	v := ^(lhs ^ rhs) & (lhs ^ res) & 0x8000_0000
	return v != 0
}

func SubV(lhs, rhs, res uint32) bool {
	v := (lhs ^ rhs) & (lhs ^ res) & 0x8000_0000
	return v > 0
}

func Align4(val uint32) uint32 { return val & 0b1111_1111_1111_1111_1111_1111_1111_1100 }
func Align2(val uint32) uint32 { return val & 0b1111_1111_1111_1111_1111_1111_1111_1110 }

// LE32 reads 32bit little-endian value from byteslice
func LE32(bs []byte) uint32 {
	switch len(bs) {
	case 0:
		return 0
	case 1:
		return uint32(bs[0])
	case 2:
		return uint32(bs[1])<<8 | uint32(bs[0])
	case 3:
		return uint32(bs[2])<<16 | uint32(bs[1])<<8 | uint32(bs[0])
	default:
		return binary.LittleEndian.Uint32(bs)
	}
}

// LE16 reads 16bit little-endian value from byteslice
func LE16(bs []byte) uint16 {
	switch len(bs) {
	case 0:
		return 0
	case 1:
		return uint16(bs[0])
	default:
		return binary.LittleEndian.Uint16(bs)
	}
}

func FillImage(i *image.RGBA, c color.RGBA) {
	for y := 0; y < 160; y++ {
		for x := 0; x < 240; x++ {
			i.Set(x, y, c)
		}
	}
}

func AddInt32(u uint32, i int32) uint32 {
	if i > 0 {
		u += uint32(i)
	} else {
		u -= uint32(-i)
	}
	return u
}
