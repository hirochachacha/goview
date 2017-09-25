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
// |           |
// |           |
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

	reltab := widgets.NewQTableView(nil)
	reltab.VerticalHeader().SetVisible(false)
	reltab.VerticalHeader().SetDefaultSectionSize(20)
	reltab.HorizontalHeader().SetStretchLastSection(true)
	reltab.HorizontalHeader().SetDefaultAlignment(core.Qt__AlignLeft)
	reltab.HorizontalHeader().SetSectionResizeMode(widgets.QHeaderView__ResizeToContents)
	reltab.SetSelectionBehavior(widgets.QAbstractItemView__SelectRows)
	reltab.SetEditTriggers(widgets.QAbstractItemView__NoEditTriggers)
	reltab.SetShowGrid(false)
	reltab.SetAlternatingRowColors(true)

	symtab.ConnectCurrentChanged(func(current *core.QModelIndex, previous *core.QModelIndex) {
		reltab.SetModel(symtabModel.Reltab(current))
	})

	// TODO add assembly widget (bottom right)

	layout := widgets.NewQGridLayout2()
	layout.AddWidget3(symtab, 0, 0, 2, 1, 0)
	layout.AddWidget(reltab, 2, 0, 0)
	// echoLayout.AddWidget(asm, 1, 1, 0)
	w.SetLayout(layout)

	return w, nil
}
