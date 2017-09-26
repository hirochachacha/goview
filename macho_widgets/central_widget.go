package macho_widgets

import (
	"debug/macho"

	"github.com/therecipe/qt/widgets"
)

func NewCentralWidget(f *macho.File) widgets.QWidget_ITF {
	tab := widgets.NewQTabWidget(nil)
	tab.AddTab(NewStructWidget(f), "Structure")
	tab.AddTab(NewSymtabWidget(f), "Symbols")
	if f.Type == macho.TypeObj {
		tab.AddTab(NewReltabWidget(f), "Relocations")
	}
	return tab
}
