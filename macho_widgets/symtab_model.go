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
	Symtab core.QAbstractItemModel_ITF
	Reltab func(index *core.QModelIndex) core.QAbstractItemModel_ITF
}

func NewSymtabModel(f *macho.File) (*SymtabModel, error) {
	return &SymtabModel{
		Symtab: newSymtabModel(f),
		Reltab: newReltabModel(f),
	}, nil
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
				// TODO
				val = strings.Join(vals, "\t")
			}
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
				sym := ssyms[k-1]
				if k == len(ssyms) {
					tsect := f.Sections[sym.Sect-1]
					if sect.Addr+uint64(r.Addr) > tsect.Addr+tsect.Size {
						// TODO handle unbinded relocations
						continue
					}
				}
				addr := sym.Value
				info := symAddrInfo[addr]
				if info == nil {
					info = new(symInfo)
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

			reltab := gui.NewQStandardItemModel(nil)
			reltab.SetHorizontalHeaderItem(0, gui.NewQStandardItem2("Address"))
			reltab.SetHorizontalHeaderItem(1, gui.NewQStandardItem2("Value"))
			reltab.SetHorizontalHeaderItem(2, gui.NewQStandardItem2("Type"))
			reltab.SetHorizontalHeaderItem(3, gui.NewQStandardItem2("Length"))
			reltab.SetHorizontalHeaderItem(4, gui.NewQStandardItem2("PC Relative"))
			reltab.SetHorizontalHeaderItem(5, gui.NewQStandardItem2("Extern"))
			reltab.SetHorizontalHeaderItem(6, gui.NewQStandardItem2("Scattered"))
			if sym.Type&N_STAB == 0 && sym.Type&N_TYPE == N_SECT {
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
						switch f.Cpu {
						case macho.Cpu386:
							reltab.SetItem(i, 2, gui.NewQStandardItem2(fmt.Sprintf("%d (%s)", r.Type, macho.RelocTypeGeneric(r.Type))))
						case macho.CpuAmd64:
							reltab.SetItem(i, 2, gui.NewQStandardItem2(fmt.Sprintf("%d (%s)", r.Type, macho.RelocTypeX86_64(r.Type))))
						case macho.CpuArm:
							reltab.SetItem(i, 2, gui.NewQStandardItem2(fmt.Sprintf("%d (%s)", r.Type, macho.RelocTypeARM(r.Type))))
						case macho.CpuArm | 0x01000000:
							reltab.SetItem(i, 2, gui.NewQStandardItem2(fmt.Sprintf("%d (%s)", r.Type, macho.RelocTypeARM64(r.Type))))
						default:
							// TODO warning
							reltab.SetItem(i, 2, gui.NewQStandardItem2(fmt.Sprintf("%#x (?)", r.Type)))
						}
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

				reltabCache[row] = reltab

				return reltab
			}
		}
		return nil
	}
}

func relocString(typ uint8, cpu macho.Cpu) string {
	switch cpu {
	case macho.Cpu386:
		return macho.RelocTypeGeneric(typ).String()
	case macho.CpuAmd64:
		return macho.RelocTypeX86_64(typ).String()
	case macho.CpuArm:
		return macho.RelocTypeARM(typ).String()
	case macho.CpuArm | 0x01000000:
		return macho.RelocTypeARM64(typ).String()
	default:
		return "?"
	}
}

type byAddr []*macho.Symbol

func (v byAddr) Len() int {
	return len(v)
}

func (v byAddr) Less(i, j int) bool {
	return v[i].Value < v[i].Value
}

func (v byAddr) Swap(i, j int) {
	v[i], v[i] = v[j], v[i]
}
