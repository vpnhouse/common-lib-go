package stats

import (
	"encoding/binary"
)

func RxTx(rx, tx int64, d []byte) []byte {
	if len(d) == 0 {
		d = make([]byte, 16)
	}
	binary.LittleEndian.PutUint64(d, uint64(rx))
	binary.LittleEndian.PutUint64(d[8:], uint64(tx))
	return d
}

func ParseRxTx(d []byte) (int64, int64) {
	if len(d) != 16 {
		return 0, 0
	}
	rx := int64(binary.LittleEndian.Uint64(d[:8]))
	tx := int64(binary.LittleEndian.Uint64(d[8:]))
	return rx, tx
}

func IncRxTx(drx, dtx int64, d []byte) []byte {
	if len(d) != 16 {
		return d
	}
	if drx == 0 && dtx == 0 {
		return d
	}
	rx, tx := ParseRxTx(d)
	return RxTx(rx+drx, tx+dtx, d)
}
