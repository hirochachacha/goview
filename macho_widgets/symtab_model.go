package macho_widgets

import (
	"debug/macho"
	"fmt"
	"strings"

	"github.com/therecipe/qt/core"
)

type SymtabModel struct {
	Symtab core.QAbstractItemModel_ITF
}

func NewSymtabModel(f *macho.File) (*SymtabModel, error) {
	m := new(SymtabModel)

	var syms []macho.Symbol

	if f.Symtab != nil {
		syms = f.Symtab.Syms
	}

	header := []string{"Name", "Type", "Section", "Description", "Value"}

	qtab := core.NewQAbstractTableModel(nil)
	qtab.ConnectRowCount(func(parent *core.QModelIndex) int {
		return len(syms)
	})
	qtab.ConnectColumnCount(func(parent *core.QModelIndex) int {
		return 5
	})
	qtab.ConnectHeaderData(func(section int, orientation core.Qt__Orientation, role int) *core.QVariant {
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
	qtab.ConnectData(func(index *core.QModelIndex, role int) *core.QVariant {
		if role != int(core.Qt__DisplayRole) {
			return core.NewQVariant()
		}

		sym := syms[index.Row()]

		var val string

		switch index.Column() {
		case 0:
			val = sym.Name
		case 1:
			val = fmt.Sprintf("%#x (%s)", sym.Type, SymbolType(sym.Type))
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
			if sym.Type&N_STAB != 0 {
				val = fmt.Sprintf("%#x", sym.Desc)
			} else {
				var vals []string
				vals = append(vals, fmt.Sprintf("%#x", sym.Desc))
				if sym.Type&N_TYPE == N_UNDF || sym.Type&N_TYPE == N_PBUD {
					vals = append(vals, fmt.Sprintf("%#x (%s)", sym.Desc&REFERENCE_TYPE, ReferenceType(sym.Desc&REFERENCE_TYPE)))
				}
				if sym.Desc&N_ARM_THUMB_DEF != 0 {
					vals = append(vals, "0x8 (N_ARM_THUMB_DEF)")
				}
				if sym.Type&N_EXT != 0 {
					if sym.Desc&REFERENCED_DYNAMICALLY != 0 {
						vals = append(vals, "0x10 (REFERENCED_DYNAMICALLY)")
					}
				}
				if f.Type == macho.TypeObj {
					if sym.Desc&N_NO_DEAD_STRIP != 0 {
						vals = append(vals, "0x20 (N_NO_DEAD_STRIP)")
					}
				} else {
					if sym.Desc&N_DESC_DISCARDED != 0 {
						vals = append(vals, "0x20 (N_DESC_DISCARDED)")
					}
				}
				switch {
				case sym.Type&N_TYPE == N_UNDF || sym.Type&N_TYPE == N_PBUD:
					if sym.Desc&N_WEAK_REF != 0 {
						vals = append(vals, "0x40 (N_WEAK_REF)")
					}
					if sym.Desc&N_REF_TO_WEAK != 0 {
						vals = append(vals, "0x80 (N_REF_TO_WEAK)")
					}
				case sym.Type&N_EXT != 0:
					if sym.Desc&N_WEAK_DEF != 0 {
						vals = append(vals, "0x80 (N_WEAK_DEF)")
					}
				}
				switch {
				case f.Type == macho.TypeObj:
					if sym.Desc&N_SYMBOL_RESOLVER != 0 {
						vals = append(vals, "0x100 (N_SYMBOL_RESOLVER)")
					}
					if sym.Desc&N_ALT_ENTRY != 0 {
						vals = append(vals, "0x200 (N_ALT_ENTRY)")
					}
				case f.Flags&macho.FlagTwoLevel != 0:
					if sym.Type&N_TYPE == N_UNDF || sym.Type&N_TYPE == N_PBUD {
						ord := (sym.Desc >> 8) & 0xff
						switch ord {
						case SELF_LIBRARY_ORDINAL:
							vals = append(vals, fmt.Sprintf("%#x (Library Ordinal = 0x0 (SELF_LIBRARY_ORDINAL))", sym.Desc&(0xff<<8)))
						case DYNAMIC_LOOKUP_ORDINAL:
							vals = append(vals, fmt.Sprintf("%#x (Library Ordinal = 0xfe (DYNAMIC_LOOKUP_ORDINAL))", sym.Desc&(0xff<<8)))
						case EXECUTABLE_ORDINAL:
							vals = append(vals, fmt.Sprintf("%#x (Library Ordinal = 0xff (EXECUTABLE_ORDINAL))", sym.Desc&(0xff<<8)))
						default:
							libs, err := f.ImportedLibraries()
							if err != nil {
								panic(err) // never happen
							}
							if int(ord) <= len(libs) {
								vals = append(vals, fmt.Sprintf("%#x (Library Ordinal = %d (%s))", sym.Desc&(0xff<<8), ord, libs[ord-1]))
							} else {
								vals = append(vals, fmt.Sprintf("%#x (Library Ordinal = %d (?))", sym.Desc&(0xff<<8), ord))
							}
						}
					}
				}
				val = strings.Join(vals, "\t")
			}
		case 4:
			val = fmt.Sprintf("%#x", sym.Value)
		}

		return core.NewQVariant14(val)
	})

	m.Symtab = qtab

	return m, nil
}
