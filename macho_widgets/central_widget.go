package macho_widgets

import (
	"debug/macho"

	"github.com/therecipe/qt/widgets"
)

func NewCentralWidget(parent widgets.QWidget_ITF, mf *macho.File) widgets.QWidget_ITF {
	f := NewFile(mf)

	tab := widgets.NewQTabWidget(parent)
	tab.AddTab(f.NewStructWidget(nil), "Structure")
	tab.AddTab(f.NewSymtabWidget(nil), "Symbols")
	if f.Type == macho.TypeObj {
		tab.AddTab(f.NewReltabWidget(nil), "Relocations")
	}
	return tab
}
