package macho_widgets

import (
	"debug/macho"
	"fmt"
	"math"

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
	switch {
	case r.Scattered:
		addr := uint64(r.Value)
		if s, base := lookup(addr); s != "" {
			if base == addr {
				return fmt.Sprintf("%#016x (%s)", r.Value, s)
			}
			return fmt.Sprintf("%#016x (%s%+x)", r.Value, s, addr-base)
		}
		return fmt.Sprintf("%#016x (?)", r.Value)
	case r.Extern:
		var syms []macho.Symbol
		if f.Symtab != nil {
			syms = f.Symtab.Syms
		}
		if len(syms) < math.MaxUint32 && 0 <= r.Value && r.Value < uint32(len(syms)) {
			return fmt.Sprintf("%d (%s)", r.Value, syms[r.Value].Name)
		} else {
			// TODO warning
			return fmt.Sprintf("%d (?)", r.Value)
		}
	default:
		if len(f.Sections) < math.MaxUint32 && 0 <= r.Value-1 && r.Value-1 < uint32(len(f.Sections)) {
			sect := f.Sections[r.Value-1]
			return fmt.Sprintf("%d (%s,%s)", r.Value, sect.Seg, sect.Name)
		} else {
			// TODO warning
			return fmt.Sprintf("%d (?)", r.Value)
		}
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

	var suffix string

	if uval != 0 {
		switch {
		case r.Scattered:
			suffix = fmt.Sprintf(" (addend: %+x)", int64(r.Value)-ival)
		case r.Extern:
			if f.Cpu == macho.CpuAmd64 {
				suffix = fmt.Sprintf(" (addend: %+x)", ival)
			} else {
				suffix = fmt.Sprintf(" (addend: %+x)", int64(s.Addr)-ival)
			}
		default:
			addr := uval
			if s, base := lookup(addr); s != "" {
				if base == addr {
					return fmt.Sprintf(" (addr: %#016x (%s))", addr, s)
				}
				return fmt.Sprintf(" (addr: %#016x (%s%+x))", addr, s, addr-base)
			}
			suffix = fmt.Sprintf(" (addr: %#016x)", addr)
		}
	}

	return fmt.Sprintf(fmt.Sprintf("%% %dx%%s", (uint64(len(data))+off)*3-1), data, suffix)
}
