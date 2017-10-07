package macho_widgets

import (
	"debug/macho"
	"fmt"
	"strings"

	"github.com/therecipe/qt/core"
	"github.com/therecipe/qt/gui"
)

type ReltabModel struct {
	Sections core.QAbstractItemModel_ITF
	reltabs  []core.QAbstractItemModel_ITF
}

func (f *File) NewReltabModel() *ReltabModel {
	m := new(ReltabModel)

	list := gui.NewQStandardItemModel(nil)

	reltabs := make([]core.QAbstractItemModel_ITF, len(f.Sections))

	for i, s := range f.Sections {
		list.AppendRow2(
			gui.NewQStandardItem2(fmt.Sprintf("%d (%s,%s) (%d)", i+1, s.Seg, s.Name, len(s.Relocs))),
		)

		reltab := m.newReltabModel(f, s)

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

func (m *ReltabModel) newReltabModel(f *File, s *macho.Section) core.QAbstractItemModel_ITF {
	header := []string{"Address", "Address (Offset)", "Value", "Type", "Len", "PC Relative", "Extern", "Scattered"}

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
				val = f.relocValueString(r)
			case 3: // Type
				val = f.relocTypeString(r.Type)
			case 4: // Length
				val = f.relocLenString(r.Len)
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

func (f *File) relocValueString(r macho.Reloc) string {
	suffix := " (?)"

	switch {
	case r.Scattered:
		addr := uint64(r.Value)
		if s := f.symAddrString(addr, false); s != "" {
			suffix = fmt.Sprintf(` (%s)`, s)
		}
		return fmt.Sprintf("%#016x%s", r.Value, suffix)
	case r.Extern:
		if s := f.symIndexString(r.Value); s != "" {
			suffix = fmt.Sprintf(` (%s)`, s)
		} else {
			// TODO warning
		}
		return fmt.Sprintf("%d%s", r.Value, suffix)
	default:
		if s := f.sectNumString(r.Value); s != "" {
			suffix = fmt.Sprintf(` (%s)`, s)
		} else {
			// TODO warning
		}
		return fmt.Sprintf("%d%s", r.Value, suffix)
	}
}

func (f *File) relocTypeString(typ uint8) string {
	switch f.Cpu {
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

func (f *File) relocLenString(len uint8) string {
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

type RelocTarget struct {
	Symnum  int
	Symaddr uint64 // exist if Symnum != -1
	Addend  int64
	Size    uint8
}

func (f *File) relocDataHtmlString(s *macho.Section, r macho.Reloc, off uint64, data []byte) (string, *RelocTarget) {
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

	var target *RelocTarget

	suffix := " (?)"

	switch f.Cpu {
	case macho.Cpu386:
		switch macho.RelocTypeGeneric(r.Type) {
		case macho.GENERIC_RELOC_VANILLA:
			switch {
			case r.Scattered:
				rs := f.symAddrString(uint64(r.Value), true)
				if r.Pcrel {
					pc := s.Addr + uint64(r.Addr) + uint64(1<<r.Len)
					suffix = fmt.Sprintf(" (addend: %#+x(%%eip) = %+d)", rs, ival, ival+int64(pc))
					target = &RelocTarget{
						Symnum:  -1,
						Symaddr: uint64(r.Value),
						Addend:  ival + int64(pc),
						Size:    1 << r.Len,
					}
				} else {
					suffix = fmt.Sprintf(" (addend : %+d)", rs, ival)
					target = &RelocTarget{
						Symnum:  -1,
						Symaddr: uint64(r.Value),
						Addend:  ival,
						Size:    1 << r.Len,
					}
				}
			case r.Extern:
				if r.Pcrel {
					pc := s.Addr + uint64(r.Addr) + uint64(1<<r.Len)
					suffix = fmt.Sprintf(" (addend: %#+x(%%eip) = %+d)", ival, ival+int64(pc))
					target = &RelocTarget{
						Symnum: int(r.Value),
						Addend: ival + int64(pc),
						Size:   1 << r.Len,
					}
				} else {
					suffix = fmt.Sprintf(" (addend: %+d)", ival)
					target = &RelocTarget{
						Symnum: int(r.Value),
						Addend: ival,
						Size:   1 << r.Len,
					}
				}
			default:
				if r.Pcrel {
					pc := s.Addr + uint64(r.Addr) + uint64(1<<r.Len)
					suffix = fmt.Sprintf(" (addr: %#x(%%eip) = %#x)", uval, uval+pc)
					target = &RelocTarget{
						Symnum:  -1,
						Symaddr: uval + pc,
						Size:    1 << r.Len,
					}
				} else {
					suffix = fmt.Sprintf(" (addr: %#x)", uval)
					target = &RelocTarget{
						Symnum:  -1,
						Symaddr: uval,
						Size:    1 << r.Len,
					}
				}
			}
		case macho.GENERIC_RELOC_PAIR:
		case macho.GENERIC_RELOC_SECTDIFF, macho.GENERIC_RELOC_LOCAL_SECTDIFF:
			for i, r1 := range s.Relocs {
				if r == r1 {
					if i+1 < len(s.Relocs) {
						n := s.Relocs[i+1]
						if n.Scattered {
							if macho.RelocTypeGeneric(n.Type) == macho.GENERIC_RELOC_PAIR {
								ns := f.symAddrString(uint64(n.Value), true)
								rs := f.symAddrString(uint64(r.Value), true)
								addend := ival + int64(n.Value) - int64(r.Value)
								suffix = fmt.Sprintf(" (addend: %#x+%s-%s = %+d)", ival, ns, rs, addend)
								target = &RelocTarget{
									Symnum:  -1,
									Symaddr: uint64(r.Value),
									Addend:  addend,
									Size:    1 << r.Len,
								}
							}
						}
					}
					break
				}
			}
		case macho.GENERIC_RELOC_PB_LA_PTR:
		case macho.GENERIC_RELOC_TLV:
			suffix = fmt.Sprintf(" (addend: %+d)", ival)
			target = &RelocTarget{
				Symnum: int(r.Value),
				Addend: ival,
				Size:   1 << r.Len,
			}
		}
	case macho.CpuAmd64:
		if macho.RelocTypeX86_64(r.Type) != macho.X86_64_RELOC_SUBTRACTOR {
			if r.Extern {
				switch macho.RelocTypeX86_64(r.Type) {
				default:
					suffix = fmt.Sprintf(" (addend: %+d)", ival)
					target = &RelocTarget{
						Symnum: int(r.Value),
						Addend: ival,
						Size:   1 << r.Len,
					}
				case macho.X86_64_RELOC_SIGNED_1:
					suffix = fmt.Sprintf(" (addend: %d+1 = %+d)", ival, ival+1)
					target = &RelocTarget{
						Symnum: int(r.Value),
						Addend: ival + 1,
						Size:   1 << r.Len,
					}
				case macho.X86_64_RELOC_SIGNED_2:
					suffix = fmt.Sprintf(" (addend: %d+2 = %+d)", ival, ival+2)
					target = &RelocTarget{
						Symnum: int(r.Value),
						Addend: ival + 2,
						Size:   1 << r.Len,
					}
				case macho.X86_64_RELOC_SIGNED_4:
					suffix = fmt.Sprintf(" (addend: %d+4 = %+d)", ival, ival+4)
					target = &RelocTarget{
						Symnum: int(r.Value),
						Addend: ival + 4,
						Size:   1 << r.Len,
					}
				}
			} else {
				pc := s.Addr + uint64(r.Addr) + uint64(1<<r.Len)

				switch macho.RelocTypeX86_64(r.Type) {
				default:
					suffix = fmt.Sprintf(" (addr: %#x(%%rip) = %#x)", uval, uval+pc)
					target = &RelocTarget{
						Symnum:  -1,
						Symaddr: uval + pc,
						Size:    1 << r.Len,
					}
				case macho.X86_64_RELOC_SIGNED_1:
					suffix = fmt.Sprintf(" (addr: %#x(%%rip)+1 = %#x)", uval, uval+pc+1)
					target = &RelocTarget{
						Symnum:  -1,
						Symaddr: uval + pc + 1,
						Size:    1 << r.Len,
					}
				case macho.X86_64_RELOC_SIGNED_2:
					suffix = fmt.Sprintf(" (addr: %#x(%%rip)+2 = %#x)", uval, uval+pc+2)
					target = &RelocTarget{
						Symnum:  -1,
						Symaddr: uval + pc + 2,
						Size:    1 << r.Len,
					}
				case macho.X86_64_RELOC_SIGNED_4:
					suffix = fmt.Sprintf(" (addr: %#x(%%rip)+4 = %#x)", uval, uval+pc+4)
					target = &RelocTarget{
						Symnum:  -1,
						Symaddr: uval + pc + 4,
						Size:    1 << r.Len,
					}
				}
			}
		}
	case macho.CpuArm:
		// TODO
	case macho.CpuArm | 0x01000000:
		// TODO
	}

	return fmt.Sprintf(fmt.Sprintf(`<body>%% %dx%%s</body>`, (uint64(len(data))+off)*3-1), data, suffix), target
}

func (f *File) relocTargetHtmlString(t *RelocTarget) string {
	if t == nil {
		return "<body>?</body>"
	}
	if t.Symnum != -1 {
		if t.Symnum < 0 || t.Symnum >= len(f.Syms) {
			if t.Addend == 0 {
				return "<body>?</body>"
			}
			return fmt.Sprintf("<body>?%+d</body>", t.Addend)
		}
		sym := &f.Syms[t.Symnum]
		if t.Addend == 0 {
			return fmt.Sprintf(`<body><a href="/symbol/%d?addend=0&size=%d">%s</a></body>`, t.Symnum, t.Size, sym.Name)
		}
		return fmt.Sprintf(`<body><a href="/symbol/%d?addend=%d&size=%d">%s%+d</a></body>`, t.Symnum, t.Addend, t.Size, sym.Name, t.Addend)
	}
	addr := t.Symaddr
	if t.Addend < 0 {
		addr -= uint64(-t.Addend)
	} else {
		addr += uint64(t.Addend)
	}
	if s, base := f.SymLookup(addr); s != "" {
		info := f.SymInfos[base]
		ss := make([]string, len(info.SymbolIndices))
		for i, si := range info.SymbolIndices {
			sym := &f.Syms[si]
			if base == addr {
				ss[i] = fmt.Sprintf(`<a href="/symbol/%d?addend=0&size=%d">%s</a>`, si, sym.Name, t.Size)
			} else {
				ss[i] = fmt.Sprintf(`<a href="/symbol/%d?addend=%d&size=%d">%s%+d</a>`, si, addr-base, t.Size, sym.Name, addr-base)
			}
		}
		return fmt.Sprintf(`<body>%s</body>`, strings.Join(ss, "|"))
	}
	return fmt.Sprintf(`<body><a href="/address/%d?size=%d">%s</a></body>`, addr, t.Size, f.symAddrString(addr, true))
}
