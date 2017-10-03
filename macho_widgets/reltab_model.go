package macho_widgets

import (
	"debug/macho"
	"fmt"

	"github.com/therecipe/qt/core"
	"github.com/therecipe/qt/gui"
)

type ReltabModel struct {
	Sections core.QAbstractItemModel_ITF
	reltabs  []core.QAbstractItemModel_ITF
}

func NewReltabModel(f *macho.File, lookup symLookup) *ReltabModel {
	m := new(ReltabModel)

	list := gui.NewQStandardItemModel(nil)

	reltabs := make([]core.QAbstractItemModel_ITF, len(f.Sections))

	for i, s := range f.Sections {
		list.AppendRow2(
			gui.NewQStandardItem2(fmt.Sprintf("%d (%s,%s) (%d)", i+1, s.Seg, s.Name, len(s.Relocs))),
		)

		reltab := m.newReltabModel(f, s, lookup)

		proxy := core.NewQSortFilterProxyModel(nil)
		proxy.SetSourceModel(reltab)

		reltabs[i] = proxy
	}

	m.Sections = list
	m.reltabs = reltabs

	return m
}

func (m *ReltabModel) Reltab(index *core.QModelIndex) core.QAbstractItemModel_ITF {
	if !index.IsValid() {
		return nil
	}
	if row := index.Row(); 0 <= row && row < len(m.reltabs) {
		return m.reltabs[row]
	}
	return nil
}

