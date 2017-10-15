package macho_widgets

import (
	"debug/macho"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/therecipe/qt/core"
)

const SymbolItemRole = core.Qt__UserRole + 1

type SymtabModel struct {
	Symtab       core.QAbstractItemModel_ITF
	filterChar   byte
	filterExtern bool
}

func (f *File) NewSymtabModel() *SymtabModel {
	m := new(SymtabModel)

	symtab := core.NewQSortFilterProxyModel(nil)
	symtab.SetSourceModel(m.newSymtabModel(f))
	symtab.ConnectFilterAcceptsRow(func(sourceRow int, sourceParent *core.QModelIndex) bool {
		sm := symtab.SourceModel()

		c := byte(sm.Index(sourceRow, 1, sourceParent).Data(int(SymbolItemRole)).ToInt(true))

		if m.FilterExternOnly() && !unicode.IsUpper(rune(c)) {
			return false
		}

		fc := m.FilterChar()
		if fc != 0 && fc != '*' {
			if fc != c {
				fc = byte(unicode.ToLower(rune(fc)))
				if fc != c {
					return false
				}
			}
		} else {
			if c == '-' {
				return false
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

func (m *SymtabModel) SetFilterChar(c byte) {
	m.filterChar = c
	m.Symtab.(*core.QSortFilterProxyModel).InvalidateFilter()
}

func (m *SymtabModel) FilterChar() byte {
	return m.filterChar
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
			if index.Column() == 0 {
				return core.NewQVariant14(sym.Name)
			}
			return core.NewQVariant7(int(f.toSymChar(sym)))
		case core.Qt__DisplayRole:
			var val string

			switch index.Column() {
			case 0:
				val = sym.Name
			case 1:
				val = f.symTypeString(sym)
			case 2:
				val = f.symSectionString(sym)
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

func (f *File) symTypeString(sym *macho.Symbol) string {
	if sym.Type&N_STAB != 0 {
		return fmt.Sprintf("%#02x (N_STAB&%s)", sym.Type, StabType(sym.Type))
	}

	var values []string

	values = append(values, fmt.Sprintf("%#02x (%s)", sym.Type&N_TYPE, SymbolType(sym.Type&N_TYPE)))
	if sym.Type&N_PEXT != 0 {
		values = append(values, "0x10 (N_PEXT)")
	}
	if sym.Type&N_EXT != 0 {
		values = append(values, "0x01 (N_EXT)")
	}

	return strings.Join(values, "\n")
}

func (f *File) symSectionString(sym *macho.Symbol) string {
	if sym.Type&N_STAB != 0 {
		switch StabType(sym.Type) {
		case N_SO:
		case N_OSO:
			// TODO what's this?
			return fmt.Sprintf("%d (?)", sym.Sect)
		case N_FUN:
		case N_BNSYM:
		case N_ENSYM:
		case N_STSYM:
		case N_GSYM:
		default:
			// TODO handle more stab
		}
	}

	switch {
	case sym.Sect == 0:
		return "0 (NO_SECT)"
	case int(sym.Sect) <= len(f.Sections):
		s := f.Sections[sym.Sect-1]
		return fmt.Sprintf("%d (%s,%s)", sym.Sect, s.Seg, s.Name)
	default:
		return fmt.Sprintf("%d (?)", sym.Sect)
	}
}

func (f *File) symDescString(sym *macho.Symbol) string {
	if sym.Type&N_STAB != 0 {
		switch StabType(sym.Type) {
		case N_SO:
			if sym.Desc == 0 {
				return ""
			}
		case N_OSO:
			// TODO what's this?
			return fmt.Sprintf("%#04x (?)", sym.Desc)
		case N_FUN:
			if sym.Desc == 0 {
				return ""
			}
		case N_BNSYM:
			if sym.Desc == 0 {
				return ""
			}
		case N_ENSYM:
			if sym.Desc == 0 {
				return ""
			}
		case N_STSYM:
			if sym.Desc == 0 {
				return ""
			}
		case N_GSYM:
			if sym.Desc == 0 {
				return ""
			}
		default:
			// TODO handle more stab
		}
		return fmt.Sprintf("%#04x (?)", sym.Desc)
	}
	desc := sym.Desc
	var vals []string
	if SymbolType(sym.Type&N_TYPE) == N_UNDF || SymbolType(sym.Type&N_TYPE) == N_PBUD {
		if f.Type == macho.TypeObj && SymbolType(sym.Type&N_TYPE) == N_UNDF && sym.Value != 0 { // common symbol
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
	case SymbolType(sym.Type&N_TYPE) == N_UNDF || SymbolType(sym.Type&N_TYPE) == N_PBUD:
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
		if SymbolType(sym.Type&N_TYPE) == N_UNDF || SymbolType(sym.Type&N_TYPE) == N_PBUD {
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
		return ""
	}
	return strings.Join(vals, "\n")
}

func (f *File) symValueString(sym *macho.Symbol) string {
	switch {
	case sym.Type&N_STAB != 0:
		switch StabType(sym.Type) {
		case N_SO:
			if sym.Value == 0 {
				return ""
			}
		case N_OSO:
			return fmt.Sprintf("%#016x (mtime: %s)", sym.Value, time.Unix(int64(sym.Value), 0))
		case N_FUN:
			if sym.Name == "" && sym.Sect == 0 {
				return fmt.Sprintf("%#016x (size: %d)", sym.Value, sym.Value)
			}
		case N_BNSYM:
		case N_ENSYM:
			return fmt.Sprintf("%#016x (size: %d)", sym.Value, sym.Value)
		case N_STSYM:
		case N_GSYM:
			if sym.Value == 0 {
				return ""
			}
		default:
			// TODO handle more stab
		}
		return fmt.Sprintf("%#016x", sym.Value)
	case SymbolType(sym.Type&N_TYPE) == N_UNDF:
		if sym.Value != 0 { // common symbol
			return fmt.Sprintf("%#016x (size: %d)", sym.Value, sym.Value)
		}
	case SymbolType(sym.Type&N_TYPE) == N_PBUD:
		if sym.Value != 0 { // ?
			// TODO warning
			return fmt.Sprintf("%#016x (?)", sym.Value)
		}
	default:
		return fmt.Sprintf("%#016x", sym.Value)
	}
	return ""
}
