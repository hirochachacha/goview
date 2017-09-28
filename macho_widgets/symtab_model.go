package macho_widgets

import (
	"debug/macho"
	"fmt"
	"sort"
	"strings"

	"github.com/therecipe/qt/core"
)

type SymtabModel struct {
	Symtab core.QAbstractItemModel_ITF
	reltab func(index *core.QModelIndex) core.QAbstractItemModel_ITF
	disasm func(index *core.QModelIndex) string
}

func NewSymtabModel(f *macho.File) *SymtabModel {
	m := new(SymtabModel)

	symtab := core.NewQSortFilterProxyModel(nil)
	symtab.SetSourceModel(m.newSymtabModel(f))

	reltab := m.newReltabModel(f, m.makeSymAddrInfo(f))

	return &SymtabModel{
		Symtab: symtab,
		reltab: reltab,
	}
}

func (m *SymtabModel) SetFilter(s string) {
	m.Symtab.(*core.QSortFilterProxyModel).SetFilterRegExp2(s)
}

func (m *SymtabModel) Reltab(index *core.QModelIndex) core.QAbstractItemModel_ITF {
	return m.reltab(m.Symtab.(*core.QSortFilterProxyModel).MapToSource(index))
}

func (m *SymtabModel) Disasm(index *core.QModelIndex) string {
	return m.disasm(m.Symtab.(*core.QSortFilterProxyModel).MapToSource(index))
}

func (m *SymtabModel) newDisasm(f *macho.File) func(*core.QModelIndex) string {
	return nil
	// if !index.IsValid() {
	// return nil
	// }
	// row := index.Row()
	// if 0 <= row && row < len(syms) {
	// sym := &syms[row]

	// if sym.Type&N_STAB == 0 && sym.Type&N_TYPE == N_SECT {
	// // if 0 < int(sym.Sect) && int(sym.Sect) <= len(f.Sections) {
	// // sect := f.Sections[sym.Sect-1]
	// // } else {
	// // // TODO warning
	// // }

	// var reltab core.QAbstractItemModel_ITF

	// if symInfo := symAddrInfo[sym.Value]; symInfo != nil {
	// reltab = newReltabModel(f, symInfo.Relocs, symInfo.Sections)
	// if reltab != nil {
	// proxy := core.NewQSortFilterProxyModel(nil)
	// proxy.SetSourceModel(reltab)
	// reltab = proxy
	// }
	// }

	// reltabCache[row] = reltab

	// return reltab
	// }
	// }
	// return nil
}

func (m *SymtabModel) newSymtabModel(f *macho.File) core.QAbstractItemModel_ITF {
	var syms []macho.Symbol
	if f.Symtab != nil {
		syms = f.Symtab.Syms
	}

	header := []string{"Name", "Type", "Section", "Description", "Value"}

	symtab := core.NewQAbstractTableModel(nil)
	symtab.ConnectRowCount(func(parent *core.QModelIndex) int {
		return len(syms)
	})
	symtab.ConnectColumnCount(func(parent *core.QModelIndex) int {
		return len(header)
	})
	symtab.ConnectHeaderData(func(section int, orientation core.Qt__Orientation, role int) *core.QVariant {
		if role == int(core.Qt__DisplayRole) {
			var val string
			switch orientation {
			case core.Qt__Horizontal:
				val = header[section]
			case core.Qt__Vertical:
				val = fmt.Sprint(section)
			}
			return core.NewQVariant14(val)
		}
		return core.NewQVariant()
	})
	symtab.ConnectData(func(index *core.QModelIndex, role int) *core.QVariant {
		if role != int(core.Qt__DisplayRole) {
			return core.NewQVariant()
		}

		sym := &syms[index.Row()]

		var val string

		switch index.Column() {
		case 0:
			val = sym.Name
		case 1:
			val = symTypeString(sym.Type)
		case 2:
			switch {
			case sym.Sect == 0:
				val = "0 (NO_SECT)"
			case int(sym.Sect) <= len(f.Sections):
				sect := f.Sections[sym.Sect-1]
				val = fmt.Sprintf("%d (%s,%s)", sym.Sect, sect.Seg, sect.Name)
			default:
				val = fmt.Sprintf("%d (?)", sym.Sect)
			}
		case 3:
			val = symDescString(f, sym)
		case 4:
			val = fmt.Sprintf("%#016x", sym.Value)
		}

		return core.NewQVariant14(val)
	})

	return symtab
}

