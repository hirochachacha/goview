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

	search := widgets.NewQLineEdit(nil)
	search.SetPlaceholderText("Search ...")

	symtab := widgets.NewQTableView(nil)
	symtab.SetModel(symtabModel.Symtab)
	symtab.VerticalHeader().SetDefaultSectionSize(30)
	symtab.HorizontalHeader().SetStretchLastSection(true)
	symtab.HorizontalHeader().SetDefaultAlignment(core.Qt__AlignLeft)
	symtab.HorizontalHeader().SetSectionResizeMode(widgets.QHeaderView__ResizeToContents)
	symtab.SetShowGrid(false)
	symtab.SetAlternatingRowColors(true)
	symtab.SetSelectionBehavior(widgets.QAbstractItemView__SelectRows)
	symtab.SetEditTriggers(widgets.QAbstractItemView__NoEditTriggers)

	search.ConnectEditingFinished(func() {
		symtabModel.SetFilter(search.Text())
	})

	reltab := widgets.NewQTableView(nil)
	reltab.VerticalHeader().SetVisible(false)
	reltab.VerticalHeader().SetDefaultSectionSize(20)
	reltab.HorizontalHeader().SetStretchLastSection(true)
	reltab.HorizontalHeader().SetDefaultAlignment(core.Qt__AlignLeft)
	reltab.HorizontalHeader().SetSectionResizeMode(widgets.QHeaderView__ResizeToContents)
	reltab.SetShowGrid(false)
	reltab.SetAlternatingRowColors(true)
	reltab.SetSelectionBehavior(widgets.QAbstractItemView__SelectRows)
	reltab.SetEditTriggers(widgets.QAbstractItemView__NoEditTriggers)
	reltab.SetSortingEnabled(true)

	symtab.ConnectCurrentChanged(func(current *core.QModelIndex, previous *core.QModelIndex) {
		reltab.SetModel(symtabModel.Reltab(current))
	})

	// TODO add assembly widget (bottom)

	symtabGroup := widgets.NewQGroupBox2("Symbols", nil)
	{
		vlayout := widgets.NewQVBoxLayout()
		vlayout.AddWidget(search, 0, 0)
		vlayout.AddWidget(symtab, 0, 0)
		symtabGroup.SetLayout(vlayout)
	}

	reltabGroup := widgets.NewQGroupBox2("Relocations", nil)
	{
		vlayout := widgets.NewQVBoxLayout()
		vlayout.AddWidget(reltab, 0, 0)
		reltabGroup.SetLayout(vlayout)
	}

	sp := widgets.NewQSplitter2(core.Qt__Vertical, nil)
	sp.AddWidget(symtabGroup)
	sp.AddWidget(reltabGroup)

	layout := widgets.NewQVBoxLayout()
	layout.AddWidget(sp, 0, 0)
	w.SetLayout(layout)

	return w, nil
}
