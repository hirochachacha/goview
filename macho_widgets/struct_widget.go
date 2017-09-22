package macho_widgets

import (
	"debug/macho"

	"github.com/therecipe/qt/core"
	"github.com/therecipe/qt/widgets"
)

const StructItemRole = int(core.Qt__UserRole) + 1

// _______________________________
// File              |___|___|___|
//   Header          |___|___|___|
//   Loads           |___|___|___|
//     LC_SEGMENT    |___|___|___|
//     LC_SEGMENT_64 |   |   |   |
func NewStructWidget(f *macho.File) (widgets.QWidget_ITF, error) {
	treeModel, err := NewStructModel(f)
	if err != nil {
		return nil, err
	}

	w := widgets.NewQWidget(nil, 0)

	tree := widgets.NewQTreeView(nil)
	tree.SetHeaderHidden(true)
	tree.SetModel(treeModel.Tree)
	tree.ExpandAll()

	table := widgets.NewQTableView(nil)
	table.VerticalHeader().SetVisible(false)
	table.VerticalHeader().SetDefaultSectionSize(20)
	table.HorizontalHeader().SetStretchLastSection(true)
	table.HorizontalHeader().SetDefaultAlignment(core.Qt__AlignLeft)
	table.HorizontalHeader().SetSectionResizeMode(widgets.QHeaderView__ResizeToContents)
	table.SetSelectionBehavior(widgets.QAbstractItemView__SelectRows)
	table.SetEditTriggers(widgets.QAbstractItemView__NoEditTriggers)
	table.SetShowGrid(false)
	table.SetAlternatingRowColors(true)

	tree.ConnectCurrentChanged(func(current *core.QModelIndex, previous *core.QModelIndex) {
		table.SetModel(nil)
		if val := current.Data(StructItemRole); val.IsValid() {
			if i := val.ToInt(false); i > 0 {
				table.SetModel(treeModel.Tables[i-1])
			}
		}
	})

	sp := widgets.NewQSplitter(nil)
	sp.AddWidget(tree)
	sp.AddWidget(table)

	layout := widgets.NewQVBoxLayout()
	layout.AddWidget(sp, 0, 0)
	w.SetLayout(layout)

	return w, nil
}
