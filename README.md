# gofdt
[![CI](https://github.com/elastic-io/gofdt/actions/workflows/go.yml/badge.svg?branch=main)](https://github.com/elastic-io/gofdt/actions/workflows/go.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/elastic-io/gofdt.svg)](https://pkg.go.dev/github.com/elastic-io/gofdt)
![GitHub](https://img.shields.io/github/license/elastic-io/gofdt)

The device tree describes the various hardware components on the system, including the CPU, memory, peripherals, etc., as well as their configuration and interconnection methods. It provides a hardware abstraction that allows the same operating system kernel to run on different hardware without having to write specific code for each hardware platform.

> The Devicetree Blob (DTB) format is a flat binary encoding of devicetree data. It used to exchange devicetree data between software programs. For example, when booting an operating system, firmware will pass a DTB to the OS kernel(https://devicetree-specification.readthedocs.io/en/stable/index.html).

gofdt is a set of fdt libraries based on spce. It currently supports API-based construction of DTB files. The output DTB files can be loaded and used by the OS kernel.

# features
This FDT implementation is a full feature implementation of FDT spec. Features includes:
- Memory Reservation Block
- Structure Block
- Strings Block
- Alignment

# usage
A Simple Example:
```go
mem := make([]byte, 0x80000000)

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
fdt.propU32("timebase-frequency", 10000000)

/* cpu */
fdt.beginNodeNum("cpu", 0)
fdt.propStr("device_type", "cpu")
fdt.propU32("reg", 0)
fdt.propStr("status", "okay")
fdt.propStr("compatible", "riscv")

maxXLen := 128 * 1048576
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
fdt.endNode()

fdt.dumpDTB("./output.dtb")
```
