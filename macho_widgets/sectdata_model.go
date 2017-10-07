package macho_widgets

import (
	"bytes"
	"debug/macho"
	"fmt"
	"strconv"

	"github.com/therecipe/qt/core"
	"github.com/therecipe/qt/gui"
)

func (f *File) NewSectionModel(typ string, sect *macho.Section, taddr uint64, tsize int64) core.QAbstractItemModel_ITF {
	switch typ {
	case "":
		return nil
	case "Code":
		return f.NewCodeSectionModel(sect, taddr, tsize)
	case "CString":
		return f.NewCStringSectionModel(sect, taddr, tsize)
	case "Float32":
		return f.NewFloat32SectionModel(sect, taddr, tsize)
	case "Float64":
		return f.NewFloat64SectionModel(sect, taddr, tsize)
	case "Float128":
		return f.NewFloat128SectionModel(sect, taddr, tsize)
	case "Pointer32":
		return f.NewPointer32SectionModel(sect, taddr, tsize)
	case "Data":
		return f.NewDataSectionModel(sect, taddr, tsize)
	default:
		panic("unreachable")
	}
}

// TODO support more section types
func (f *File) guessSectType(sect *macho.Section) string {
	if sect == nil {
		return ""
	}

	switch SectionType(sect.Flags & SECTION_TYPE) {
	case S_ZEROFILL:
		return ""
	case S_CSTRING_LITERALS:
		return "CString"
	case S_4BYTE_LITERALS:
		return "Float32"
	case S_8BYTE_LITERALS:
		return "Float64"
	case S_LITERAL_POINTERS:
		return "Pointer32"
	case S_NON_LAZY_SYMBOL_POINTERS:
		return "Pointer32"
	case S_LAZY_SYMBOL_POINTERS:
		return "Pointer32"
	case S_SYMBOL_STUBS:
		return "Code"
	case S_MOD_INIT_FUNC_POINTERS:
		return "Pointer32"
	case S_MOD_TERM_FUNC_POINTERS:
		return "Pointer32"
	case S_GB_ZEROFILL:
		return ""
	case S_16BYTE_LITERALS:
		return "Float128"
	default:
		if sect.Flags&S_ATTR_SOME_INSTRUCTIONS != 0 || sect.Flags&S_ATTR_PURE_INSTRUCTIONS != 0 {
			return "Code"
		} else {
			return "Data"
		}
	}
}

func (f *File) NewCodeSectionModel(sect *macho.Section, taddr uint64, tsize int64) core.QAbstractItemModel_ITF {
	disasm := f.disasmFunc()
	if disasm == nil {
		// TODO warning
		return nil
	}

	return f.newSectionModel(sect, taddr, tsize, disasm, true)
}

func (f *File) NewDataSectionModel(sect *macho.Section, taddr uint64, tsize int64) core.QAbstractItemModel_ITF {
	return f.newSectionModel(sect, taddr, tsize, func(data []byte, addr uint64) (string, int) {
		size := 8
		if len(data) < 8 {
			size = len(data)
		}
		return f.toASCII(data[:size]), size
	}, true)
}

func (f *File) NewCStringSectionModel(sect *macho.Section, taddr uint64, tsize int64) core.QAbstractItemModel_ITF {
	return f.newSectionModel(sect, taddr, tsize, func(data []byte, addr uint64) (string, int) {
		var size int
		if c := bytes.IndexByte(data, 0); c != -1 {
			size = c + 1
			val := strconv.Quote(string(data[:size-1]))
			return val, size
		} else {
			size = len(data)
			val := strconv.Quote(string(data[:size]))
			return val[:len(val)-1], size
		}
	}, false)
}

func (f *File) NewFloat32SectionModel(sect *macho.Section, taddr uint64, tsize int64) core.QAbstractItemModel_ITF {
	return f.newSectionModel(sect, taddr, tsize, func(data []byte, addr uint64) (string, int) {
		size := 4
		if len(data) < 4 {
			size = len(data)
		}
		return f.toFloat32(data[:size]), size
	}, false)
}

