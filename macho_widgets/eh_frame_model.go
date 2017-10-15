//go:generate stringer -type=DW_EH_PE_basicType,DW_EH_PE_modType -output eh_frame_model_string.go

package macho_widgets

// reference:
// http://www.airs.com/blog/archives/460
// https://refspecs.linuxfoundation.org/LSB_3.0.0/LSB-PDA/LSB-PDA/ehframechpt.html
// http://www.hexblog.com/wp-content/uploads/2012/06/Recon-2012-Skochinsky-Compiler-Internals.pdf

import (
	"bytes"
	"debug/macho"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/therecipe/qt/core"
	"github.com/therecipe/qt/gui"
)

type DW_EH_PE_basicType uint8

type DW_EH_PE_modType uint8

const (
	// basic pointer encodings

	DW_EH_PE_ptr     DW_EH_PE_basicType = 0x00
	DW_EH_PE_uleb128 DW_EH_PE_basicType = 0x01
	DW_EH_PE_udata2  DW_EH_PE_basicType = 0x02
	DW_EH_PE_udata4  DW_EH_PE_basicType = 0x03
	DW_EH_PE_udata8  DW_EH_PE_basicType = 0x04
	DW_EH_PE_signed  DW_EH_PE_basicType = 0x08
	DW_EH_PE_sleb128 DW_EH_PE_basicType = 0x09
	DW_EH_PE_sdata2  DW_EH_PE_basicType = 0x0a
	DW_EH_PE_sdata4  DW_EH_PE_basicType = 0x0b
	DW_EH_PE_sdata8  DW_EH_PE_basicType = 0x0c

	// modifiers

	DW_EH_PE_absptr  DW_EH_PE_modType = 0x00
	DW_EH_PE_pcrel   DW_EH_PE_modType = 0x10
	DW_EH_PE_textrel DW_EH_PE_modType = 0x20
	DW_EH_PE_datarel DW_EH_PE_modType = 0x30
	DW_EH_PE_funcrel DW_EH_PE_modType = 0x40
	DW_EH_PE_aligned DW_EH_PE_modType = 0x50

	// indirect
	DW_EH_PE_indirect uint8 = 0x80

	// omit

	DW_EH_PE_omit uint8 = 0xff

	// mask

	DW_EH_PE_basic    uint8 = 0x0f
	DW_EH_PE_modifier uint8 = 0xf0
)

type parser struct {
	f        *File
	cieInfos map[uint64]*cieInfo
	scratch  [8]byte
	cieNum   int
}

type cieInfo struct {
	aug  string
	penc uint8  // Personlity encoding
	pptr uint64 // Personlity pointer
	fenc uint8  // FDE encoding
	lenc uint8  // LSDA encoding

	cfiItem *gui.QStandardItem
	fdeNum  int
}

func (f *File) NewEHFrameSectionModel(sect *macho.Section) core.QAbstractItemModel_ITF {
	m := gui.NewQStandardItemModel(nil)
	m.SetHorizontalHeaderItem(0, gui.NewQStandardItem2("Address"))
	m.SetHorizontalHeaderItem(1, gui.NewQStandardItem2("Data"))
	m.SetHorizontalHeaderItem(2, gui.NewQStandardItem2("Name"))
	m.SetHorizontalHeaderItem(3, gui.NewQStandardItem2("Interpretation"))

	p := &parser{
		f:        f,
		cieInfos: make(map[uint64]*cieInfo),
	}

	off := uint64(0)

	for off < sect.Size {
		length, extended, ok := p.populateItem(m, sect, off)
		if !ok {
			return nil
		}

		if extended {
			if f.Cpu&0x01000000 == 0 { // 32bit
				off += ((8 + length) + (4 - 1)) &^ (4 - 1)
			} else {
				off += ((8 + length) + (8 - 1)) &^ (8 - 1)
			}
		} else {
			if f.Cpu&0x01000000 == 0 { // 32bit
				off += ((4 + length) + (4 - 1)) &^ (4 - 1)
			} else {
				off += ((4 + length) + (8 - 1)) &^ (8 - 1)
			}
		}
	}

	return m
}