type symInfo struct {
	Size     uint64
	Relocs   []macho.Reloc
	Sections []*macho.Section
	Symbols  []*macho.Symbol
}

func (m *SymtabModel) makeSymAddrInfo(f *macho.File) map[uint64]*symInfo {
	var syms []macho.Symbol
	if f.Symtab != nil {
		syms = f.Symtab.Syms
	}

	symAddrInfo := make(map[uint64]*symInfo)

	ssyms := make([]*macho.Symbol, 0, len(syms))
	for i := range syms {
		sym := &syms[i]
		if sym.Type&N_STAB == 0 && sym.Type&N_TYPE == N_SECT {
			ssyms = append(ssyms, sym)
		}
	}
	sort.Sort(byAddr(ssyms))
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
					info.Sections = append(info.Sections, sect)
				}
			}
		}
	}

	return symAddrInfo
}

func (m *SymtabModel) newReltabModel(f *macho.File, symAddrInfo map[uint64]*symInfo) func(*core.QModelIndex) core.QAbstractItemModel_ITF {
	var syms []macho.Symbol
	if f.Symtab != nil {
		syms = f.Symtab.Syms
	}

	reltabCache := make(map[int]core.QAbstractItemModel_ITF, len(syms))

	return func(index *core.QModelIndex) core.QAbstractItemModel_ITF {
		if !index.IsValid() {
			return nil
		}
		row := index.Row()
		if 0 <= row && row < len(syms) {
			if reltab, ok := reltabCache[row]; ok {
				return reltab
			}

			sym := &syms[row]

			if sym.Type&N_STAB == 0 && sym.Type&N_TYPE == N_SECT {
				var reltab core.QAbstractItemModel_ITF

				if symInfo := symAddrInfo[sym.Value]; symInfo != nil {
					reltab = newReltabModel(f, symInfo.Relocs, symInfo.Sections)
					if reltab != nil {
						proxy := core.NewQSortFilterProxyModel(nil)
						proxy.SetSourceModel(reltab)
						reltab = proxy
					}
				}

				reltabCache[row] = reltab

				return reltab
			}
		}
		return nil
	}
}

func symTypeString(typ uint8) string {
	var values []string
	switch {
	case typ&N_STAB != 0:
		values = append(values, fmt.Sprintf("%#02x (N_STAB)", typ&N_STAB))
	case typ&N_TYPE != 0:
		switch typ & N_TYPE {
		case N_ABS:
			values = append(values, fmt.Sprintf("%#02x (N_ABS)", typ&N_TYPE))
		case N_SECT:
			values = append(values, fmt.Sprintf("%#02x (N_SECT)", typ&N_TYPE))
		case N_PBUD:
			values = append(values, fmt.Sprintf("%#02x (N_PBUD)", typ&N_TYPE))
		case N_INDR:
			values = append(values, fmt.Sprintf("%#02x (N_INDR)", typ&N_TYPE))
		default:
			values = append(values, fmt.Sprintf("%#02x (?)", typ&N_TYPE))
		}
	default:
		values = append(values, "0x00 (N_UNDF)")
	}
	if typ&N_PEXT != 0 {
		values = append(values, "0x10 (N_PEXT)")
	}
	if typ&N_EXT != 0 {
		values = append(values, "0x01 (N_EXT)")
	}
	return strings.Join(values, "\n")
}