func (f *File) NewFloat64SectionModel(sect *macho.Section, taddr uint64, tsize int64) core.QAbstractItemModel_ITF {
	return f.newSectionModel(sect, taddr, tsize, func(data []byte, addr uint64) (string, int) {
		size := 8
		if len(data) < 8 {
			size = len(data)
		}
		return f.toFloat64(data[:size]), size
	}, false)
}

func (f *File) NewFloat128SectionModel(sect *macho.Section, taddr uint64, tsize int64) core.QAbstractItemModel_ITF {
	return f.newSectionModel(sect, taddr, tsize, func(data []byte, addr uint64) (string, int) {
		size := 16
		if len(data) < 16 {
			size = len(data)
		}
		return f.toFloat128(data[:size]), size
	}, false)
}

func (f *File) NewPointer32SectionModel(sect *macho.Section, taddr uint64, tsize int64) core.QAbstractItemModel_ITF {
	return f.newSectionModel(sect, taddr, tsize, func(data []byte, addr uint64) (string, int) {
		size := 4
		if len(data) < 4 {
			size = len(data)
		}
		return f.toPointer32(data[:size]), size
	}, false)
}

func (f *File) newSectionModel(sect *macho.Section, taddr uint64, tsize int64, valueFunc func(data []byte, addr uint64) (string, int), hasRel bool) core.QAbstractItemModel_ITF {
	m := gui.NewQStandardItemModel(nil)
	m.SetHorizontalHeaderItem(0, gui.NewQStandardItem2("Address"))
	m.SetHorizontalHeaderItem(1, gui.NewQStandardItem2("Data"))
	m.SetHorizontalHeaderItem(2, gui.NewQStandardItem2("Value"))
	if f.Type == macho.TypeObj && hasRel {
		m.SetHorizontalHeaderItem(3, gui.NewQStandardItem2("Type"))
		m.SetHorizontalHeaderItem(4, gui.NewQStandardItem2("PC Relative"))
		m.SetHorizontalHeaderItem(5, gui.NewQStandardItem2("Extern"))
		m.SetHorizontalHeaderItem(6, gui.NewQStandardItem2("Scattered"))
		m.SetHorizontalHeaderItem(7, gui.NewQStandardItem2("Relocatable"))
	}

	data := make([]byte, sect.Size)
	n, err := sect.ReadAt(data, 0)
	if n != len(data) || err != nil {
		// TODO warning
		return m
	}

	var info *SymInfo

	addr := sect.Addr

	for len(data) != 0 {
		if i := f.SymInfos[addr]; i != nil {
			// TODO make tree
			info = i
		}

		value, size := valueFunc(data, addr)

		addrItem := gui.NewQStandardItem2(fmt.Sprintf("%#016x", addr))
		dataItem := gui.NewQStandardItem2(fmt.Sprintf("% x", data[:size]))
		valueItem := gui.NewQStandardItem2(value)

		if f.Type == macho.TypeObj && hasRel {
			if info != nil {
				for i := range info.Relocs {
					r := info.Relocs[i]
					s := info.RelocSections[i]
					raddr := s.Addr + uint64(r.Addr)
					if addr <= raddr && raddr+uint64(1<<r.Len) <= addr+uint64(size) {
						rdata := data[raddr-addr : raddr-addr+uint64(1<<r.Len)]
						rdataString, rtarget := f.relocDataHtmlString(s, r, raddr-addr, rdata)
						addrItem.AppendRow([]*gui.QStandardItem{
							gui.NewQStandardItem2(fmt.Sprintf("%#016x", raddr)),
							gui.NewQStandardItem2(rdataString),
							gui.NewQStandardItem2(f.relocValueString(r)),
							gui.NewQStandardItem2(f.relocTypeString(r.Type)),
							gui.NewQStandardItem2(fmt.Sprintf("%t", r.Pcrel)),
							gui.NewQStandardItem2(fmt.Sprintf("%t", r.Extern)),
							gui.NewQStandardItem2(fmt.Sprintf("%t", r.Scattered)),
							gui.NewQStandardItem2(f.relocTargetHtmlString(rtarget)),
						})
					}
				}
			}
		}

		// TODO setdata (handle taddr)

		m.AppendRow([]*gui.QStandardItem{
			addrItem,
			dataItem,
			valueItem,
		})

		data = data[size:]
		addr += uint64(size)
	}

	return m
}