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
}

func NewSymtabModel(f *macho.File) *SymtabModel {
	m := new(SymtabModel)

	symtab := core.NewQSortFilterProxyModel(nil)
	symtab.SetSourceModel(m.newSymtabModel(f))

	reltab := m.newReltabModel(f)

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
			val = fmt.Sprintf("%#x", sym.Value)
		}

		return core.NewQVariant14(val)
	})

	return symtab
}

func (m *SymtabModel) newReltabModel(f *macho.File) func(*core.QModelIndex) core.QAbstractItemModel_ITF {
	var syms []macho.Symbol
	if f.Symtab != nil {
		syms = f.Symtab.Syms
	}

	type symInfo struct {
		Symbol    *macho.Symbol
		Relocs    []macho.Reloc
		Sections  []*macho.Section
		SameAddrs []*macho.Symbol
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
		for _, sect := range f.Sections {
			for i := range sect.Relocs {
				r := sect.Relocs[i]
				k := sort.Search(len(ssyms), func(i int) bool {
					return ssyms[i].Value > sect.Addr+uint64(r.Addr)
				})
				if k == 0 {
					// TODO warning
					continue
				}
				tsym := ssyms[k-1]
				if k == len(ssyms) {
					tsect := f.Sections[tsym.Sect-1]
					if sect.Addr+uint64(r.Addr) > tsect.Addr+tsect.Size {
						// TODO handle unbinded relocations
						continue
					}
				}
				addr := tsym.Value
				info := symAddrInfo[addr]
				if info == nil {
					info = new(symInfo)
					info.Symbol = tsym
					for k := k - 1; k >= 0 && ssyms[k].Value == addr; k-- {
						info.SameAddrs = append(info.SameAddrs, ssyms[k])
					}
					symAddrInfo[addr] = info
				}
				info.Relocs = append(info.Relocs, r)
				info.Sections = append(info.Sections, sect)
			}
		}
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

			var reltab core.QAbstractItemModel_ITF

			sym := &syms[row]

			if sym.Type&N_STAB == 0 && sym.Type&N_TYPE == N_SECT {
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
		values = append(values, fmt.Sprintf("%#x (N_STAB)", typ&N_STAB))
	case typ&N_TYPE != 0:
		switch typ & N_TYPE {
		case N_ABS:
			values = append(values, fmt.Sprintf("%#x (N_ABS)", typ&N_TYPE))
		case N_SECT:
			values = append(values, fmt.Sprintf("%#x (N_SECT)", typ&N_TYPE))
		case N_PBUD:
			values = append(values, fmt.Sprintf("%#x (N_PBUD)", typ&N_TYPE))
		case N_INDR:
			values = append(values, fmt.Sprintf("%#x (N_INDR)", typ&N_TYPE))
		default:
			values = append(values, fmt.Sprintf("%#x (?)", typ&N_TYPE))
		}
	default:
		values = append(values, "0x0 (N_UNDF)")
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
		return fmt.Sprintf("%#x", sym.Desc)
	}
	desc := sym.Desc
	var vals []string
	if sym.Type&N_TYPE == N_UNDF || sym.Type&N_TYPE == N_PBUD {
		v := desc & REFERENCE_TYPE
		vals = append(vals, fmt.Sprintf("%#x (%s)", v, ReferenceType(v)))
		desc ^= v
	}
	if desc&N_ARM_THUMB_DEF != 0 {
		vals = append(vals, "0x8 (N_ARM_THUMB_DEF)")
		desc ^= N_ARM_THUMB_DEF
	}
	if sym.Type&N_EXT != 0 {
		if desc&REFERENCED_DYNAMICALLY != 0 {
			vals = append(vals, "0x10 (REFERENCED_DYNAMICALLY)")
			desc ^= REFERENCED_DYNAMICALLY
		}
	}
	if f.Type == macho.TypeObj {
		if desc&N_NO_DEAD_STRIP != 0 {
			vals = append(vals, "0x20 (N_NO_DEAD_STRIP)")
			desc ^= N_NO_DEAD_STRIP
		}
	} else {
		if desc&N_DESC_DISCARDED != 0 {
			vals = append(vals, "0x20 (N_DESC_DISCARDED)")
			desc ^= N_DESC_DISCARDED
		}
	}
	switch {
	case sym.Type&N_TYPE == N_UNDF || sym.Type&N_TYPE == N_PBUD:
		if desc&N_WEAK_REF != 0 {
			vals = append(vals, "0x40 (N_WEAK_REF)")
			desc ^= N_WEAK_REF
		}
		if desc&N_REF_TO_WEAK != 0 {
			vals = append(vals, "0x80 (N_REF_TO_WEAK)")
			desc ^= N_REF_TO_WEAK
		}
	case sym.Type&N_EXT != 0:
		if desc&N_WEAK_DEF != 0 {
			vals = append(vals, "0x80 (N_WEAK_DEF)")
			desc ^= N_WEAK_DEF
		}
	}
	switch {
	case f.Type == macho.TypeObj:
		if desc&N_SYMBOL_RESOLVER != 0 {
			vals = append(vals, "0x100 (N_SYMBOL_RESOLVER)")
			desc ^= N_SYMBOL_RESOLVER
		}
		if desc&N_ALT_ENTRY != 0 {
			vals = append(vals, "0x200 (N_ALT_ENTRY)")
			desc ^= N_ALT_ENTRY
		}
	case f.Flags&macho.FlagTwoLevel != 0:
		if sym.Type&N_TYPE == N_UNDF || sym.Type&N_TYPE == N_PBUD {
			v := desc & (0xff << 8)
			switch ord := v >> 8; ord {
			case SELF_LIBRARY_ORDINAL:
				vals = append(vals, fmt.Sprintf("%#x (SELF_LIBRARY_ORDINAL)", v))
			case DYNAMIC_LOOKUP_ORDINAL:
				vals = append(vals, fmt.Sprintf("%#x (DYNAMIC_LOOKUP_ORDINAL)", v))
			case EXECUTABLE_ORDINAL:
				vals = append(vals, fmt.Sprintf("%#x (EXECUTABLE_ORDINAL)", v))
			default:
				libs, err := f.ImportedLibraries()
				if err != nil {
					panic(err) // never happen
				}
				if int(ord) <= len(libs) {
					vals = append(vals, fmt.Sprintf("%#x (%s)", v, libs[ord-1]))
				} else {
					// TODO warning
					vals = append(vals, fmt.Sprintf("%#x (?)", v))
				}
			}
			desc ^= v
		}
	}
	if desc != 0 {
		// TODO warning
		vals = append(vals, fmt.Sprintf("%#x (??)", desc))
	}
	if len(vals) == 0 {
		return "0x0"
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