func symDescString(f *macho.File, sym *macho.Symbol) string {
	if sym.Type&N_STAB != 0 {
		// TODO handle stab
		return fmt.Sprintf("%#04x", sym.Desc)
	}
	desc := sym.Desc
	var vals []string
	if sym.Type&N_TYPE == N_UNDF || sym.Type&N_TYPE == N_PBUD {
		v := desc & REFERENCE_TYPE
		vals = append(vals, fmt.Sprintf("%#04x (%s)", v, ReferenceType(v)))
		desc ^= v
	}
	if desc&N_ARM_THUMB_DEF != 0 {
		vals = append(vals, "0x0008 (N_ARM_THUMB_DEF)")
		desc ^= N_ARM_THUMB_DEF
	}
	if sym.Type&N_EXT != 0 || sym.Type&N_PEXT != 0 {
		if desc&REFERENCED_DYNAMICALLY != 0 {
			vals = append(vals, "0x0010 (REFERENCED_DYNAMICALLY)")
			desc ^= REFERENCED_DYNAMICALLY
		}
	}
	if f.Type == macho.TypeObj {
		if desc&N_NO_DEAD_STRIP != 0 {
			vals = append(vals, "0x0020 (N_NO_DEAD_STRIP)")
			desc ^= N_NO_DEAD_STRIP
		}
		if sym.Type&N_TYPE == N_UNDF && sym.Type&N_EXT != 0 && sym.Value != 0 { // common symbol
			v := desc & (0x0f << 8)
			vals = append(vals, fmt.Sprintf("%#04x (alignment: %d)", v, v>>7))
			desc ^= v
		}
	} else {
		if desc&N_DESC_DISCARDED != 0 {
			vals = append(vals, "0x0020 (N_DESC_DISCARDED)")
			desc ^= N_DESC_DISCARDED
		}
	}
	switch {
	case sym.Type&N_TYPE == N_UNDF || sym.Type&N_TYPE == N_PBUD:
		if desc&N_WEAK_REF != 0 {
			vals = append(vals, "0x0040 (N_WEAK_REF)")
			desc ^= N_WEAK_REF
		}
		if desc&N_REF_TO_WEAK != 0 {
			vals = append(vals, "0x0080 (N_REF_TO_WEAK)")
			desc ^= N_REF_TO_WEAK
		}
	case sym.Type&N_EXT != 0 || sym.Type&N_PEXT != 0:
		if desc&N_WEAK_DEF != 0 {
			vals = append(vals, "0x0080 (N_WEAK_DEF)")
			desc ^= N_WEAK_DEF
		}
	}
	switch {
	case f.Type == macho.TypeObj:
		if desc&N_SYMBOL_RESOLVER != 0 {
			vals = append(vals, "0x0100 (N_SYMBOL_RESOLVER)")
			desc ^= N_SYMBOL_RESOLVER
		}
		if desc&N_ALT_ENTRY != 0 {
			vals = append(vals, "0x0200 (N_ALT_ENTRY)")
			desc ^= N_ALT_ENTRY
		}
	case f.Flags&macho.FlagTwoLevel != 0:
		if sym.Type&N_TYPE == N_UNDF || sym.Type&N_TYPE == N_PBUD {
			v := desc & (0xff << 8)
			switch ord := v >> 8; ord {
			case SELF_LIBRARY_ORDINAL:
				vals = append(vals, fmt.Sprintf("%#04x (SELF_LIBRARY_ORDINAL)", v))
			case DYNAMIC_LOOKUP_ORDINAL:
				vals = append(vals, fmt.Sprintf("%#04x (DYNAMIC_LOOKUP_ORDINAL)", v))
			case EXECUTABLE_ORDINAL:
				vals = append(vals, fmt.Sprintf("%#04x (EXECUTABLE_ORDINAL)", v))
			default:
				libs, err := f.ImportedLibraries()
				if err != nil {
					panic(err) // never happen
				}
				if int(ord) <= len(libs) {
					vals = append(vals, fmt.Sprintf("%#04x (%s)", v, libs[ord-1]))
				} else {
					// TODO warning
					vals = append(vals, fmt.Sprintf("%#04x (?)", v))
				}
			}
			desc ^= v
		}
	}
	if desc != 0 {
		// TODO warning
		vals = append(vals, fmt.Sprintf("%#04x (??)", desc))
	}
	if len(vals) == 0 {
		return "0x0000"
	}
	return strings.Join(vals, "\n")
}

type byAddr []*macho.Symbol

func (v byAddr) Len() int {
	return len(v)
}

func (v byAddr) Less(i, j int) bool {
	return v[i].Value < v[j].Value
}

func (v byAddr) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}
