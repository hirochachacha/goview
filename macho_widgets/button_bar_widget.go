package macho_widgets

import "github.com/therecipe/qt/widgets"

type ButtonBarWidget struct {
	*widgets.QWidget

	bg *widgets.QButtonGroup

	labels []string
}

func (f *File) NewButtonBarWidget(parent widgets.QWidget_ITF, labels []string) *ButtonBarWidget {
	bg := widgets.NewQButtonGroup(nil)

	hlayout := widgets.NewQHBoxLayout()

	for i, label := range labels {
		btn := widgets.NewQPushButton2(label, nil)
		btn.SetCheckable(true)

		bg.AddButton(btn, i+1)

		hlayout.AddWidget(btn, 0, 0)
	}

	hlayout.SetContentsMargins(0, 0, 0, 0)
	hlayout.SetSpacing(0)

	w := widgets.NewQWidget(parent, 0)
	w.SetLayout(hlayout)

	return &ButtonBarWidget{
		QWidget: w,
		bg:      bg,
		labels:  labels,
	}
}

func (bb *ButtonBarWidget) SetExclusive(exclusive bool) {
	bb.bg.SetExclusive(exclusive)
}

func (bb *ButtonBarWidget) Toggle(label string) {
	for i, l := range bb.labels {
		if l == label {
			bb.bg.Button(i + 1).Toggle()
			return
		}
	}
}

func (bb *ButtonBarWidget) SetChecked(label string, checked bool) {
	for i, l := range bb.labels {
		if l == label {
			bb.bg.Button(i + 1).SetChecked(checked)
			return
		}
	}
}

func (bb *ButtonBarWidget) ConnectButtonToggled2(f func(string, bool)) {
	bb.bg.ConnectButtonToggled2(func(id int, checked bool) {
		f(bb.labels[id-1], checked)
	})
}
