package macho_widgets

import (
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
func (f *File) NewSymtabWidget(parent widgets.QWidget_ITF) widgets.QWidget_ITF {
	symtabModel := f.NewSymtabModel()

	symChars := []string{"*", "U", "T", "D", "B", "C", "S", "A", "I", "-"}
	symChar := f.NewButtonBarWidget(nil, symChars)
	symChar.Toggle("*")

	externOnly := widgets.NewQCheckBox2("Extern Only", nil)

	searchName := widgets.NewQLineEdit(nil)
	searchName.SetPlaceholderText("Search...")

	symtab := widgets.NewQTableView(nil)
	symtab.SetModel(symtabModel.Symtab)
	symtab.VerticalHeader().SetDefaultSectionSize(30)
	symtab.HorizontalHeader().SetDefaultAlignment(core.Qt__AlignLeft)
	symtab.HorizontalHeader().SetSectionResizeMode(widgets.QHeaderView__ResizeToContents)
	symtab.SetShowGrid(false)
	symtab.SetAlternatingRowColors(true)
	symtab.SetSelectionBehavior(widgets.QAbstractItemView__SelectRows)
	symtab.SetEditTriggers(widgets.QAbstractItemView__NoEditTriggers)

	symChar.ConnectButtonToggled2(func(label string, checked bool) {
		if checked {
			symtabModel.SetFilterChar(label[0])
		}
	})

	externOnly.ConnectClicked(func(checked bool) {
		symtabModel.SetFilterExternOnly(checked)
	})

	searchName.ConnectEditingFinished(func() {
		symtabModel.SetFilterName(searchName.Text())
	})

	symdata := f.NewSymdataWidget(nil)

	symtab.ConnectCurrentChanged(func(current *core.QModelIndex, previous *core.QModelIndex) {
		current = symtabModel.Symtab.(*core.QSortFilterProxyModel).MapToSource(current)
		if current.IsValid() {
			row := current.Row()
			if 0 <= row && row < len(f.Syms) {
				symdata.SetSymbol(&f.Syms[row], 0, 0)
				return
			}
		}
		symdata.SetModel("")
	})

	symtabGroup := widgets.NewQWidget(nil, 0)
	{
		hlayout := widgets.NewQHBoxLayout()
		hlayout.AddWidget(symChar, 0, 0)
		hlayout.AddWidget(externOnly, 0, 0)
		hlayout.AddWidget(searchName, 0, 0)
		hlayout.SetContentsMargins(0, 0, 0, 0)

		vlayout := widgets.NewQVBoxLayout()
		vlayout.AddLayout(hlayout, 0)
		vlayout.AddWidget(symtab, 0, 0)
		vlayout.SetContentsMargins(0, 0, 0, 0)

		symtabGroup.SetLayout(vlayout)
	}

	sp := widgets.NewQSplitter2(core.Qt__Vertical, nil)
	sp.AddWidget(symtabGroup)
	sp.AddWidget(symdata)

	w := widgets.NewQWidget(parent, 0)
	layout := widgets.NewQVBoxLayout()
	layout.AddWidget(sp, 0, 0)
	w.SetLayout(layout)

	searchName.SetFocus2()

	return w
}
