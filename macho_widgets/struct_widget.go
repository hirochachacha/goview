package macho_widgets

import (
	"github.com/therecipe/qt/core"
	"github.com/therecipe/qt/widgets"
)

// ___________________________
// File              |___|___|
//   Header          |___|___|
//   Loads           |___|___|
//     LC_SEGMENT    |___|___|
//     LC_SEGMENT_64 |   |   |
func (f *File) NewStructWidget(parent widgets.QWidget_ITF) widgets.QWidget_ITF {
	strctModel := f.NewStructModel()

	strct := widgets.NewQTreeView(nil)
	strct.SetHeaderHidden(true)
	strct.SetModel(strctModel.Tree)
	strct.SetEditTriggers(widgets.QAbstractItemView__NoEditTriggers)
	strct.ExpandAll()

	attr := f.NewDataView(nil)
	attr.SetAlternatingRowColors(true)

	strct.ConnectCurrentChanged(func(current *core.QModelIndex, previous *core.QModelIndex) {
		attr.SetModel(strctModel.AttrTab(current))
	})

	sp := widgets.NewQSplitter(nil)
	sp.AddWidget(strct)
	sp.AddWidget(attr)
	sp.SetStretchFactor(0, 2)
	sp.SetStretchFactor(1, 3)

	layout := widgets.NewQVBoxLayout()
	layout.AddWidget(sp, 0, 0)

	w := widgets.NewQWidget(parent, 0)
	w.SetLayout(layout)

	return w
}
