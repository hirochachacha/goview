package macho_widgets

import (
	"debug/macho"

	"github.com/therecipe/qt/core"
	"github.com/therecipe/qt/widgets"
)

func NewReltabWidget(f *macho.File, lookup symLookup) widgets.QWidget_ITF {
	reltabModel := NewReltabModel(f, lookup)

	seclist := widgets.NewQListView(nil)
	seclist.SetModel(reltabModel.Sections)
	seclist.SetEditTriggers(widgets.QAbstractItemView__NoEditTriggers)

	// TODO add toggle button that can switch relocated (address offset/address)

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

	seclist.ConnectCurrentChanged(func(current *core.QModelIndex, previous *core.QModelIndex) {
		reltab.SetModel(reltabModel.Reltab(current))
	})

	sp := widgets.NewQSplitter(nil)
	sp.AddWidget(seclist)
	sp.AddWidget(reltab)
	sp.SetStretchFactor(1, 2)

	w := widgets.NewQWidget(nil, 0)
	layout := widgets.NewQVBoxLayout()
	layout.AddWidget(sp, 0, 0)
	w.SetLayout(layout)

	return w
}
