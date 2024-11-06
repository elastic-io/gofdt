package gofdt

import "encoding/binary"

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func CopyMemory(dst, src ptr, size p) {
	d := (*[1 << 30]byte)(dst)[:size]
	s := (*[1 << 30]byte)(src)[:size]
	copy(d, s)
}

func cpuToBE32(v uint32) uint32 {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, v)
	return binary.BigEndian.Uint32(b)
}

func pointerToBytesArrayWithLen(p ptr, len int) []byte {
	return (*[1 << 30]byte)(p)[:len]
}

func assert(condition bool, message string) {
	if !condition {
		panic(message)
	}
}
