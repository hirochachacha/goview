package macho_widgets

import (
	"bytes"
	"debug/dwarf"
	"debug/macho"
	"fmt"
	"strconv"
	"strings"

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
	case "DwarfType":
		return f.newDwarfTypeSymbolModel(sym, taddend, tsize)
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
	if f.isZeroSym(sym) {
		return f.newSymbolModel(sym, taddend, tsize, func(data []byte, addr uint64) (string, int) {
			size := 8
			if len(data) < 8 {
				size = len(data)
			}
			return "zero-fill", size
		}, false)
	}

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

func (f *File) decodeValue(data []byte, typ dwarf.Type, zero bool, label bool) (val string, ok bool) {
	bo := f.ByteOrder

	switch typ := typ.(type) {
	case *dwarf.TypedefType:
		val, ok = f.decodeValue(data, typ.Type, zero, false)
		if !ok {
			return "", false
		}
	case *dwarf.QualType:
		val, ok = f.decodeValue(data, typ.Type, zero, false)
		if !ok {
			return "", false
		}
	case *dwarf.StructType:
		size := typ.Size()
		if int64(len(data)) < size {
			// TODO warning
			return "", false
		}
		if zero {
			val = "{0}"
		} else {
			vals := make([]string, len(typ.Field))
			for i, field := range typ.Field {
				ftyp := field.Type
				foff := field.ByteOffset
				fsize := ftyp.Size()
				if int64(len(data)) < foff+fsize {
					// TODO warning
					return "", false
				}
				if bsize := field.BitSize; bsize != 0 {
					// TODO I don't know how to deal bit size
					return "", false
				} else {
					val, ok = f.decodeValue(data[foff:foff+fsize], ftyp, false, true)
					if !ok {
						return "", false
					}
					vals[i] = fmt.Sprintf(".%s = %s", field.Name, val)
				}
			}
			val = fmt.Sprintf("{%s}", strings.Join(vals, "; "))
		}
	case *dwarf.ArrayType:
		etyp := typ.Type
		esize := etyp.Size()
		n := typ.Count
		size := esize * n
		if int64(len(data)) < size {
			// TODO warning
			return "", false
		}
		if zero {
			val = "{0}"
		} else {
			vals := make([]string, n)
			if !zero {
				for i := int64(0); i < int64(len(vals)); i++ {
					val, ok = f.decodeValue(data[esize*i:esize*i+esize], etyp, false, false)
					if !ok {
						return "", false
					}
					vals[i] = val
				}
			}
			val = fmt.Sprintf("{%s}", strings.Join(vals, ", "))
		}
	case *dwarf.PtrType:
		size := typ.Size()
		if int64(len(data)) < size {
			// TODO warning
			return "", false
		}
		var v uint64
		if !zero {
			switch size {
			case 1:
				v = uint64(data[0])
			case 2:
				v = uint64(bo.Uint16(data[:2]))
			case 4:
				v = uint64(bo.Uint32(data[:4]))
			case 8:
				v = bo.Uint64(data[:8])
			default:
				// TODO
				return "", false
			}
		}
		val = fmt.Sprintf("%#x", v)
	case *dwarf.BoolType:
		size := typ.Size()
		if int64(len(data)) < size {
			// TODO warning
			return "", false
		}
		if size != 1 {
			// TODO warning
			return "", false
		}
		var v uint8
		if !zero {
			v = data[0]
		}
		val = fmt.Sprintf("%d", v)
	case *dwarf.CharType:
		size := typ.Size()
		if int64(len(data)) < size {
			// TODO warning
			return "", false
		}
		if size != 1 {
			// TODO warning
			return "", false
		}
		var v int8
		if !zero {
			v = int8(data[0])
		}
		val = fmt.Sprintf("%q", v)
	case *dwarf.UcharType:
		size := typ.Size()
		if int64(len(data)) < size {
			// TODO warning
			return "", false
		}
		if size != 1 {
			// TODO warning
			return "", false
		}
		var v uint8
		if !zero {
			v = data[0]
		}
		val = fmt.Sprintf("%q", v)
	case *dwarf.IntType:
		size := typ.Size()
		if int64(len(data)) < size {
			// TODO warning
			return "", false
		}
		var v int64
		if !zero {
			switch size {
			case 1:
				v = int64(int8(data[0]))
			case 2:
				v = int64(int16(bo.Uint16(data[:2])))
			case 4:
				v = int64(int32(bo.Uint32(data[:4])))
			case 8:
				v = int64(bo.Uint64(data[:8]))
			default:
				// TODO
				return "", false
			}
		}
		val = fmt.Sprintf("%#x", v)
	case *dwarf.UintType:
		size := typ.Size()
		if int64(len(data)) < size {
			// TODO warning
			return "", false
		}
		var v uint64
		if !zero {
			switch size {
			case 1:
				v = uint64(data[0])
			case 2:
				v = uint64(bo.Uint16(data[:2]))
			case 4:
				v = uint64(bo.Uint32(data[:4]))
			case 8:
				v = bo.Uint64(data[:8])
			default:
				// TODO
				return "", false
			}
		}
		val = fmt.Sprintf("%#x", v)
	case *dwarf.FloatType:
		size := typ.Size()
		if int64(len(data)) < size {
			// TODO warning
			return "", false
		}
		if zero {
			val = "0"
		} else {
			switch size {
			case 4:
				val = f.toFloat32(data[:size])
			case 8:
				val = f.toFloat64(data[:size])
			case 16:
				val = f.toFloat128(data[:size])
			default:
				return "", false
			}
		}
	case *dwarf.ComplexType:
		size := typ.Size()
		if int64(len(data)) < size {
			// TODO warning
			return "", false
		}
		if zero {
			val = "0"
		} else {
			switch size {
			case 8:
				val = fmt.Sprintf("%s + %si", f.toFloat32(data[:4]), f.toFloat32(data[4:8]))
			case 16:
				val = fmt.Sprintf("%s + %si", f.toFloat64(data[:8]), f.toFloat64(data[8:16]))
			default:
				return "", false
			}
		}
	case *dwarf.EnumType:
		size := typ.Size()
		if int64(len(data)) < size {
			// TODO warning
			return "", false
		}
		var v int64
		if !zero {
			switch size {
			case 1:
				v = int64(data[0])
			case 2:
				v = int64(int16(bo.Uint16(data[:2])))
			case 4:
				v = int64(int32(bo.Uint32(data[:4])))
			case 8:
				v = int64(bo.Uint64(data[:8]))
			default:
				// TODO
				return "", false
			}
		}
		val = fmt.Sprintf("%#x", v)
		for _, ev := range typ.Val {
			if ev.Val == v {
				val = ev.Name
				break
			}
		}
	default:
		// TODO
		return "", false
	}

	if label {
		if strings.ContainsRune(typ.String(), ' ') {
			if strings.HasPrefix(val, "{") {
				val = fmt.Sprintf("(%s)%s", typ, val)
			} else {
				val = fmt.Sprintf("(%s)(%s)", typ, val)
			}
		} else {
			if strings.HasPrefix(val, "{") {
				val = fmt.Sprintf("%s%s", typ, val)
			} else {
				val = fmt.Sprintf("%s(%s)", typ, val)
			}
		}
	}

	return val, true
}

func (f *File) newDwarfTypeSymbolModel(sym *macho.Symbol, taddend, tsize int64) core.QAbstractItemModel_ITF {
	var typ dwarf.Type

	d, err := f.DWARF()
	if err != nil {
		return nil
	}

	r := d.Reader()

L:
	for {
		e, err := r.Next()
		if err != nil {
			return nil
		}
		if e == nil {
			break
		}
		switch e.Tag {
		case dwarf.TagVariable:
			name, _ := e.Val(dwarf.AttrName).(string)
			if strings.HasPrefix(sym.Name, "_") && name == sym.Name[1:] || name == sym.Name {
				typOff, _ := e.Val(dwarf.AttrType).(dwarf.Offset)
				if typOff != 0 {
					typ, err = d.Type(typOff)
					if err != nil {
						return nil
					}
				}
				break L
			}
		}
		if e.Tag != dwarf.TagCompileUnit {
			r.SkipChildren()
		}
	}

	if typ == nil {
		return nil
	}

	return f.newSymbolModel(sym, taddend, tsize, func(data []byte, addr uint64) (string, int) {
		val, ok := f.decodeValue(data, typ, f.isZeroSym(sym), true)
		if !ok {
			return f.toASCII(data), len(data)
		}
		return val, len(data)
	}, true)
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
