package macho_widgets

import (
	"bytes"
	"debug/macho"
	"fmt"
	"math"
	"runtime"
	"sort"
	"strings"
	"unsafe"

	"golang.org/x/arch/arm/armasm"
	"golang.org/x/arch/arm64/arm64asm"
	"golang.org/x/arch/ppc64/ppc64asm"
	"golang.org/x/arch/x86/x86asm"
)

// #include <stdio.h>
//
// int sprintf_3(char * restrict s, const char * restrict format, void * val1) {
//   return sprintf(s, format, *(long double *)val1);
// }
import "C"

type File struct {
	*macho.File
	Syms      []macho.Symbol
	SymInfos  map[uint64]*SymInfo
	SymLookup SymLookup
}

type SymInfo struct {
	Size          uint64
	Relocs        []macho.Reloc
	RelocSections []*macho.Section
	SymbolIndices []int
}

type SymLookup func(addr uint64) (string, uint64)

func NewFile(f *macho.File) *File {
	var syms []macho.Symbol
	if f.Symtab != nil {
		syms = f.Symtab.Syms
	}
	ssyms := makeSortedSymbols(f)
	symInfos := makeSymInfos(f, ssyms)
	symLookup := func(addr uint64) (string, uint64) {
		j := sort.Search(len(ssyms), func(i int) bool {
			return addr < ssyms[i].Value
		})
		if j > 0 {
			sym := ssyms[j-1]
			info := symInfos[sym.Value]
			if sym.Value != 0 && sym.Value <= addr && addr < sym.Value+info.Size {
				ss := make([]string, len(info.SymbolIndices))
				for i, si := range info.SymbolIndices {
					sym := &syms[si]
					if sym.Value == addr {
						ss[i] = sym.Name
					} else {
						ss[i] = fmt.Sprintf("%s%+x", sym.Name, addr-sym.Value)
					}
				}
				return strings.Join(ss, "|"), sym.Value
			}
		}
		return "", 0
	}
	return &File{
		File:      f,
		Syms:      syms,
		SymInfos:  symInfos,
		SymLookup: symLookup,
	}
}

type SortedSymbols []struct {
	*macho.Symbol

	Index int
}

func (v SortedSymbols) Len() int {
	return len(v)
}

func (v SortedSymbols) Less(i, j int) bool {
	return v[i].Value < v[j].Value
}

func (v SortedSymbols) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}

func makeSortedSymbols(f *macho.File) SortedSymbols {
	var syms []macho.Symbol
	if f.Symtab != nil {
		syms = f.Symtab.Syms
	}

	ssyms := make(SortedSymbols, 0, len(syms))

	for i := range syms {
		sym := &syms[i]
		if sym.Type&N_STAB == 0 && sym.Type&N_TYPE == N_SECT {
			ssyms = append(ssyms, struct {
				*macho.Symbol
				Index int
			}{
				Symbol: sym,
				Index:  i,
			})
		}
	}

	sort.Sort(ssyms)

	return ssyms
}

