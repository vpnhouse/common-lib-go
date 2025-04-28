package xstats

import (
	"encoding/binary"
)

func ParseUint64(d []byte) uint64 {
	if len(d) < 8 {
		return 0
	}
	return binary.LittleEndian.Uint64(d[:8])
}

func SetUint64(v uint64, d []byte) {
	if len(d) < 8 {
		return
	}
	binary.LittleEndian.PutUint64(d, v)
}

func ParseUint16(d []byte) uint16 {
	if len(d) < 2 {
		return 0
	}
	return binary.LittleEndian.Uint16(d[:2])
}

func SetUint16(v uint16, d []byte) {
	if len(d) < 2 {
		return
	}
	binary.LittleEndian.PutUint16(d, v)
}

func RxTx(rx, tx uint64, d []byte) []byte {
	if len(d) == 0 {
		d = make([]byte, 16)
	}
	binary.LittleEndian.PutUint64(d, rx)
	binary.LittleEndian.PutUint64(d[8:], tx)
	return d
}

func Rx(rx uint64, d []byte) []byte {
	if len(d) == 0 {
		d = make([]byte, 16)
	}
	binary.LittleEndian.PutUint64(d, rx)
	return d
}

func Tx(tx uint64, d []byte) []byte {
	if len(d) == 0 {
		d = make([]byte, 16)
	}
	binary.LittleEndian.PutUint64(d[8:], tx)
	return d
}

func ParseRxTx(d []byte) (uint64, uint64) {
	if len(d) < 16 {
		return 0, 0
	}
	rx := binary.LittleEndian.Uint64(d[:8])
	tx := binary.LittleEndian.Uint64(d[8:16])
	return rx, tx
}

func ParseRx(d []byte) uint64 {
	if len(d) < 8 {
		return 0
	}
	return binary.LittleEndian.Uint64(d[:8])
}

func ParseTx(d []byte) uint64 {
	if len(d) < 16 {
		return 0
	}
	return binary.LittleEndian.Uint64(d[8:16])
}

func AddRxTx(drx, dtx uint64, d []byte) []byte {
	if drx == 0 && dtx == 0 {
		return d
	}
	rx, tx := ParseRxTx(d)
	return RxTx(rx+drx, tx+dtx, d)
}

func AddRx(drx uint64, d []byte) []byte {
	if drx == 0 {
		return d
	}
	rx := ParseRx(d)
	return Rx(rx+drx, d)
}

func AddTx(dtx uint64, d []byte) []byte {
	if dtx == 0 {
		return d
	}
	tx := ParseTx(d)
	return Tx(tx+dtx, d)
}
