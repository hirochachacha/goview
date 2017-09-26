package macho_widgets

import (
	"debug/macho"

	"github.com/therecipe/qt/core"
	"github.com/therecipe/qt/widgets"
)

// ___________________________
// File              |___|___|
//   Header          |___|___|
//   Loads           |___|___|
//     LC_SEGMENT    |___|___|
//     LC_SEGMENT_64 |   |   |
func NewStructWidget(f *macho.File) widgets.QWidget_ITF {
	treeModel := NewStructModel(f)

	tree := widgets.NewQTreeView(nil)
	tree.SetHeaderHidden(true)
	tree.SetModel(treeModel.Tree)
	tree.SetEditTriggers(widgets.QAbstractItemView__NoEditTriggers)
	tree.ExpandAll()

	attrtab := widgets.NewQTableView(nil)
	attrtab.VerticalHeader().SetVisible(false)
	attrtab.VerticalHeader().SetDefaultSectionSize(20)
	attrtab.VerticalHeader().SetSectionResizeMode(widgets.QHeaderView__ResizeToContents)
	attrtab.HorizontalHeader().SetStretchLastSection(true)
	attrtab.HorizontalHeader().SetDefaultAlignment(core.Qt__AlignLeft)
	attrtab.HorizontalHeader().SetSectionResizeMode(widgets.QHeaderView__ResizeToContents)
	attrtab.SetShowGrid(false)
	attrtab.SetAlternatingRowColors(true)
	attrtab.SetSelectionBehavior(widgets.QAbstractItemView__SelectRows)
	attrtab.SetEditTriggers(widgets.QAbstractItemView__NoEditTriggers)

	tree.ConnectCurrentChanged(func(current *core.QModelIndex, previous *core.QModelIndex) {
		attrtab.SetModel(treeModel.AttrTab(current))
	})

	sp := widgets.NewQSplitter(nil)
	sp.AddWidget(tree)
	sp.AddWidget(attrtab)
	sp.SetStretchFactor(0, 2)
	sp.SetStretchFactor(1, 3)

	layout := widgets.NewQVBoxLayout()
	layout.AddWidget(sp, 0, 0)

	w := widgets.NewQWidget(nil, 0)
	w.SetLayout(layout)

	return w
}
