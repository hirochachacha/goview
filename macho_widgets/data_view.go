package macho_widgets

import (
	"math"

	"github.com/therecipe/qt/core"
	"github.com/therecipe/qt/gui"
	"github.com/therecipe/qt/widgets"
)

const (
	defaultWidth  = 900
	defaultHeight = 450
)

type DataView struct {
	*widgets.QWidget

	tree *widgets.QTreeView
}

func (f *File) NewDataView(parent widgets.QWidget_ITF) *DataView {
	v := widgets.NewQTreeView(nil)
	v.SetSelectionBehavior(widgets.QAbstractItemView__SelectRows)
	v.SetEditTriggers(widgets.QAbstractItemView__NoEditTriggers)
	v.SetItemDelegate(NewHtmlItemDelegate(nil))
	v.ConnectMousePressEvent(func(e *gui.QMouseEvent) {
		v.MousePressEventDefault(e)

		pos := e.Pos()

		index := v.IndexAt(pos)
		if !index.IsValid() {
			return
		}

		ipos := v.VisualRect(index).TopLeft()
		rpos := core.NewQPoint2(pos.X()-ipos.X(), pos.Y()-ipos.Y())

		html := index.Data(int(core.Qt__DisplayRole)).ToString()

		doc := gui.NewQTextDocument(nil)
		doc.SetHtml(html)

		layout := doc.DocumentLayout()
		anchor := layout.AnchorAt(core.NewQPointF2(rpos))

		if len(anchor) != 0 {
			cw := f.NewAnchorWidget(anchor)
			mw := widgets.NewQMainWindow(nil, 0)
			mw.SetWindowTitle(anchor)
			mw.SetCentralWidget(cw)
			mw.Resize2(defaultWidth, defaultHeight)
			mw.Show()
		}
	})

	layout := widgets.NewQVBoxLayout()
	layout.AddWidget(v, 0, 0)

	w := widgets.NewQWidget(parent, 0)
	w.SetLayout(layout)

	return &DataView{
		QWidget: w,
		tree:    v,
	}
}

func (d *DataView) Header() *widgets.QHeaderView {
	return d.tree.Header()
}

func (d *DataView) SetAlternatingRowColors(b bool) {
	d.tree.SetAlternatingRowColors(b)
}

func (d *DataView) SetModel(m core.QAbstractItemModel_ITF) {
	d.tree.SetModel(m)
}

func NewHtmlItemDelegate(parent core.QObject_ITF) widgets.QAbstractItemDelegate_ITF {
	d := widgets.NewQStyledItemDelegate(parent)

	doc := gui.NewQTextDocument(nil)
	doc.SetDefaultStyleSheet(`body { white-space: pre; }`)

	d.ConnectPaint(func(painter *gui.QPainter, option *widgets.QStyleOptionViewItem, index *core.QModelIndex) {
		option = widgets.NewQStyleOptionViewItem2(option)

		d.InitStyleOption(option, index)

		doc.SetHtml(option.Text())

		option.SetText("")
		option.Widget().Style().DrawControl(widgets.QStyle__CE_ItemViewItem, option, painter, nil)

		rect := option.Rect()
		clip := core.NewQRectF4(0, 0, float64(rect.Width()), float64(rect.Height()))

		painter.Save()
		painter.Translate2(rect.TopLeft())
		doc.DrawContents(painter, clip)
		painter.Restore()
	})
	d.ConnectSizeHint(func(option *widgets.QStyleOptionViewItem, index *core.QModelIndex) *core.QSize {
		option = widgets.NewQStyleOptionViewItem2(option)

		d.InitStyleOption(option, index)

		doc.SetHtml(option.Text())

		return core.NewQSize2(int(math.Ceil(doc.IdealWidth())), int(math.Ceil(doc.Size().Height())))
	})
	return d
}