func (p *parser) populateItem(m *gui.QStandardItemModel, sect *macho.Section, top uint64) (length uint64, extended bool, ok bool) {
	item := gui.NewQStandardItem()

	bo := p.f.ByteOrder

	off := int64(top)

	_, err := sect.ReadAt(p.scratch[:8], off)
	if err != nil {
		// TODO warning
		return 0, false, false
	}
	if l := bo.Uint32(p.scratch[:4]); l == 0xFFFFFFFF {
		extended = true
		length = bo.Uint64(p.scratch[:8])
		item.AppendRow([]*gui.QStandardItem{
			gui.NewQStandardItem2(fmt.Sprintf("%#016x", sect.Addr+uint64(off))),
			gui.NewQStandardItem2(fmt.Sprintf("% x", p.scratch[:8])),
			gui.NewQStandardItem2("Length"),
			gui.NewQStandardItem2(fmt.Sprintf("%d", length)),
		})
		off += 8
	} else {
		extended = false
		length = uint64(l)
		item.AppendRow([]*gui.QStandardItem{
			gui.NewQStandardItem2(fmt.Sprintf("%#016x", sect.Addr+uint64(off))),
			gui.NewQStandardItem2(fmt.Sprintf("% x", p.scratch[:4])),
			gui.NewQStandardItem2("Length"),
			gui.NewQStandardItem2(fmt.Sprintf("%d", length)),
		})
		off += 4
	}

	end := off + int64(length)

	_, err = sect.ReadAt(p.scratch[:4], off)
	if err != nil {
		// TODO warning
		return 0, false, false
	}
	cieId := bo.Uint32(p.scratch[:4])
	if cieId == 0 {
		cfiItem := gui.NewQStandardItem2(fmt.Sprintf("CFI %d", p.cieNum))
		p.cieInfos[top] = &cieInfo{
			cfiItem: cfiItem,
		}
		m.AppendRow2(cfiItem)

		item.SetText(fmt.Sprintf("CIE %d", p.cieNum))

		item.AppendRow([]*gui.QStandardItem{
			gui.NewQStandardItem2(fmt.Sprintf("%#016x", sect.Addr+uint64(off))),
			gui.NewQStandardItem2(fmt.Sprintf("% x", p.scratch[:4])),
			gui.NewQStandardItem2("CIE ID"),
			gui.NewQStandardItem2(fmt.Sprintf("%d", cieId)),
		})
		off += 4
		if ok := p.populateCIEItem(item, sect, top, off, end, cieId); !ok {
			// TODO warning
			return 0, false, false
		}

		cfiItem.AppendRow2(item)

		p.cieNum++
	} else {
		cieTop := uint64(off) - uint64(cieId)
		info, ok := p.cieInfos[cieTop]
		if !ok {
			// TODO warning
			return 0, false, false
		}

		cfiItem := info.cfiItem

		item.SetText(fmt.Sprintf("FDE %d", info.fdeNum))

		item.AppendRow([]*gui.QStandardItem{
			gui.NewQStandardItem2(fmt.Sprintf("%#016x", sect.Addr+uint64(off))),
			gui.NewQStandardItem2(fmt.Sprintf("% x", p.scratch[:4])),
			gui.NewQStandardItem2("CIE Pointer"),
			gui.NewQStandardItem2(fmt.Sprintf("addr = -%#x(%%rip) = %#x", cieId, sect.Addr+uint64(off)-uint64(cieId))),
		})
		off += 4
		if ok := p.populateFDEItem(item, sect, off, end, info); !ok {
			// TODO warning
			return 0, false, false
		}

		cfiItem.AppendRow2(item)

		info.fdeNum++
	}

	return length, extended, true
}