func makeSymInfos(f *macho.File, ssyms SortedSymbols) map[uint64]*SymInfo {
	symInfos := make(map[uint64]*SymInfo)

	if len(ssyms) != 0 {
		for i := 0; i < len(ssyms); i++ {
			sym := ssyms[i]
			info := new(SymInfo)
			info.SymbolIndices = append(info.SymbolIndices, sym.Index)
			if i == len(ssyms)-1 {
				if 0 < int(sym.Sect) && int(sym.Sect) <= len(f.Sections) {
					sect := f.Sections[sym.Sect-1]
					info.Size = sect.Addr + sect.Size - sym.Value
				}
			} else {
				for j := i + 1; j < len(ssyms); j++ {
					nsym := ssyms[j]
					if sym.Value != nsym.Value {
						if sym.Sect == nsym.Sect {
							info.Size = nsym.Value - sym.Value
						} else {
							if 0 < int(sym.Sect) && int(sym.Sect) <= len(f.Sections) {
								sect := f.Sections[sym.Sect-1]
								info.Size = sect.Addr + sect.Size - sym.Value
							}
						}
						i = j - 1
						break
					}
					info.SymbolIndices = append(info.SymbolIndices, nsym.Index)
				}
			}
			symInfos[sym.Value] = info
		}

		for _, sect := range f.Sections {
			for _, r := range sect.Relocs {
				k := sort.Search(len(ssyms), func(j int) bool {
					sym := ssyms[j]
					return sym.Value > sect.Addr+uint64(r.Addr)
				})
				if k == 0 {
					continue
				}
				sym := ssyms[k-1]
				info := symInfos[sym.Value]
				if sym.Value <= sect.Addr+uint64(r.Addr) && sect.Addr+uint64(r.Addr)+(1<<r.Len) <= sym.Value+info.Size {
					info.Relocs = append(info.Relocs, r)
					info.RelocSections = append(info.RelocSections, sect)
				}
			}
		}
	}

	return symInfos
}

func (f *File) symAddrString(addr uint64, force bool) string {
	if s, base := f.SymLookup(addr); s != "" {
		info := f.SymInfos[base]
		ss := make([]string, len(info.SymbolIndices))
		for i, si := range info.SymbolIndices {
			sym := &f.Syms[si]
			if base == addr {
				ss[i] = sym.Name
			} else {
				ss[i] = fmt.Sprintf("%s%+x", sym.Name, addr-base)
			}
		}
		return strings.Join(ss, "|")
	}
	if force {
		return fmt.Sprintf("%#x", addr)
	}
	return ""
}

func (f *File) symIndexString(i uint32) string {
	if sym := f.symIndex(i); sym != nil {
		return sym.Name
	}
	return ""
}

func (f *File) symIndex(i uint32) *macho.Symbol {
	if len(f.Syms) < math.MaxUint32 && 0 <= i && i < uint32(len(f.Syms)) {
		return &f.Syms[i]
	}
	return nil
}

func (f *File) sectNumString(num uint32) string {
	if len(f.Sections) < math.MaxUint32 && 0 <= num-1 && num-1 < uint32(len(f.Sections)) {
		sect := f.Sections[num-1]
		return fmt.Sprintf("%s,%s", sect.Seg, sect.Name)
	}
	return ""
}

func (f *File) disasmFunc() func(code []byte, pc uint64) (string, int) {
	switch f.Cpu {
	case macho.Cpu386:
		return func(code []byte, pc uint64) (string, int) {
			inst, err := x86asm.Decode(code, 32)
			if err != nil {
				return "?", 1
			}
			syntax := x86asm.GNUSyntax(inst, pc, x86asm.SymLookup(f.SymLookup))
			return syntax, inst.Len
		}
	case macho.CpuAmd64:
		return func(code []byte, pc uint64) (string, int) {
			inst, err := x86asm.Decode(code, 64)
			if err != nil {
				return "?", 1
			}
			syntax := x86asm.GNUSyntax(inst, pc, x86asm.SymLookup(f.SymLookup))
			return syntax, inst.Len
		}
	case macho.CpuArm:
		return func(code []byte, pc uint64) (string, int) {
			inst, err := armasm.Decode(code, armasm.ModeARM)
			if err != nil {
				return "?", 1
			}
			syntax := armasm.GNUSyntax(inst)
			return syntax, inst.Len
		}
	case macho.CpuArm | 0x01000000:
		return func(code []byte, pc uint64) (string, int) {
			inst, err := arm64asm.Decode(code)
			if err != nil {
				return "?", 4
			}
			syntax := arm64asm.GNUSyntax(inst)
			return syntax, 4
		}
	case macho.CpuPpc64:
		return func(code []byte, pc uint64) (string, int) {
			inst, err := ppc64asm.Decode(code, f.ByteOrder)
			if err != nil {
				return "?", 1
			}
			syntax := ppc64asm.GNUSyntax(inst)
			return syntax, inst.Len
		}
	}

	return nil
}