func (m *ReltabModel) newReltabModel(f *macho.File, s *macho.Section, lookup symLookup) core.QAbstractItemModel_ITF {
	header := []string{"Address", "Address Offset", "Value", "Type", "Length", "PC Relative", "Extern", "Scattered"}

	reltab := core.NewQAbstractTableModel(nil)
	reltab.ConnectRowCount(func(parent *core.QModelIndex) int {
		return len(s.Relocs)
	})
	reltab.ConnectColumnCount(func(parent *core.QModelIndex) int {
		return len(header)
	})
	reltab.ConnectHeaderData(func(section int, orientation core.Qt__Orientation, role int) *core.QVariant {
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
	reltab.ConnectData(func(index *core.QModelIndex, role int) *core.QVariant {
		if role != int(core.Qt__DisplayRole) {
			return core.NewQVariant()
		}
		if !index.IsValid() {
			return core.NewQVariant()
		}
		if row := index.Row(); 0 <= row && row < len(s.Relocs) {
			r := s.Relocs[row]

			var val string

			switch index.Column() {
			case 0: // Addr
				val = fmt.Sprintf("%#016x", s.Addr+uint64(r.Addr))
			case 1: // Addr Offset
				val = fmt.Sprintf("%#016x", r.Addr)
			case 2: // Value
				val = relocValueString(f, r, lookup)
			case 3: // Type
				val = relocTypeString(r.Type, f.Cpu)
			case 4: // Length
				val = relocLenString(r.Len)
			case 5: // Pcrel
				val = fmt.Sprintf("%t", r.Pcrel)
			case 6: // Extern
				if !r.Scattered {
					val = fmt.Sprintf("%t", r.Extern)
				}
			case 7: // Scattered
				if r.Scattered {
					val = fmt.Sprintf("%t", r.Scattered)
				}
			}
			return core.NewQVariant14(val)
		}
		return core.NewQVariant()
	})

	return reltab
}

func relocValueString(f *macho.File, r macho.Reloc, lookup func(addr uint64) (string, uint64)) string {
	suffix := " (?)"

	switch {
	case r.Scattered:
		addr := uint64(r.Value)
		if s := symAddrString(addr, lookup, false); s != "" {
			suffix = fmt.Sprintf(` (%s)`, s)
		}
		return fmt.Sprintf("%#016x%s", r.Value, suffix)
	case r.Extern:
		if s := symIndexString(f, r.Value); s != "" {
			suffix = fmt.Sprintf(` (%s)`, s)
		} else {
			// TODO warning
		}
		return fmt.Sprintf("%d%s", r.Value, suffix)
	default:
		if s := sectNumString(f, r.Value); s != "" {
			suffix = fmt.Sprintf(` (%s)`, s)
		} else {
			// TODO warning
		}
		return fmt.Sprintf("%d%s", r.Value, suffix)
	}
}

func relocTypeString(typ uint8, cpu macho.Cpu) string {
	switch cpu {
	case macho.Cpu386:
		return fmt.Sprintf("%d (%s)", typ, macho.RelocTypeGeneric(typ))
	case macho.CpuAmd64:
		return fmt.Sprintf("%d (%s)", typ, macho.RelocTypeX86_64(typ))
	case macho.CpuArm:
		return fmt.Sprintf("%d (%s)", typ, macho.RelocTypeARM(typ))
	case macho.CpuArm | 0x01000000:
		return fmt.Sprintf("%d (%s)", typ, macho.RelocTypeARM64(typ))
	default:
		// TODO warning
		return fmt.Sprintf("%d (?)", typ)
	}
}

func relocLenString(len uint8) string {
	switch len {
	case 0:
		return "0 (byte)"
	case 1:
		return "1 (word)"
	case 2:
		return "2 (long)"
	case 3:
		return "3 (quad)"
	default:
		panic("unreachable")
	}
}

func relocDataString(f *macho.File, s *macho.Section, r macho.Reloc, off uint64, data []byte, lookup symLookup) string {
	var uval uint64
	var ival int64

	switch len(data) {
	case 0:
		val := data[0]
		uval = uint64(val)
		ival = int64(int8(val))
	case 2:
		val := f.ByteOrder.Uint16(data)
		uval = uint64(val)
		ival = int64(int16(val))
	case 4:
		val := f.ByteOrder.Uint32(data)
		uval = uint64(val)
		ival = int64(int32(val))
	case 8:
		val := f.ByteOrder.Uint64(data)
		uval = val
		ival = int64(val)
	default:
		panic("unreachable")
	}

	suffix := " (?)"

	var addr uint64

	switch f.Cpu {
	case macho.Cpu386:
		switch macho.RelocTypeGeneric(r.Type) {
		case macho.GENERIC_RELOC_VANILLA:
			switch {
			case r.Scattered:
				rs := symAddrString(uint64(r.Value), lookup, true)
				if r.Pcrel {
					suffix = fmt.Sprintf(" ((%s+addend)(%%rip): %#016x)", rs, ival)
					pc := s.Addr + uint64(r.Addr) + uint64(1<<r.Len)
					if ival < 0 {
						addr = uint64(r.Value) - uint64(-ival) + pc
					} else {
						addr = uint64(r.Value) + uint64(ival) + pc
					}
				} else {
					suffix = fmt.Sprintf(" (%s+addend : %#016x)", rs, ival)
					if ival < 0 {
						addr = uint64(r.Value) - uint64(-ival)
					} else {
						addr = uint64(r.Value) + uint64(ival)
					}
				}
			case r.Extern:
				rsym := symIndex(f, r.Value)
				if r.Pcrel {
					suffix = fmt.Sprintf(" ((addend)(%%rip): %d)", ival)
					if rsym != nil {
						pc := s.Addr + uint64(r.Addr) + uint64(1<<r.Len)
						if ival < 0 {
							addr = rsym.Value - uint64(-ival) + pc
						} else {
							addr = rsym.Value + uint64(ival) + pc
						}
					}
				} else {
					suffix = fmt.Sprintf(" (addend: %d)", ival)
					if rsym != nil {
						if ival < 0 {
							addr = rsym.Value - uint64(-ival)
						} else {
							addr = rsym.Value + uint64(ival)
						}
					}
				}
			default:
				suffix = fmt.Sprintf(" (addr: %s)", symAddrString(uval, lookup, true))
				addr = uval
			}
		case macho.GENERIC_RELOC_PAIR:
		case macho.GENERIC_RELOC_SECTDIFF, macho.GENERIC_RELOC_LOCAL_SECTDIFF:
			for i, r1 := range s.Relocs {
				if r == r1 {
					if i+1 < len(s.Relocs) {
						n := s.Relocs[i+1]
						if n.Scattered {
							if macho.RelocTypeGeneric(n.Type) == macho.GENERIC_RELOC_PAIR {
								rs2 := symAddrString(uint64(n.Value), lookup, true)
								rs1 := symAddrString(uint64(r.Value), lookup, true)
								suffix = fmt.Sprintf(" (addend+%s-%s: %d)", rs1, rs2, ival)
								if ival < 0 {
									addr = uint64(n.Value) - uint64(-ival)
								} else {
									addr = uint64(n.Value) + uint64(ival)
								}
							}
						}
					}
					break
				}
			}
		case macho.GENERIC_RELOC_PB_LA_PTR:
		case macho.GENERIC_RELOC_TLV:
			suffix = fmt.Sprintf(" (addr: %s)", symAddrString(uval, lookup, true))
			addr = uval
		}
	case macho.CpuAmd64:
		if macho.RelocTypeX86_64(r.Type) != macho.X86_64_RELOC_SUBTRACTOR {
			if r.Extern {
				rsym := symIndex(f, r.Value)
				switch macho.RelocTypeX86_64(r.Type) {
				case macho.X86_64_RELOC_SIGNED_1:
					suffix = fmt.Sprintf(" (addend-1: %d)", ival)
					if rsym != nil {
						if ival < 0 {
							addr = rsym.Value - uint64(-ival) + 1
						} else {
							addr = rsym.Value + uint64(ival) + 1
						}
					}
				case macho.X86_64_RELOC_SIGNED_2:
					suffix = fmt.Sprintf(" (addend-2: %d)", ival)
					if rsym != nil {
						if ival < 0 {
							addr = rsym.Value - uint64(-ival) + 2
						} else {
							addr = rsym.Value + uint64(ival) + 2
						}
					}
				case macho.X86_64_RELOC_SIGNED_4:
					suffix = fmt.Sprintf(" (addend-4: %d)", ival)
					if rsym != nil {
						if ival < 0 {
							addr = rsym.Value - uint64(-ival) + 4
						} else {
							addr = rsym.Value + uint64(ival) + 4
						}
					}
				default:
					suffix = fmt.Sprintf(" (addend: %d)", ival)
					if rsym != nil {
						if ival < 0 {
							addr = rsym.Value - uint64(-ival)
						} else {
							addr = rsym.Value + uint64(ival)
						}
					}
				}
			} else {
				suffix = fmt.Sprintf(" (addr: %s)", symAddrString(uval, lookup, true))
				addr = uval
			}
		}
	case macho.CpuArm:
	case macho.CpuArm | 0x01000000:
	}

	_ = addr

	return fmt.Sprintf(fmt.Sprintf("%% %dx%%s", (uint64(len(data))+off)*3-1), data, suffix)
}