func (p *parser) populateCIEItem(item *gui.QStandardItem, sect *macho.Section, top uint64, off, end int64, id uint32) (ok bool) {
	_, err := sect.ReadAt(p.scratch[:1], off)
	if err != nil {
		// TODO warning
		return
	}
	version := p.scratch[0]
	item.AppendRow([]*gui.QStandardItem{
		gui.NewQStandardItem2(fmt.Sprintf("%#016x", sect.Addr+uint64(off))),
		gui.NewQStandardItem2(fmt.Sprintf("% x", p.scratch[:1])),
		gui.NewQStandardItem2("Version"),
		gui.NewQStandardItem2(fmt.Sprintf("%d", version)),
	})
	off++

	n, err := sect.ReadAt(p.scratch[:], off)
	if n <= 0 && err != nil {
		// TODO warning
		return
	}
	i := bytes.IndexByte(p.scratch[:n], 0)
	if i == -1 {
		// TODO warning
		return
	}
	aug := string(p.scratch[:i])
	item.AppendRow([]*gui.QStandardItem{
		gui.NewQStandardItem2(fmt.Sprintf("%#016x", sect.Addr+uint64(off))),
		gui.NewQStandardItem2(fmt.Sprintf("% x", p.scratch[:1])),
		gui.NewQStandardItem2("Augumentation String"),
		gui.NewQStandardItem2(aug),
	})
	off += int64(i) + 1

	if aug == "eh" {
		if p.f.Cpu&0x01000000 == 0 { // 32bit
			_, err := sect.ReadAt(p.scratch[:4], off)
			if err != nil {
				// TODO warning
				return
			}
			item.AppendRow([]*gui.QStandardItem{
				gui.NewQStandardItem2(fmt.Sprintf("%#016x", sect.Addr+uint64(off))),
				gui.NewQStandardItem2(fmt.Sprintf("% x", p.scratch[:4])),
				gui.NewQStandardItem2("EH Data"),
				nil, // TODO
			})
			off += 4
		} else {
			_, err := sect.ReadAt(p.scratch[:8], off)
			if err != nil {
				// TODO warning
				return
			}
			item.AppendRow([]*gui.QStandardItem{
				gui.NewQStandardItem2(fmt.Sprintf("%#016x", sect.Addr+uint64(off))),
				gui.NewQStandardItem2(fmt.Sprintf("% x", p.scratch[:8])),
				gui.NewQStandardItem2("EH Data"),
				nil, // TODO
			})
			off += 8
		}
	}

	var caf uint64
	n, err = p.uleb128(sect, off, &caf)
	if err != nil {
		// TODO warning
		return
	}
	item.AppendRow([]*gui.QStandardItem{
		gui.NewQStandardItem2(fmt.Sprintf("%#016x", sect.Addr+uint64(off))),
		gui.NewQStandardItem2(fmt.Sprintf("% x", p.scratch[:n])),
		gui.NewQStandardItem2("Code Alignment Factor"),
		gui.NewQStandardItem2(fmt.Sprintf("%d", caf)),
	})
	off += int64(n)

	var daf int64
	n, err = p.sleb128(sect, off, &daf)
	if err != nil {
		// TODO warning
		return
	}
	item.AppendRow([]*gui.QStandardItem{
		gui.NewQStandardItem2(fmt.Sprintf("%#016x", sect.Addr+uint64(off))),
		gui.NewQStandardItem2(fmt.Sprintf("% x", p.scratch[:n])),
		gui.NewQStandardItem2("Data Alignment Factor"),
		gui.NewQStandardItem2(fmt.Sprintf("%d", daf)),
	})
	off += int64(n)

	var rar uint64
	switch version {
	case 1:
		_, err := sect.ReadAt(p.scratch[:1], off)
		if err != nil {
			// TODO warning
			return
		}
		rar = uint64(p.scratch[0])
		item.AppendRow([]*gui.QStandardItem{
			gui.NewQStandardItem2(fmt.Sprintf("%#016x", sect.Addr+uint64(off))),
			gui.NewQStandardItem2(fmt.Sprintf("% x", p.scratch[:1])),
			gui.NewQStandardItem2("Return Address Register"),
			gui.NewQStandardItem2(p.f.registerString(rar)),
		})
		off++
	case 3:
		n, err := p.uleb128(sect, off, &rar)
		if err != nil {
			// TODO warning
			return
		}
		item.AppendRow([]*gui.QStandardItem{
			gui.NewQStandardItem2(fmt.Sprintf("%#016x", sect.Addr+uint64(off))),
			gui.NewQStandardItem2(fmt.Sprintf("% x", p.scratch[:n])),
			gui.NewQStandardItem2("Return Address Register"),
			gui.NewQStandardItem2(p.f.registerString(rar)),
		})
		off += int64(n)
	}

	penc := DW_EH_PE_omit
	pptr := uint64(0)
	fenc := uint8(DW_EH_PE_ptr) | uint8(DW_EH_PE_absptr)
	lenc := DW_EH_PE_omit

	if aug != "" && aug[0] == 'z' {
		var augdatalen uint64
		n, err := p.uleb128(sect, off, &augdatalen)
		if err != nil {
			// TODO warning
			return
		}
		item.AppendRow([]*gui.QStandardItem{
			gui.NewQStandardItem2(fmt.Sprintf("%#016x", sect.Addr+uint64(off))),
			gui.NewQStandardItem2(fmt.Sprintf("% x", p.scratch[:n])),
			gui.NewQStandardItem2("Augumentation Data Length"),
			gui.NewQStandardItem2(fmt.Sprintf("%d", augdatalen)),
		})
		off += int64(n)

		augend := off + int64(augdatalen)

		for _, r := range aug[1:] {
			switch r {
			case 'P': // Personlity encoding & pointer
				_, err := sect.ReadAt(p.scratch[:1], off)
				if err != nil {
					// TODO warning
					return
				}
				penc = p.scratch[0]
				item.AppendRow([]*gui.QStandardItem{
					gui.NewQStandardItem2(fmt.Sprintf("%#016x", sect.Addr+uint64(off))),
					gui.NewQStandardItem2(fmt.Sprintf("% x", p.scratch[:1])),
					gui.NewQStandardItem2("Augumentation Data (Personlity Encoding)"),
					gui.NewQStandardItem2(p.encodingString(penc, true)),
				})
				off++

				n, err := p.pointer(sect, off, penc, &pptr)
				if err != nil {
					// TODO warning
					return
				}
				item.AppendRow([]*gui.QStandardItem{
					gui.NewQStandardItem2(fmt.Sprintf("%#016x", sect.Addr+uint64(off))),
					gui.NewQStandardItem2(fmt.Sprintf("% x", p.scratch[:n])),
					gui.NewQStandardItem2("Augumentation Data (Personlity Pointer)"),
					gui.NewQStandardItem2(p.pointerString(pptr, sect.Addr+uint64(off), penc)),
				})
				off += int64(n)
			case 'R': // FDE encoding
				_, err := sect.ReadAt(p.scratch[:1], off)
				if err != nil {
					// TODO warning
					return
				}
				fenc = p.scratch[0]
				item.AppendRow([]*gui.QStandardItem{
					gui.NewQStandardItem2(fmt.Sprintf("%#016x", sect.Addr+uint64(off))),
					gui.NewQStandardItem2(fmt.Sprintf("% x", p.scratch[:1])),
					gui.NewQStandardItem2("Augumentation Data (FDE Encoding)"),
					gui.NewQStandardItem2(p.encodingString(fenc, true)),
				})
				off++
			case 'L': // LSDA encoding
				_, err := sect.ReadAt(p.scratch[:1], off)
				if err != nil {
					// TODO warning
					return
				}
				lenc = p.scratch[0]
				item.AppendRow([]*gui.QStandardItem{
					gui.NewQStandardItem2(fmt.Sprintf("%#016x", sect.Addr+uint64(off))),
					gui.NewQStandardItem2(fmt.Sprintf("% x", p.scratch[:1])),
					gui.NewQStandardItem2("Augumentation Data (LSDA Encoding)"),
					gui.NewQStandardItem2(p.encodingString(lenc, true)),
				})
				off++
			default:
				if off > augend {
					// TODO warning
					return
				}

				if off != augend {
					rest := make([]byte, augend-off)
					_, err := sect.ReadAt(rest, off)
					if err != nil {
						// TODO warning
						return
					}
					item.AppendRow([]*gui.QStandardItem{
						gui.NewQStandardItem2(fmt.Sprintf("%#016x", sect.Addr+uint64(off))),
						gui.NewQStandardItem2(fmt.Sprintf("% x", rest)),
						gui.NewQStandardItem2("Augumentation Data (Unknown)"),
						nil, // TODO
					})
				}

				off = augend
			}
		}

		if off > augend {
			// TODO warning
			return
		}

		if off != augend {
			rest := make([]byte, augend-off)
			_, err := sect.ReadAt(rest, off)
			if err != nil {
				// TODO warning
				return
			}
			item.AppendRow([]*gui.QStandardItem{
				gui.NewQStandardItem2(fmt.Sprintf("%#016x", sect.Addr+uint64(off))),
				gui.NewQStandardItem2(fmt.Sprintf("% x", rest)),
				gui.NewQStandardItem2("Augumentation Data (Remains)"),
				nil, // TODO
			})
		}

		off = augend
	}

	if off > end {
		// TODO warning
		return
	}

	insts := make([]byte, end-off)
	_, err = sect.ReadAt(insts, off)
	if err != nil {
		// TODO warning
		return
	}
	item.AppendRow([]*gui.QStandardItem{
		gui.NewQStandardItem2(fmt.Sprintf("%#016x", sect.Addr+uint64(off))),
		gui.NewQStandardItem2(fmt.Sprintf("% x", insts)),
		gui.NewQStandardItem2("Initial Instructions"),
		nil, // TODO
	})

	info := p.cieInfos[top]

	info.aug = aug
	info.penc = penc
	info.pptr = pptr
	info.lenc = lenc

	return true
}

