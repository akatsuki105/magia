package util

import (
	"encoding/binary"
	"fmt"
	"reflect"
	"strings"
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

// ToBool converts value to boolean
func ToBool(val interface{}) bool {
	switch val := val.(type) {
	case bool:
		return val
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return !(val == 0)
	case string:
		return len(val) > 0
	}
	return false
}

// BoolToInt converts boolean to int
func BoolToInt(b bool) int {
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

	case int:
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
	}
	return false
}

func AddV(lhs, rhs, res uint32) bool {
	v := ^(lhs ^ rhs) & (lhs ^ res) & 0x80000000
	return ToBool(v)
}

func SubV(lhs, rhs, res uint32) bool {
	v := (lhs ^ rhs) & (lhs ^ res) & 0x80000000
	return ToBool(v)
}

func Align4(val uint32) uint32 {
	return val & 0b1111_1111_1111_1111_1111_1111_1111_1100
}
func Align2(val uint32) uint32 {
	return val & 0b1111_1111_1111_1111_1111_1111_1111_1110
}

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
