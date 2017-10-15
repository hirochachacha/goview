package macho_widgets

import (
	"debug/macho"

	"github.com/therecipe/qt/widgets"
)

type SectdataWidget struct {
	*widgets.QWidget

	bb    *ButtonBarWidget
	tree  *DataView
	f     *File
	sect  *macho.Section
	taddr uint64
	tsize int64
}

func (f *File) NewSectdataWidget(parent widgets.QWidget_ITF) *SectdataWidget {
	w := new(SectdataWidget)

	labels := []string{"Code", "CString", "Float32", "Float64", "Float128", "Pointer32", "Data"}

	w.bb = f.NewButtonBarWidget(nil, labels)
	w.tree = f.NewDataView(nil)

	w.tree.Header().SetStretchLastSection(true)
	w.tree.Header().SetSectionResizeMode(widgets.QHeaderView__ResizeToContents)

	w.bb.SetSizePolicy2(widgets.QSizePolicy__Fixed, widgets.QSizePolicy__Fixed)

	w.bb.ConnectButtonToggled2(func(label string, checked bool) {
		if checked {
			w.SetModel(label)
		}
	})

	w.f = f

	vlayout := widgets.NewQVBoxLayout()
	vlayout.AddWidget(w.bb, 0, 0)
	vlayout.AddWidget(w.tree, 0, 0)
	vlayout.SetContentsMargins(0, 0, 0, 0)

	w.QWidget = widgets.NewQWidget(parent, 0)
	w.QWidget.SetLayout(vlayout)

	return w
}

func (w *SectdataWidget) SetSection(sect *macho.Section, taddr uint64, tsize int64) {
	w.sect = sect
	w.taddr = taddr
	w.tsize = tsize

	typ := w.f.guessSectType(w.sect)

	w.bb.SetChecked(typ, true)

	w.SetModel(typ)
}

func (w *SectdataWidget) SetModel(typ string) {
	if w.sect == nil {
		return
	}

	w.tree.SetModel(w.f.NewSectionModel(typ, w.sect, w.taddr, w.tsize))
}
