package macho_widgets

import (
	"math"

	"github.com/therecipe/qt/core"
	"github.com/therecipe/qt/gui"
	"github.com/therecipe/qt/widgets"
)

func NewHtmlItemDelegate(columns ...int) widgets.QAbstractItemDelegate_ITF {
	d := widgets.NewQStyledItemDelegate(nil)

	doc := gui.NewQTextDocument(nil)
	doc.SetDefaultStyleSheet(`body { white-space: pre; }`)

	d.ConnectPaint(func(painter *gui.QPainter, option *widgets.QStyleOptionViewItem, index *core.QModelIndex) {
		var override bool
		for _, c := range columns {
			if c == index.Column() {
				override = true
				break
			}
		}

		if !override {
			d.PaintDefault(painter, option, index)
			return
		}

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
		var override bool
		for _, c := range columns {
			if c == index.Column() {
				override = true
				break
			}
		}

		if !override {
			return d.SizeHintDefault(option, index)
		}

		option = widgets.NewQStyleOptionViewItem2(option)

		d.InitStyleOption(option, index)

		doc.SetHtml(option.Text())

		return core.NewQSize2(int(math.Ceil(doc.IdealWidth())), int(math.Ceil(doc.Size().Height())))
	})
	return d
}