func (p *parser) populateFDEItem(item *gui.QStandardItem, sect *macho.Section, off, end int64, info *cieInfo) (ok bool) {
	var pcBegin, pcRange uint64
	n, err := p.pointer(sect, off, info.fenc, &pcBegin)
	if err != nil {
		// TODO warning
		return false
	}
	item.AppendRow([]*gui.QStandardItem{
		gui.NewQStandardItem2(fmt.Sprintf("%#016x", sect.Addr+uint64(off))),
		gui.NewQStandardItem2(fmt.Sprintf("% x", p.scratch[:n])),
		gui.NewQStandardItem2("PC Begin"),
		gui.NewQStandardItem2(p.pointerString(pcBegin, sect.Addr+uint64(off), info.fenc)),
	})
	off += int64(n)
	n, err = p.pointer(sect, off, info.fenc, &pcRange)
	if err != nil {
		// TODO warning
		return false
	}
	item.AppendRow([]*gui.QStandardItem{
		gui.NewQStandardItem2(fmt.Sprintf("%#016x", sect.Addr+uint64(off))),
		gui.NewQStandardItem2(fmt.Sprintf("% x", p.scratch[:n])),
		gui.NewQStandardItem2("PC Range"),
		gui.NewQStandardItem2(fmt.Sprintf("%d", pcRange)),
	})
	off += int64(n)

	if info.aug != "" && info.aug[0] == 'z' {
		var augdatalen uint64
		n, err := p.uleb128(sect, off, &augdatalen)
		if err != nil {
			// TODO warning
			return
		}
		item.AppendRow([]*gui.QStandardItem{
			gui.NewQStandardItem2(fmt.Sprintf("%#016x", sect.Addr+uint64(off))),
			gui.NewQStandardItem2(fmt.Sprintf("% x", p.scratch[:n])),
			gui.NewQStandardItem2("Augumentation Data Length"),
			gui.NewQStandardItem2(fmt.Sprintf("%d", augdatalen)),
		})
		off += int64(n)

		augend := off + int64(augdatalen)

		for _, r := range info.aug[1:] {
			switch r {
			case 'P':
			case 'R':
			case 'L': // LSDA pointer
				var lptr uint64
				n, err := p.pointer(sect, off, info.lenc, &lptr)
				if err != nil {
					// TODO warning
					return false
				}
				item.AppendRow([]*gui.QStandardItem{
					gui.NewQStandardItem2(fmt.Sprintf("%#016x", sect.Addr+uint64(off))),
					gui.NewQStandardItem2(fmt.Sprintf("% x", p.scratch[:n])),
					gui.NewQStandardItem2("Augumentation Data (LSDA Pointer)"),
					gui.NewQStandardItem2(p.pointerString(lptr, sect.Addr+uint64(off), info.lenc)),
				})
				off += int64(n)
			default:
				if off > augend {
					// TODO warning
					return false
				}

				if off != augend {
					rest := make([]byte, augend-off)
					_, err := sect.ReadAt(rest, off)
					if err != nil {
						// TODO warning
						return
					}
					item.AppendRow([]*gui.QStandardItem{
						gui.NewQStandardItem2(fmt.Sprintf("%#016x", sect.Addr+uint64(off))),
						gui.NewQStandardItem2(fmt.Sprintf("% x", rest)),
						gui.NewQStandardItem2("Augumentation Data (Unknown)"),
						nil, // TODO
					})
				}

				off = augend
			}
		}

		if off > augend {
			// TODO warning
			return false
		}

		if off != augend {
			rest := make([]byte, augend-off)
			_, err := sect.ReadAt(rest, off)
			if err != nil {
				// TODO warning
				return false
			}
			item.AppendRow([]*gui.QStandardItem{
				gui.NewQStandardItem2(fmt.Sprintf("%#016x", sect.Addr+uint64(off))),
				gui.NewQStandardItem2(fmt.Sprintf("% x", rest)),
				gui.NewQStandardItem2("Augumentation Data (Remains)"),
				nil, // TODO
			})
		}

		off = augend
	}

	if off > end {
		// TODO warning
		return false
	}

	insts := make([]byte, end-off)
	_, err = sect.ReadAt(insts, off)
	if err != nil {
		// TODO warning
		return false
	}
	item.AppendRow([]*gui.QStandardItem{
		gui.NewQStandardItem2(fmt.Sprintf("%#016x", sect.Addr+uint64(off))),
		gui.NewQStandardItem2(fmt.Sprintf("% x", insts)),
		gui.NewQStandardItem2("Initial Instructions"),
		nil, // TODO
	})

	return true
}

