package gofdt

import (
	"encoding/binary"
	"fmt"
	"os"
	"strings"
	"unsafe"
)

type ptr unsafe.Pointer
type p uintptr

const (
	FdtMagic   = 0xd00dfeed
	FdtVersion = 17
)

const (
	FdtBeginNode = 1
	FdtEndNode   = 2
	FdtProp      = 3
	FdtNop       = 4
	FdtEnd       = 9
)

type FDTHeader struct {
	magic           uint32
	totalSize       uint32
	offDtStruct     uint32
	offDtStrings    uint32
	offMemRSVmap    uint32
	version         uint32
	lastCompVersion uint32 /* <= 17 */
	bootCpuidPhys   uint32
	sizeDtStrings   uint32
	sizeDtStruct    uint32
}

type FDTReserveEntry struct {
	address uint64
	size    uint64
}

type FDT struct {
	p               ptr
	tab             []uint32
	tabLen          int
	tabSize         int
	openNodeCount   int
	stringTable     []byte
	stringTableLen  int
	stringTableSize int
}

func NewFDT(mem []byte) *FDT {
	return &FDT{
		p: ptr(&mem[0]),
	}
}

func (f *FDT) alloc(len int) {
	if len > f.tabSize {
		newSize := maxInt(len, f.tabSize*3/2)
		newTab := make([]uint32, newSize)
		copy(newTab, f.tab)
		f.tab = newTab
		f.tabSize = newSize
	}
}

func (f *FDT) put32(v uint32) {
	f.alloc(f.tabLen + 1)
	f.tab[f.tabLen] = cpuToBE32(v)
	f.tabLen++
}

func (f *FDT) putData(data []byte, l int) {
	if data == nil {
		return
	}

	if len(data) == 0 {
		f.alloc(f.tabLen + 1)
		f.tab[f.tabLen] = uint32(0)
		f.tabLen++
		return
	}

	len1 := (l + 3) / 4
	f.alloc(f.tabLen + len1)

	for i := 0; i < l; i += 4 {
		var chunk uint32
		if i+4 <= len(data) {
			chunk = binary.LittleEndian.Uint32(data[i : i+4])
		} else {
			tmp := make([]byte, 4)
			copy(tmp, data[i:])
			chunk = binary.LittleEndian.Uint32(tmp)
		}
		f.tab[f.tabLen] = chunk
		f.tabLen++
	}
}

func (f *FDT) beginNode(name string) {
	f.put32(FdtBeginNode)
	f.putData([]byte(name), len(name)+1)
	f.openNodeCount++
}

func (f *FDT) beginNodeNum(name string, n uint64) {
	f.beginNode(fmt.Sprintf("%s@%x", name, n))
}

func (f *FDT) endNode() {
	f.put32(FdtEndNode)
	f.openNodeCount--
}

func (f *FDT) endFdt() {
	f.put32(FdtEnd)
}

func (f *FDT) getStringOffset(name string) int {
	pos := strings.Index(string(f.stringTable[:f.stringTableLen]), name)
	if pos != -1 {
		return pos
	}

	nameBytes := []byte(name)
	nameLen := len(nameBytes) + 1
	newLen := f.stringTableLen + nameLen
	if newLen > f.stringTableSize {
		newSize := maxInt(newLen, f.stringTableSize*3/2)
		newStringTable := make([]byte, newSize)
		copy(newStringTable, f.stringTable)
		f.stringTable = newStringTable
		f.stringTableSize = newSize
	}

	pos = f.stringTableLen
	copy(f.stringTable[pos:], nameBytes)
	f.stringTableLen = newLen
	return pos
}

func (f *FDT) prop(name string, data []byte, dataLen int) {
	f.put32(FdtProp)
	f.put32(uint32(dataLen))
	f.put32(uint32(f.getStringOffset(name)))
	f.putData(data, dataLen)
}

func (f *FDT) propTabU32(name string, tab *uint32, tabLen int) {
	f.put32(FdtProp)
	f.put32(uint32(tabLen * int(unsafe.Sizeof(uint32(0)))))
	f.put32(uint32(f.getStringOffset(name)))
	for i := 0; i < tabLen; i++ {
		tabArr := (*[1 << 30]uint32)(ptr(tab))[:tabLen]
		f.put32(tabArr[i])
	}
}

func (f *FDT) propU32(name string, val uint32) {
	f.propTabU32(name, &val, 1)
}

func (f *FDT) propTabU64(name string, v0 uint64) {
	tab := [2]uint32{uint32(v0 >> 32), uint32(v0)}
	f.propTabU32(name, &tab[0], 2)
}

func (f *FDT) propTabU64Double(name string, v0, v1 uint64) {
	tab := [4]uint32{uint32(v0 >> 32), uint32(v0), uint32(v1 >> 32), uint32(v1)}
	f.propTabU32(name, &tab[0], 4)
}

func (f *FDT) propStr(name, str string) {
	f.prop(name, []byte(str), len(str)+1)
}

func (f *FDT) propTabStr(name string, args ...string) {
	var size int
	for _, str := range args {
		size += len(str) + 1
	}

	tab := make([]byte, size)
	offset := 0
	for _, str := range args {
		copy(tab[offset:], str)
		offset += len(str)
		tab[offset] = 0
		offset++
	}

	f.prop(name, tab, offset)
}

func (f *FDT) output() int {
	assert(f.openNodeCount == 0, fmt.Errorf("openNodeCount must be 0, current: %d", f.openNodeCount).Error())

	f.endFdt()

	dtStructSize := f.tabLen * int(unsafe.Sizeof(uint32(0)))
	dtStringsSize := f.stringTableLen

	h := (*FDTHeader)(f.p)

	// header
	h.magic = cpuToBE32(FdtMagic)
	h.version = cpuToBE32(FdtVersion)
	h.lastCompVersion = cpuToBE32(16)
	h.bootCpuidPhys = cpuToBE32(0)
	h.sizeDtStrings = cpuToBE32(uint32(dtStringsSize))
	h.sizeDtStruct = cpuToBE32(uint32(dtStructSize))

	pos := int(unsafe.Sizeof(*h))

	// align to 8
	for (pos & 7) != 0 {
		*(*uint8)(ptr(p(f.p) + p(pos))) = 0
		pos++
	}

	// memory rsv
	h.offMemRSVmap = cpuToBE32(uint32(pos))
	re := (*FDTReserveEntry)(ptr(p(f.p) + p(pos)))
	re.address = 0
	re.size = 0
	pos += int(unsafe.Sizeof(*re))
	h.offDtStruct = cpuToBE32(uint32(pos))

	// structure block
	if f.tabLen > 0 {
		CopyMemory(ptr(p(f.p)+p(pos)), ptr(&f.tab[0]), p(dtStructSize))
		pos += dtStructSize
	}
	h.offDtStrings = cpuToBE32(uint32(pos))

	// string block
	if f.stringTableLen > 0 {
		CopyMemory(ptr(p(f.p)+p(pos)), ptr(&f.stringTable[0]), p(dtStringsSize))
		pos += dtStringsSize
	}

	h.totalSize = cpuToBE32(uint32(pos))

	return pos
}

func (f *FDT) dumpDTB(file string) {
	fp, err := os.Create(file)
	if err != nil {
		panic(err)
	}
	defer fp.Close()

	_, err = fp.Write(pointerToBytesArrayWithLen(f.p, f.output()))
	if err != nil {
		panic(err)
	}
}
