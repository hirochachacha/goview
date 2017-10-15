package macho_widgets

import (
	"bytes"
	"debug/macho"
	"fmt"
	"strconv"

	"github.com/therecipe/qt/core"
	"github.com/therecipe/qt/gui"
)

func (f *File) NewSymbolModel(typ string, sym *macho.Symbol, taddend int64, tsize int64) core.QAbstractItemModel_ITF {
	switch typ {
	case "":
		return nil
	case "Code":
		return f.newCodeSymbolModel(sym, taddend, tsize)
	case "CString":
		return f.newCStringSymbolModel(sym, taddend, tsize)
	case "Float32":
		return f.newFloat32SymbolModel(sym, taddend, tsize)
	case "Float64":
		return f.newFloat64SymbolModel(sym, taddend, tsize)
	case "Float128":
		return f.newFloat128SymbolModel(sym, taddend, tsize)
	case "Pointer32":
		return f.newPointer32SymbolModel(sym, taddend, tsize)
	case "Data":
		return f.newDataSymbolModel(sym, taddend, tsize)
	default:
		panic("unreachable")
	}
}

func (f *File) guessSymType(sym *macho.Symbol) string {
	if sym == nil {
		return ""
	}

	if sym.Type&N_STAB != 0 || SymbolType(sym.Type&N_TYPE) != N_SECT {
		return ""
	}

	if 0 < int(sym.Sect) && int(sym.Sect) <= len(f.Sections) {
		return f.guessSectType(f.Sections[sym.Sect-1])
	}

	return ""
}

func (f *File) newCodeSymbolModel(sym *macho.Symbol, taddend, tsize int64) core.QAbstractItemModel_ITF {
	disasm := f.disasmFunc()
	if disasm == nil {
		// TODO warning
		return nil
	}

	return f.newSymbolModel(sym, taddend, tsize, disasm, true)
}

func (f *File) newDataSymbolModel(sym *macho.Symbol, taddend, tsize int64) core.QAbstractItemModel_ITF {
	return f.newSymbolModel(sym, taddend, tsize, func(data []byte, addr uint64) (string, int) {
		size := 8
		if len(data) < 8 {
			size = len(data)
		}
		return f.toASCII(data[:size]), size
	}, true)
}

func (f *File) newCStringSymbolModel(sym *macho.Symbol, taddend, tsize int64) core.QAbstractItemModel_ITF {
	return f.newSymbolModel(sym, taddend, tsize, func(data []byte, addr uint64) (string, int) {
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

func (f *File) newFloat32SymbolModel(sym *macho.Symbol, taddend, tsize int64) core.QAbstractItemModel_ITF {
	return f.newSymbolModel(sym, taddend, tsize, func(data []byte, addr uint64) (string, int) {
		size := 4
		if len(data) < 4 {
			size = len(data)
		}
		return f.toFloat32(data[:size]), size
	}, false)
}

func (f *File) newFloat64SymbolModel(sym *macho.Symbol, taddend, tsize int64) core.QAbstractItemModel_ITF {
	return f.newSymbolModel(sym, taddend, tsize, func(data []byte, addr uint64) (string, int) {
		size := 8
		if len(data) < 8 {
			size = len(data)
		}
		return f.toFloat64(data[:size]), size
	}, false)
}

func (f *File) newFloat128SymbolModel(sym *macho.Symbol, taddend, tsize int64) core.QAbstractItemModel_ITF {
	return f.newSymbolModel(sym, taddend, tsize, func(data []byte, addr uint64) (string, int) {
		size := 16
		if len(data) < 16 {
			size = len(data)
		}
		return f.toFloat128(data[:size]), size
	}, false)
}

func (f *File) newPointer32SymbolModel(sym *macho.Symbol, taddend, tsize int64) core.QAbstractItemModel_ITF {
	return f.newSymbolModel(sym, taddend, tsize, func(data []byte, addr uint64) (string, int) {
		size := 4
		if len(data) < 4 {
			size = len(data)
		}
		return f.toPointer32(data[:size]), size
	}, false)
}

func (f *File) newSymbolModel(sym *macho.Symbol, taddend int64, tsize int64, valueFunc func(data []byte, addr uint64) (string, int), hasRel bool) core.QAbstractItemModel_ITF {
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

	addr := sym.Value
	sect := f.Sections[sym.Sect-1]
	info := f.SymInfos[addr]

	data := make([]byte, info.Size)
	n, err := sect.ReadAt(data, int64(addr-sect.Addr))
	if n != len(data) || err != nil {
		// TODO warning
		return nil
	}

	for len(data) != 0 {
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
