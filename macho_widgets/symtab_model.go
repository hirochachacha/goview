package macho_widgets

import (
	"debug/macho"
	"fmt"
	"strings"
	"unicode"

	"github.com/therecipe/qt/core"
)

const SymbolItemRole = core.Qt__UserRole + 1

type SymtabModel struct {
	Symtab       core.QAbstractItemModel_ITF
	filterType   byte
	filterExtern bool
}

func (f *File) NewSymtabModel() *SymtabModel {
	m := new(SymtabModel)

	symtab := core.NewQSortFilterProxyModel(nil)
	symtab.SetSourceModel(m.newSymtabModel(f))
	symtab.ConnectFilterAcceptsRow(func(sourceRow int, sourceParent *core.QModelIndex) bool {
		sm := symtab.SourceModel()

		var typ uint8
		var typDone bool

		if m.FilterExternOnly() {
			typ = uint8(sm.Index(sourceRow, 1, sourceParent).Data(int(SymbolItemRole)).ToUInt(true))
			typDone = true
			if typ&N_EXT == 0 {
				return false
			}
		}

		fc := m.FilterType()
		if fc != 0 && fc != '*' {
			if !typDone {
				typ = uint8(sm.Index(sourceRow, 1, sourceParent).Data(int(SymbolItemRole)).ToUInt(true))
			}
			sect := uint8(sm.Index(sourceRow, 2, sourceParent).Data(int(SymbolItemRole)).ToUInt(true))
			val := sm.Index(sourceRow, 4, sourceParent).Data(int(SymbolItemRole)).ToULongLong(true)
			c := f.toSymChar(typ, sect, val)
			if fc != c {
				fc = byte(unicode.ToLower(rune(fc)))
				if fc != c {
					return false
				}
			}
		}

		name := sm.Index(sourceRow, 0, sourceParent).Data(int(SymbolItemRole)).ToString()

		return symtab.FilterRegExp().IndexIn(name, 0, core.QRegExp__CaretAtZero) != -1
	})

	m.Symtab = symtab

	return m
}

func (m *SymtabModel) SetFilterName(s string) {
	m.Symtab.(*core.QSortFilterProxyModel).SetFilterRegExp2(s)
}

func (m *SymtabModel) SetFilterType(typ byte) {
	m.filterType = typ
	m.Symtab.(*core.QSortFilterProxyModel).InvalidateFilter()
}

func (m *SymtabModel) FilterType() byte {
	return m.filterType
}

func (m *SymtabModel) SetFilterExternOnly(b bool) {
	m.filterExtern = b
	m.Symtab.(*core.QSortFilterProxyModel).InvalidateFilter()
}

func (m *SymtabModel) FilterExternOnly() bool {
	return m.filterExtern
}

func (m *SymtabModel) newSymtabModel(f *File) core.QAbstractItemModel_ITF {
	header := []string{"Name", "Type", "Sect", "Desc", "value"}

	symtab := core.NewQAbstractTableModel(nil)
	symtab.ConnectRowCount(func(parent *core.QModelIndex) int {
		return len(f.Syms)
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
		sym := &f.Syms[index.Row()]

		switch core.Qt__ItemDataRole(role) {
		case SymbolItemRole:
			switch index.Column() {
			case 0:
				return core.NewQVariant14(sym.Name)
			case 1:
				return core.NewQVariant8(uint(sym.Type))
			case 2:
				return core.NewQVariant8(uint(sym.Sect))
			case 3:
				return core.NewQVariant8(uint(sym.Desc))
			case 4:
				return core.NewQVariant10(sym.Value)
			}

			return core.NewQVariant()
		case core.Qt__DisplayRole:
			var val string

			switch index.Column() {
			case 0:
				val = sym.Name
			case 1:
				val = f.symTypeString(sym.Type)
			case 2:
				val = f.symSectionString(sym.Sect)
			case 3:
				val = f.symDescString(sym)
			case 4:
				val = f.symValueString(sym)
			}

			return core.NewQVariant14(val)
		}

		return core.NewQVariant()
	})

	return symtab
}

func (f *File) symTypeString(typ uint8) string {
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

func (f *File) symSectionString(sect uint8) string {
	switch {
	case sect == 0:
		return "0 (NO_SECT)"
	case int(sect) <= len(f.Sections):
		s := f.Sections[sect-1]
		return fmt.Sprintf("%d (%s,%s)", sect, s.Seg, s.Name)
	default:
		return fmt.Sprintf("%d (?)", sect)
	}
}

func (f *File) symDescString(sym *macho.Symbol) string {
	if sym.Type&N_STAB != 0 {
		// TODO handle stab
		return fmt.Sprintf("%#04x", sym.Desc)
	}
	desc := sym.Desc
	var vals []string
	if sym.Type&N_TYPE == N_UNDF || sym.Type&N_TYPE == N_PBUD {
		if f.Type == macho.TypeObj && sym.Type&N_TYPE == N_UNDF && sym.Value != 0 { // common symbol
			v := desc & (0x0f << 8)
			vals = append(vals, fmt.Sprintf("%#04x (alignment: %d)", v, v>>7))
			desc ^= v
		} else {
			v := desc & REFERENCE_TYPE
			vals = append(vals, fmt.Sprintf("%#04x (%s)", v, ReferenceType(v)))
			desc ^= v
		}
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

func (f *File) symValueString(sym *macho.Symbol) string {
	switch {
	case sym.Type&N_STAB != 0:
		// TODO handle stab
		return fmt.Sprintf("%#016x", sym.Value)
	case sym.Type&N_TYPE == N_UNDF:
		if sym.Value != 0 { // common symbol
			return fmt.Sprintf("%d (size: %d)", sym.Value, sym.Value)
		}
	case sym.Type&N_TYPE == N_PBUD:
		if sym.Value != 0 { // ?
			// TODO warning
			return fmt.Sprintf("%d (?)", sym.Value)
		}
	default:
		return fmt.Sprintf("%#016x", sym.Value)
	}
	return ""
}
