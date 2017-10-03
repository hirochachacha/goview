package macho_widgets

import (
	"debug/macho"
	"fmt"
	"math"
	"sort"
)

type symLookup func(addr uint64) (string, uint64)

func makeSortedSymbols(f *macho.File) []*macho.Symbol {
	var syms []macho.Symbol
	if f.Symtab != nil {
		syms = f.Symtab.Syms
	}

	ssyms := make([]*macho.Symbol, 0, len(syms))

	for i := range syms {
		sym := &syms[i]
		if sym.Type&N_STAB == 0 && sym.Type&N_TYPE == N_SECT {
			ssyms = append(ssyms, sym)
		}
	}
	sort.Sort(byAddr(ssyms))

	return ssyms
}

type symInfo struct {
	Size          uint64
	Relocs        []macho.Reloc
	RelocSections []*macho.Section
	Symbols       []*macho.Symbol
}

func makeSymAddrInfo(f *macho.File, ssyms []*macho.Symbol) map[uint64]*symInfo {
	symAddrInfo := make(map[uint64]*symInfo)

	if len(ssyms) != 0 {
		for i := 0; i < len(ssyms); i++ {
			sym := ssyms[i]
			info := new(symInfo)
			info.Symbols = append(info.Symbols, sym)
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
					info.Symbols = append(info.Symbols, nsym)
				}
			}
			symAddrInfo[sym.Value] = info
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
				info := symAddrInfo[sym.Value]
				if sym.Value <= sect.Addr+uint64(r.Addr) && sect.Addr+uint64(r.Addr)+(1<<r.Len) <= sym.Value+info.Size {
					info.Relocs = append(info.Relocs, r)
					info.RelocSections = append(info.RelocSections, sect)
				}
			}
		}
	}

	return symAddrInfo
}

func symAddrString(addr uint64, lookup symLookup, force bool) string {
	if s, base := lookup(addr); s != "" {
		if base == addr {
			return s
		}
		return fmt.Sprintf("%s%+x", s, addr-base)
	}
	if force {
		return fmt.Sprintf("%#x", addr)
	}
	return ""
}

func symIndexString(f *macho.File, i uint32) string {
	if sym := symIndex(f, i); sym != nil {
		return sym.Name
	}
	return ""
}

func symIndex(f *macho.File, i uint32) *macho.Symbol {
	var syms []macho.Symbol
	if f.Symtab != nil {
		syms = f.Symtab.Syms
	}
	if len(syms) < math.MaxUint32 && 0 <= i && i < uint32(len(syms)) {
		return &syms[i]
	}
	return nil
}

func sectNumString(f *macho.File, num uint32) string {
	if len(f.Sections) < math.MaxUint32 && 0 <= num-1 && num-1 < uint32(len(f.Sections)) {
		sect := f.Sections[num-1]
		return fmt.Sprintf("%s,%s", sect.Seg, sect.Name)
	}
	return ""
}
