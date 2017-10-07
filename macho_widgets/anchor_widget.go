package macho_widgets

import (
	"debug/macho"
	"fmt"
	"net/url"
	"strconv"

	"github.com/therecipe/qt/widgets"
)

func (f *File) NewAnchorWidget(anchor string) widgets.QWidget_ITF {
	u, err := url.Parse(anchor)
	if err != nil {
		panic(err)
	}

	var symnum int
	if _, err := fmt.Sscanf(u.Path, "/symbol/%d", &symnum); err == nil {
		sym := &f.Syms[symnum]

		q, err := url.ParseQuery(u.RawQuery)
		if err != nil {
			panic(err)
		}
		addend, err := strconv.ParseInt(q.Get("addend"), 10, 8)
		if err != nil {
			panic(err)
		}
		size, err := strconv.ParseInt(q.Get("size"), 10, 8)
		if err != nil {
			panic(err)
		}

		head := []string{"Name", "Type", "Sect", "Desc", "Value"}
		row := []string{sym.Name, f.symTypeString(sym.Type), f.symSectionString(sym.Sect), f.symDescString(sym), f.symValueString(sym)}

		h := widgets.NewQTableWidget(nil)
		h.VerticalHeader().SetVisible(false)
		h.VerticalHeader().SetSectionResizeMode(widgets.QHeaderView__ResizeToContents)
		h.HorizontalHeader().SetStretchLastSection(true)
		h.HorizontalHeader().SetSectionResizeMode(widgets.QHeaderView__ResizeToContents)
		h.SetShowGrid(false)
		h.SetRowCount(1)
		h.SetColumnCount(len(row))
		h.SetHorizontalHeaderLabels(head)
		h.SetAlternatingRowColors(true)
		h.SetSelectionBehavior(widgets.QAbstractItemView__SelectRows)
		h.SetEditTriggers(widgets.QAbstractItemView__NoEditTriggers)
		for column := 0; column < h.ColumnCount(); column++ {
			h.SetItem(0, column, widgets.NewQTableWidgetItem2(row[column], 0))
		}

		hh := h.RowHeight(0) + h.HorizontalHeader().Height() + 2*h.FrameWidth()
		h.SetMaximumHeight(hh)
		h.SetSizePolicy2(widgets.QSizePolicy__Expanding, widgets.QSizePolicy__Fixed)

		v := f.NewSymdataWidget(nil)
		v.SetSymbol(sym, addend, size)

		vlayout := widgets.NewQVBoxLayout()
		vlayout.AddWidget(h, 0, 0)
		vlayout.AddWidget(v, 0, 0)
		vlayout.SetContentsMargins(0, 0, 0, 0)

		w := widgets.NewQWidget(nil, 0)
		w.SetLayout(vlayout)

		return w
	}

	var addr uint64
	if _, err := fmt.Sscanf(u.Path, "/address/%d", &addr); err == nil {
		var sect *macho.Section

		for _, s := range f.Sections {
			if s.Addr <= addr && addr < s.Addr+s.Size {
				sect = s
				break
			}
		}

		if sect == nil {
			// TODO warning
			return nil
		}

		q, err := url.ParseQuery(u.RawQuery)
		if err != nil {
			panic(err)
		}
		size, err := strconv.ParseInt(q.Get("size"), 10, 8)
		if err != nil {
			panic(err)
		}

		head := []string{"Sectname", "Segname", "Addr", "Size", "Offset", "Align", "Reloff", "Nreloc", "Flags"}
		row := []string{sect.Name, sect.Seg, fmt.Sprintf("%#016x", sect.Addr), fmt.Sprint(sect.Size), fmt.Sprint(sect.Offset), fmt.Sprintf("%d (%d)", sect.Align, 1<<sect.Align), fmt.Sprint(sect.Reloff), fmt.Sprint(sect.Nreloc), f.sectionFlagsString(sect.Flags, false)}

		h := widgets.NewQTableWidget(nil)
		h.VerticalHeader().SetVisible(false)
		h.VerticalHeader().SetSectionResizeMode(widgets.QHeaderView__ResizeToContents)
		h.HorizontalHeader().SetStretchLastSection(true)
		h.HorizontalHeader().SetSectionResizeMode(widgets.QHeaderView__ResizeToContents)
		h.SetShowGrid(false)
		h.SetRowCount(1)
		h.SetColumnCount(len(row))
		h.SetHorizontalHeaderLabels(head)
		h.SetAlternatingRowColors(true)
		h.SetSelectionBehavior(widgets.QAbstractItemView__SelectRows)
		h.SetEditTriggers(widgets.QAbstractItemView__NoEditTriggers)
		for column := 0; column < h.ColumnCount(); column++ {
			h.SetItem(0, column, widgets.NewQTableWidgetItem2(row[column], 0))
		}

		hh := h.RowHeight(0) + h.HorizontalHeader().Height() + 2*h.FrameWidth()
		h.SetMaximumHeight(hh)
		h.SetSizePolicy2(widgets.QSizePolicy__Expanding, widgets.QSizePolicy__Fixed)

		v := f.NewSectdataWidget(nil)
		v.SetSection(sect, addr, size)

		vlayout := widgets.NewQVBoxLayout()
		vlayout.AddWidget(h, 0, 0)
		vlayout.AddWidget(v, 0, 0)
		vlayout.SetContentsMargins(0, 0, 0, 0)

		w := widgets.NewQWidget(nil, 0)
		w.SetLayout(vlayout)

		return w
	}

	panic(fmt.Errorf("unhandled anchor %s", anchor))

	return nil
}
