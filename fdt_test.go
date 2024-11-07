package gofdt

import (
	"fmt"
	"testing"
)

type MemoryUnit uint64

const (
	_             = iota
	KB MemoryUnit = 1 << (10 * iota)
	MB
	GB
	TB
)

const (
	RamBaseAddr    = 0x80000000
	ClintBaseAddr  = 0x02000000
	ClintSize      = 0x000c0000
	VIRTIOBaseAddr = 0x40010000
	VIRTIOSize     = 0x1000
	VirtualIOIrq   = 1
	PLICBaseAddr   = 0x40100000
	PLICSize       = 0x00400000
	RtcFreq        = 10000000
)

func TestFDT_Workflow(t *testing.T) {
	mem := make([]byte, RamBaseAddr)

	fdt := NewFDT(mem)

	curPHandle := uint32(1)

	fdt.beginNode("")
	fdt.propU32("#address-cells", 2)
	fdt.propU32("#size-cells", 2)
	fdt.propStr("compatible", "ucbbar,riscvemu-bar_dev")
	fdt.propStr("model", "ucbbar,riscvemu-bare")

	/* CPU list */
	fdt.beginNode("cpus")
	fdt.propU32("#address-cells", 1)
	fdt.propU32("#size-cells", 0)
	fdt.propU32("timebase-frequency", RtcFreq)

	/* cpu */
	fdt.beginNodeNum("cpu", 0)
	fdt.propStr("device_type", "cpu")
	fdt.propU32("reg", 0)
	fdt.propStr("status", "okay")
	fdt.propStr("compatible", "riscv")

	maxXLen := 128 * MB
	misa := 19
	isaString := fmt.Sprintf("rv%d", maxXLen)
	for i := 0; i < 26; i++ {
		if misa&(1<<i) != 0 {
			isaString += string('a' + byte(i))
		}
	}
	fdt.propStr("riscv,isa", isaString)
	fdt.propStr("mmu-type", func() string {
		if maxXLen <= 32 {
			return "riscv,sv32"
		} else {
			return "riscv,sv48"
		}
	}())
	fdt.propU32("clock-frequency", 2000000000)
	fdt.endNode() // cpu

	fdt.beginNode("interrupt-controller")
	fdt.propU32("#interrupt-cells", 1)
	fdt.prop("interrupt-controller", nil, 0)
	fdt.propStr("compatible", "riscv,cpu-intc")
	intCPHandle := curPHandle
	curPHandle++
	fdt.propU32("phandle", intCPHandle)
	fdt.endNode() // interrupt-controller

	fdt.endNode() // cpus

	fdt.beginNodeNum("memory", RamBaseAddr)
	fdt.propStr("device_type", "memory")

	kernelStart := uint64(12)
	kernelSize := uint64(1024 * 1024 * 16) // 16MB
	tab := [4]uint32{
		uint32(kernelStart >> 32),
		uint32(kernelStart),
		uint32(kernelStart + kernelSize>>32),
		uint32(kernelStart + kernelSize),
	}
	fdt.propTabU32("reg", &tab[0], 4)
	fdt.endNode() // memory

	fdt.beginNode("htif")
	fdt.propStr("compatible", "ucb,htif0")
	fdt.endNode() // htif

	fdt.beginNode("soc")
	fdt.propU32("#address-cells", 2)
	fdt.propU32("#size-cells", 2)
	fdt.propTabStr("compatible", "ucbbar,riscvemu-bar-soc", "simple-bus")
	//fdt.prop("ranges", nil, 0)

	fdt.beginNodeNum("clint", ClintBaseAddr)
	fdt.propStr("compatible", "riscv,clint0")

	tab[0] = intCPHandle
	tab[1] = 3 // M IPI irq
	tab[2] = intCPHandle
	tab[3] = 7 // M timer irq
	fdt.propTabU32("interrupts-extended", &tab[0], 4)

	fdt.propTabU64Double("reg", ClintBaseAddr, ClintSize)

	fdt.endNode() // clint

	fdt.beginNodeNum("plic", PLICBaseAddr)
	fdt.propU32("#interrupt-cells", 1)
	fdt.prop("interrupt-controller", nil, 0)
	fdt.propStr("compatible", "riscv,plic0")
	fdt.propU32("riscv,ndev", 31)
	fdt.propTabU64Double("reg", PLICBaseAddr, PLICSize)
	tab[0] = intCPHandle
	tab[1] = 9 // S ext irq
	tab[2] = intCPHandle
	tab[3] = 11 // M ext irq
	fdt.propTabU32("interrupts-extended", &tab[0], 4)
	plicPHandle := curPHandle
	curPHandle++
	fdt.propU32("phandle", plicPHandle)
	fdt.endNode() // plic

	VIRTIoCount := 3
	for i := 0; i < VIRTIoCount; i++ {
		fdt.beginNodeNum("virtio", uint64(VIRTIOBaseAddr+i*VIRTIOSize))
		fdt.propStr("compatible", "virtio,mmio")
		fdt.propTabU64Double("reg", uint64(VIRTIOBaseAddr+i*VIRTIOSize), VIRTIOSize)
		tab[0] = plicPHandle
		tab[1] = VirtualIOIrq + uint32(i)
		fdt.propTabU32("interrupts-extended", &tab[0], 2)
		fdt.endNode() // virtio
	}

	fdt.endNode() // soc

	fdt.beginNode("chosen")
	cmdLine := "loglevel=3 console=hvc0 root=/dev/vda rw"
	fdt.propStr("bootargs", cmdLine)
	if kernelSize > 0 {
		fdt.propTabU64("riscv,kernel-start", kernelStart)
		fdt.propTabU64("riscv,kernel-end", kernelStart+kernelSize)
	}

	initrdSize := uint64(30)
	initrdStart := uint64(10)
	if initrdSize > 0 {
		fdt.propTabU64("linux,initrd-start", initrdStart)
		fdt.propTabU64("linux,initrd-end", initrdStart+initrdSize)
	}
	fdt.endNode()
	fdt.endNode()

	fdt.dumpDTB("./output.dtb")
}