func (p *parser) encodingString(enc uint8, html bool) string {
	if enc == DW_EH_PE_omit {
		return "0xff (DW_EH_PE_omit)"
	}
	var values []string
	values = append(values, fmt.Sprintf("%#02x (%s)", enc&DW_EH_PE_basic, DW_EH_PE_basicType(enc&DW_EH_PE_basic)))
	values = append(values, fmt.Sprintf("%#02x (%s)", enc&DW_EH_PE_modifier, DW_EH_PE_modType(enc&DW_EH_PE_modifier)))
	if enc&DW_EH_PE_indirect != 0 {
		values = append(values, "0x80 (DW_EH_PE_indirect)")
	}
	s := strings.Join(values, "\n")
	if html {
		s = "<body>" + s + "</body>"
	}
	return s
}

func (p *parser) pointerString(val uint64, addr uint64, enc uint8) string {
	if enc&DW_EH_PE_indirect != 0 {
		// TODO
		return fmt.Sprintf("%#x", val)
	}

	switch DW_EH_PE_modType(enc & DW_EH_PE_modifier &^ DW_EH_PE_indirect) {
	case DW_EH_PE_absptr:
		return p.f.symAddrString(val, true)
	case DW_EH_PE_pcrel:
		return fmt.Sprintf("%#x(%%rip) = %s", val, p.f.symAddrString(addr+val, true))
	case DW_EH_PE_textrel:
		// return fmt.Sprintf("__text:%#x = %s", val, p.f.symAddrString(addr+val, true))
	case DW_EH_PE_datarel:
		// return fmt.Sprintf("__data:%#x = %s", val, p.f.symAddrString(addr+val, true))
	case DW_EH_PE_funcrel:
		// return fmt.Sprintf("__data:%#x = %s", val, p.f.symAddrString(addr+val, true))
	case DW_EH_PE_aligned:
	}

	// TODO handle more modifiers

	return fmt.Sprintf("%#x", val)
}

