package macho_widgets

import (
	"debug/macho"

	"github.com/therecipe/qt/core"
	"github.com/therecipe/qt/widgets"
)

// _____________
// |___|___|___|
// |___|___|___|
// |___|___|___|
// |___|___|___|
// |     |     |
// |     |     |
func NewSymtabWidget(f *macho.File) (widgets.QWidget_ITF, error) {
	symtabModel, err := NewSymtabModel(f)
	if err != nil {
		return nil, err
	}

	w := widgets.NewQWidget(nil, 0)

	// TODO add search box (top right)

	symtab := widgets.NewQTableView(nil)
	symtab.SetModel(symtabModel.Symtab)
	symtab.VerticalHeader().SetDefaultSectionSize(20)
	symtab.HorizontalHeader().SetStretchLastSection(true)
	symtab.HorizontalHeader().SetDefaultAlignment(core.Qt__AlignLeft)
	symtab.HorizontalHeader().SetSectionResizeMode(widgets.QHeaderView__ResizeToContents)
	symtab.SetSelectionBehavior(widgets.QAbstractItemView__SelectRows)
	symtab.SetEditTriggers(widgets.QAbstractItemView__NoEditTriggers)
	symtab.SetShowGrid(false)
	symtab.SetAlternatingRowColors(true)

	// TODO add relocations widget (bottom left)
	// TODO add assembly widget (bottom right)

	layout := widgets.NewQVBoxLayout()
	layout.AddWidget(symtab, 0, 0)
	w.SetLayout(layout)

	return w, nil
}
