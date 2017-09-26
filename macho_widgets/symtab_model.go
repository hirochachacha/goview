package macho_widgets

import (
	"debug/macho"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/therecipe/qt/core"
	"github.com/therecipe/qt/gui"
)

type SymtabModel struct {
	Symtab    core.QAbstractItemModel_ITF
	RawReltab func(index *core.QModelIndex) core.QAbstractItemModel_ITF
}

func NewSymtabModel(f *macho.File) (*SymtabModel, error) {
	symtab := core.NewQSortFilterProxyModel(nil)
	symtab.SetSourceModel(newSymtabModel(f))

	return &SymtabModel{
		Symtab:    symtab,
		RawReltab: newReltabModel(f),
	}, nil
}

func (m *SymtabModel) SetFilter(s string) {
	m.Symtab.(*core.QSortFilterProxyModel).SetFilterRegExp2(s)
}

func (m *SymtabModel) Reltab(index *core.QModelIndex) core.QAbstractItemModel_ITF {
	return m.RawReltab(m.Symtab.(*core.QSortFilterProxyModel).MapToSource(index))
}

func newSymtabModel(f *macho.File) core.QAbstractItemModel_ITF {
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

func newReltabModel(f *macho.File) func(*core.QModelIndex) core.QAbstractItemModel_ITF {
	var syms []macho.Symbol
	if f.Symtab != nil {
		syms = f.Symtab.Syms
	}

	type relocInfo struct {
		*macho.Reloc
		Sect *macho.Section
	}

	type symInfo struct {
		Symbol    *macho.Symbol
		Relocs    []*relocInfo
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
				r := &sect.Relocs[i]
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
				info.Relocs = append(info.Relocs, &relocInfo{Reloc: r, Sect: sect})
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

			sym := &syms[row]

			if sym.Type&N_STAB == 0 && sym.Type&N_TYPE == N_SECT {
				reltab := gui.NewQStandardItemModel(nil)
				reltab.SetHorizontalHeaderItem(0, gui.NewQStandardItem2("Address"))
				reltab.SetHorizontalHeaderItem(1, gui.NewQStandardItem2("Value"))
				reltab.SetHorizontalHeaderItem(2, gui.NewQStandardItem2("Type"))
				reltab.SetHorizontalHeaderItem(3, gui.NewQStandardItem2("Length"))
				reltab.SetHorizontalHeaderItem(4, gui.NewQStandardItem2("PC Relative"))
				reltab.SetHorizontalHeaderItem(5, gui.NewQStandardItem2("Extern"))
				reltab.SetHorizontalHeaderItem(6, gui.NewQStandardItem2("Scattered"))

				if symInfo := symAddrInfo[sym.Value]; symInfo != nil {
					for i, r := range symInfo.Relocs {
						reltab.SetItem(i, 0, gui.NewQStandardItem2(fmt.Sprintf("%#x+%#x (%s,%s)", r.Addr, r.Sect.Addr, r.Sect.Seg, r.Sect.Name)))
						switch {
						case r.Scattered:
							reltab.SetItem(i, 1, gui.NewQStandardItem2(fmt.Sprintf("%#x (?)", r.Value)))
						case r.Extern:
							if len(syms) < math.MaxUint32 && 0 <= r.Value && r.Value < uint32(len(syms)) {
								reltab.SetItem(i, 1, gui.NewQStandardItem2(fmt.Sprintf("%#x (%s)", r.Value, syms[r.Value].Name)))
							} else {
								// TODO warning
								reltab.SetItem(i, 1, gui.NewQStandardItem2(fmt.Sprintf("%#x (?)", r.Value)))
							}
						default:
							if len(f.Sections) < math.MaxUint32 && 0 <= r.Value-1 && r.Value-1 < uint32(len(f.Sections)) {
								sect := f.Sections[r.Value-1]
								reltab.SetItem(i, 1, gui.NewQStandardItem2(fmt.Sprintf("%#x (%s,%s)", r.Value, sect.Seg, sect.Name)))
							} else {
								// TODO warning
								reltab.SetItem(i, 1, gui.NewQStandardItem2(fmt.Sprintf("%#x (?)", r.Value)))
							}
						}
						reltab.SetItem(i, 2, gui.NewQStandardItem2(relocString(r.Type, f.Cpu)))
						switch r.Len {
						case 0:
							reltab.SetItem(i, 3, gui.NewQStandardItem2("0 (byte)"))
						case 1:
							reltab.SetItem(i, 3, gui.NewQStandardItem2("1 (word)"))
						case 2:
							reltab.SetItem(i, 3, gui.NewQStandardItem2("2 (long)"))
						case 3:
							reltab.SetItem(i, 3, gui.NewQStandardItem2("3 (quad)"))
						default:
							panic("unreachable")
						}
						reltab.SetItem(i, 4, gui.NewQStandardItem2(fmt.Sprintf("%t", r.Pcrel)))
						if r.Scattered {
							reltab.SetItem(i, 6, gui.NewQStandardItem2(fmt.Sprintf("%t", r.Scattered)))
						} else {
							reltab.SetItem(i, 5, gui.NewQStandardItem2(fmt.Sprintf("%t", r.Extern)))
						}
					}
				}

				proxy := core.NewQSortFilterProxyModel(nil)
				proxy.SetSourceModel(reltab)

				reltabCache[row] = proxy

				return proxy
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

func relocString(r uint8, cpu macho.Cpu) string {
	switch cpu {
	case macho.Cpu386:
		return fmt.Sprintf("%d (%s)", r, macho.RelocTypeGeneric(r))
	case macho.CpuAmd64:
		return fmt.Sprintf("%d (%s)", r, macho.RelocTypeX86_64(r))
	case macho.CpuArm:
		return fmt.Sprintf("%d (%s)", r, macho.RelocTypeARM(r))
	case macho.CpuArm | 0x01000000:
		return fmt.Sprintf("%d (%s)", r, macho.RelocTypeARM64(r))
	default:
		// TODO warning
		return fmt.Sprintf("%#x (?)", r)
	}
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