func (p *parser) pointer(r io.ReaderAt, off int64, enc uint8, val *uint64) (int, error) {
	if enc == DW_EH_PE_omit {
		return 0, nil
	}

	bo := p.f.ByteOrder

	switch DW_EH_PE_basicType(enc & DW_EH_PE_basic) {
	case DW_EH_PE_ptr, DW_EH_PE_signed:
		if p.f.Cpu&0x01000000 == 0 { // 32bit
			_, err := r.ReadAt(p.scratch[:4], off)
			if err != nil {
				return 0, err
			}
			*val = uint64(bo.Uint32(p.scratch[:4]))
			return 4, nil
		} else {
			_, err := r.ReadAt(p.scratch[:8], off)
			if err != nil {
				return 0, err
			}
			*val = bo.Uint64(p.scratch[:8])
			return 8, nil
		}
	case DW_EH_PE_uleb128:
		return p.uleb128(r, off, val)
	case DW_EH_PE_sleb128:
		var i64 int64
		n, err := p.sleb128(r, off, &i64)
		if err != nil {
			return 0, err
		}
		*val = uint64(i64)
		return n, nil
	case DW_EH_PE_udata2, DW_EH_PE_sdata2:
		_, err := r.ReadAt(p.scratch[:2], off)
		if err != nil {
			return 0, err
		}
		*val = uint64(bo.Uint16(p.scratch[:2]))
		return 2, nil
	case DW_EH_PE_udata4, DW_EH_PE_sdata4:
		_, err := r.ReadAt(p.scratch[:4], off)
		if err != nil {
			return 0, err
		}
		*val = uint64(bo.Uint32(p.scratch[:4]))
		return 4, nil
	case DW_EH_PE_udata8, DW_EH_PE_sdata8:
		_, err := r.ReadAt(p.scratch[:8], off)
		if err != nil {
			return 0, err
		}
		*val = bo.Uint64(p.scratch[:8])
		return 8, nil
	}

	return 0, nil
}

