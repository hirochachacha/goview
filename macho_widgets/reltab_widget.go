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

	addrCheck := widgets.NewQCheckBox2("Calculate Address", nil)

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
		if addrCheck.IsChecked() {
			reltab.SetColumnHidden(0, false)
			reltab.SetColumnHidden(1, true)
		} else {
			reltab.SetColumnHidden(0, true)
			reltab.SetColumnHidden(1, false)
		}
	})

	addrCheck.ConnectClicked(func(checked bool) {
		if checked {
			reltab.SetColumnHidden(0, false)
			reltab.SetColumnHidden(1, true)
		} else {
			reltab.SetColumnHidden(0, true)
			reltab.SetColumnHidden(1, false)
		}
	})

	reltabGroup := widgets.NewQWidget(nil, 0)
	{
		vlayout := widgets.NewQVBoxLayout()
		vlayout.AddWidget(addrCheck, 0, 0)
		vlayout.AddWidget(reltab, 0, 0)
		vlayout.SetContentsMargins(0, 0, 0, 0)

		reltabGroup.SetLayout(vlayout)
	}

	sp := widgets.NewQSplitter(nil)
	sp.AddWidget(seclist)
	sp.AddWidget(reltabGroup)
	sp.SetStretchFactor(1, 2)

	w := widgets.NewQWidget(nil, 0)
	layout := widgets.NewQVBoxLayout()
	layout.AddWidget(sp, 0, 0)
	w.SetLayout(layout)

	return w
}
