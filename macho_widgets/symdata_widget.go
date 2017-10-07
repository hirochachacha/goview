package macho_widgets

import (
	"debug/macho"

	"github.com/therecipe/qt/widgets"
)

type SymdataWidget struct {
	*widgets.QWidget

	bb      *ButtonBarWidget
	tree    *widgets.QTreeView
	f       *File
	sym     *macho.Symbol
	taddend int64
	tsize   int64
}

func (f *File) NewSymdataWidget(parent widgets.QWidget_ITF) *SymdataWidget {
	w := new(SymdataWidget)

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

func (w *SymdataWidget) SetSymbol(sym *macho.Symbol, taddend, tsize int64) {
	w.sym = sym
	w.taddend = taddend
	w.tsize = tsize

	typ := w.f.guessSymType(w.sym)

	w.bb.SetChecked(typ, true)

	w.SetModel(typ)
}

func (w *SymdataWidget) SetModel(typ string) {
	if w.sym == nil {
		return
	}

	if w.sym.Type&N_TYPE != N_SECT {
		return
	}

	w.tree.SetModel(w.f.NewSymbolModel(typ, w.sym, w.taddend, w.tsize))
}