func (p *parser) uleb128(r io.ReaderAt, off int64, val *uint64) (int, error) {
	v := uint64(0)
	s := uint8(0)
	i := 0

	for {
		n, _ := r.ReadAt(p.scratch[:], off)
		if n == 0 {
			return 0, errors.New("truncated uleb128")
		}

		p := p.scratch[:n]

		for _, c := range p {
			i++
			if i > 8 {
				return 0, errors.New("uleb128 too large")
			}
			v |= uint64(c&0x7f) << s
			if c&0x80 == 0 {
				*val = v

				return i, nil
			}
			s += 7
		}

		off += int64(n)
	}
}

func (p *parser) sleb128(r io.ReaderAt, off int64, val *int64) (int, error) {
	v := uint64(0)
	s := uint8(0)
	i := 0

	for {
		n, _ := r.ReadAt(p.scratch[:], off)
		if n == 0 {
			return 0, errors.New("truncated sleb128")
		}

		for _, c := range p.scratch[:n] {
			i++
			if i > 8 {
				return 0, errors.New("sleb128 too large")
			}
			v |= uint64(c&0x7f) << s
			s += 7
			if c&0x80 == 0 {
				if c&0x40 != 0 {
					v |= -(1 << s)
				}
				*val = int64(v)

				return i, nil
			}
		}

		off += int64(n)
	}
}