func (f *File) toSymChar(typ uint8, sect uint8, val uint64) byte {
	if typ&N_STAB != 0 {
		return '-'
	}
	switch typ & N_TYPE {
	case N_UNDF:
		if val == 0 {
			if typ&N_EXT != 0 {
				return 'U'
			}
			return 'u'
		}
		if typ&N_EXT != 0 {
			return 'C'
		}
		return 'c'
	case N_ABS:
		if typ&N_EXT != 0 {
			return 'A'
		}
		return 'a'
	case N_SECT:
		if sect == 0 {
			if typ&N_EXT != 0 {
				return 'B'
			}
			return 'b'
		}
		if 0 <= int(sect-1) && int(sect-1) < len(f.Sections) {
			s := f.Sections[sect-1]
			switch {
			case s.Seg == "__TEXT" && s.Name == "__text":
				if typ&N_EXT != 0 {
					return 'T'
				}
				return 't'
			case s.Seg == "__DATA" && s.Name == "__data":
				if typ&N_EXT != 0 {
					return 'D'
				}
				return 'd'
			}
		}
		if typ&N_EXT != 0 {
			return 'S'
		}
		return 's'
	case N_PBUD:
		if typ&N_EXT != 0 {
			return 'U'
		}
		return 'u'
	case N_INDR:
		if typ&N_EXT != 0 {
			return 'I'
		}
		return 'i'
	default:
		return '?'
	}
}

func (f *File) toASCII(data []byte) string {
	ret := make([]byte, len(data))
	for i, c := range data {
		if 32 <= c && c < 127 {
			ret[i] = c
		} else {
			ret[i] = '.'
		}
	}
	return string(ret)
}

func (f *File) toFloat32(data []byte) string {
	if len(data) != 4 {
		return ""
	}
	return fmt.Sprintf("%g", math.Float32frombits(f.ByteOrder.Uint32(data)))
}

func (f *File) toFloat64(data []byte) string {
	if len(data) != 8 {
		return ""
	}
	return fmt.Sprintf("%g", math.Float64frombits(f.ByteOrder.Uint64(data)))
}

func (f *File) toFloat128(data []byte) string {
	if len(data) != 16 {
		return ""
	}

	// TODO encoding of `long double` is platform dependent, so this is not precisely correct.

	switch f.Cpu {
	case macho.Cpu386:
		if runtime.GOARCH != "386" {
			return ""
		}
	case macho.CpuAmd64:
		if runtime.GOARCH != "amd64" {
			return ""
		}
	case macho.CpuArm:
		if runtime.GOARCH != "arm" {
			return ""
		}
	case macho.CpuArm | 0x01000000:
		if runtime.GOARCH != "arm64" {
			return ""
		}
	case macho.CpuPpc:
		if runtime.GOARCH != "ppc" {
			return ""
		}
	case macho.CpuPpc64:
		if runtime.GOARCH != "ppc64" {
			return ""
		}
	}

	format := []byte("%Lg")
	bs := make([]byte, 64)
	for {
		i := int(C.sprintf_3((*C.char)(unsafe.Pointer(&bs[0])), (*C.char)(unsafe.Pointer(&format[0])), unsafe.Pointer(&data[0])))
		if i == -1 {
			return ""
		}
		if i < len(bs) {
			break
		}
		bs = make([]byte, len(bs)*2)
	}
	i := bytes.IndexByte(bs, 0)
	if i == -1 {
		return ""
	}
	return string(bs[:i])
}

func (f *File) toPointer32(data []byte) string {
	if len(data) != 4 {
		return ""
	}
	addr := uint64(f.ByteOrder.Uint32(data))
	suffix := ""
	if s := f.symAddrString(addr, false); s != "" {
		suffix = fmt.Sprintf(` (%s)`, s)
	}
	return fmt.Sprintf("%#016x%s", addr, suffix)
}
