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
func NewSymtabWidget(f *macho.File, ssyms []*macho.Symbol, symAddrInfo map[uint64]*symInfo, lookup symLookup) widgets.QWidget_ITF {
	symtabModel := NewSymtabModel(f, ssyms, symAddrInfo, lookup)

	search := widgets.NewQLineEdit(nil)
	search.SetPlaceholderText("Search...")

	searchType := widgets.NewQComboBox(nil)
	searchType.AddItems([]string{"(*) Any", "(U)undefined", "(A)bsolute", "(T)ext", "(D)ata", "(B)ss", "(C)ommon", "(-) debug", "(S) other", "(I)ndirect"})

	externCheck := widgets.NewQCheckBox2("Extern Only", nil)

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
		symtabModel.SetFilterName(search.Text())
	})

	searchType.ConnectCurrentIndexChanged2(func(text string) {
		if 3 <= len(text) && text[0] == '(' && text[2] == ')' {
			symtabModel.SetFilterType(text[1])
		}
	})

	externCheck.ConnectClicked(func(checked bool) {
		symtabModel.SetFilterExternOnly(checked)
	})

	asmtree := widgets.NewQTreeView(nil)
	asmtree.Header().SetStretchLastSection(true)
	asmtree.Header().SetSectionResizeMode(widgets.QHeaderView__ResizeToContents)
	asmtree.SetSelectionBehavior(widgets.QAbstractItemView__SelectRows)
	asmtree.SetEditTriggers(widgets.QAbstractItemView__NoEditTriggers)

	symtab.ConnectCurrentChanged(func(current *core.QModelIndex, previous *core.QModelIndex) {
		asmtree.SetModel(symtabModel.Asmtree(current))
	})

	symtabGroup := widgets.NewQWidget(nil, 0)
	{
		hlayout := widgets.NewQHBoxLayout()
		hlayout.AddWidget(search, 0, 0)
		hlayout.AddWidget(searchType, 0, 0)
		hlayout.AddWidget(externCheck, 0, 0)
		hlayout.SetContentsMargins(0, 0, 0, 0)

		vlayout := widgets.NewQVBoxLayout()
		vlayout.AddLayout(hlayout, 0)
		vlayout.AddWidget(symtab, 0, 0)
		vlayout.SetContentsMargins(0, 0, 0, 0)

		symtabGroup.SetLayout(vlayout)
	}

	sp := widgets.NewQSplitter2(core.Qt__Vertical, nil)
	sp.AddWidget(symtabGroup)
	sp.AddWidget(asmtree)

	w := widgets.NewQWidget(nil, 0)
	layout := widgets.NewQVBoxLayout()
	layout.AddWidget(sp, 0, 0)
	w.SetLayout(layout)

	return w
}
