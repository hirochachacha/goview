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
func NewSymtabWidget(f *macho.File) widgets.QWidget_ITF {
	symtabModel := NewSymtabModel(f)

	search := widgets.NewQLineEdit(nil)
	search.SetPlaceholderText("Search ...")

	symtab := widgets.NewQTableView(nil)
	symtab.SetModel(symtabModel.Symtab)
	symtab.VerticalHeader().SetDefaultSectionSize(30)
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

	asmtab := widgets.NewQTableView(nil)
	asmtab.VerticalHeader().SetVisible(false)
	asmtab.VerticalHeader().SetDefaultSectionSize(20)
	asmtab.HorizontalHeader().SetStretchLastSection(true)
	asmtab.HorizontalHeader().SetDefaultAlignment(core.Qt__AlignLeft)
	asmtab.HorizontalHeader().SetSectionResizeMode(widgets.QHeaderView__ResizeToContents)
	asmtab.SetShowGrid(false)
	asmtab.SetSelectionBehavior(widgets.QAbstractItemView__SelectRows)
	asmtab.SetEditTriggers(widgets.QAbstractItemView__NoEditTriggers)

	symtab.ConnectCurrentChanged(func(current *core.QModelIndex, previous *core.QModelIndex) {
		asmtab.SetModel(symtabModel.Asmtab(current))
	})

	symtabGroup := widgets.NewQWidget(nil, 0)
	{
		vlayout := widgets.NewQVBoxLayout()
		vlayout.AddWidget(search, 0, 0)
		vlayout.AddWidget(symtab, 0, 0)
		vlayout.SetContentsMargins(0, 0, 0, 0)
		symtabGroup.SetLayout(vlayout)
	}

	var ra widgets.QWidget_ITF
	if f.Type == macho.TypeObj {
		tab := widgets.NewQTabWidget(nil)
		if f.Type == macho.TypeObj {
			tab.AddTab(reltab, "Relocations")
		}
		tab.AddTab(asmtab, "Assembly")
		ra = tab
	} else {
		ra = asmtab
	}

	sp := widgets.NewQSplitter2(core.Qt__Vertical, nil)
	sp.AddWidget(symtabGroup)
	sp.AddWidget(ra)

	w := widgets.NewQWidget(nil, 0)
	layout := widgets.NewQVBoxLayout()
	layout.AddWidget(sp, 0, 0)
	w.SetLayout(layout)

	return w
}
