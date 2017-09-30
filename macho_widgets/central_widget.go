package macho_widgets

import (
	"debug/macho"
	"sort"

	"github.com/therecipe/qt/widgets"
)

func NewCentralWidget(f *macho.File) widgets.QWidget_ITF {
	// define common data structure
	ssyms := makeSortedSymbols(f)
	symAddrInfo := makeSymAddrInfo(f, ssyms)
	lookup := func(addr uint64) (string, uint64) {
		j := sort.Search(len(ssyms), func(i int) bool {
			return addr < ssyms[i].Value
		})
		if j > 0 {
			sym := ssyms[j-1]
			info := symAddrInfo[sym.Value]
			if sym.Value != 0 && sym.Value <= addr && addr <= sym.Value+info.Size {
				return sym.Name, sym.Value
			}
		}
		return "", 0
	}

	tab := widgets.NewQTabWidget(nil)
	tab.AddTab(NewStructWidget(f), "Structure")
	tab.AddTab(NewSymtabWidget(f, ssyms, symAddrInfo, lookup), "Symbols")
	if f.Type == macho.TypeObj {
		tab.AddTab(NewReltabWidget(f, lookup), "Relocations")
	}
	return tab
}

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
