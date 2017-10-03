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
