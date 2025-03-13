package xutils

import (
	"unsafe"
)

// This conversion *does not* copy data. Note that casting via "(string)([]byte)" *does* copy data.
// Also note that you *should not* change the byte slice after conversion, because Go strings
// are treated as immutable. This would cause a segmentation violation panic.
func BytesToString(bytes []byte) string {
	return unsafe.String(unsafe.SliceData(bytes), len(bytes))
}

// This conversion *does not* copy data. Note that casting via "([]byte)(string)" *does* copy data.
// Also note that you *should not* change the byte slice after conversion, because Go strings
// are treated as immutable. This would cause a segmentation violation panic.
func StringToBytes(string_ string) (bytes []byte) {
	return unsafe.Slice(unsafe.StringData(string_), len(string_))
}
