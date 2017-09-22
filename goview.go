package main

import (
	"debug/macho"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/hirochachacha/goview/macho_widgets"
	"github.com/therecipe/qt/gui"
	"github.com/therecipe/qt/widgets"
)

const (
	defaultWidth  = 900
	defaultHeight = 450
)

var rsrcPath string

func init() {
	if runtime.GOOS == "darwin" {
		rsrcPath = ":/qml/images/mac"
	} else {
		rsrcPath = ":/qml/images/win"
	}
}

func main() {
	app := widgets.NewQApplication(len(os.Args), os.Args)
	app.SetApplicationName("GoView")
	app.SetApplicationVersion("0.0.1")

	mw, err := NewMainWindow(os.Args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	mw.Show()

	os.Exit(app.Exec())
}

type MainWindow struct {
	*widgets.QMainWindow
}

func NewMainWindow(args []string) (*MainWindow, error) {
	mw := &MainWindow{widgets.NewQMainWindow(nil, 0)}

	var path string

	if len(args) <= 1 {
		var err error
		path, err = mw.openFile()
		if err != nil {
			return nil, err
		}
	} else {
		path = args[1]
	}

	cw, err := NewCentralWidget(path)
	if err != nil {
		return nil, err
	}

	mw.addMenu()
	mw.SetWindowTitle(filepath.Base(path))
	mw.SetCentralWidget(cw)
	mw.Resize2(defaultWidth, defaultHeight)

	return mw, nil
}

func (mw *MainWindow) addMenu() {
	menu := mw.MenuBar().AddMenu2("&File")
	icon := gui.QIcon_FromTheme2("document-open", gui.NewQIcon5(rsrcPath+"/fileopen.png"))
	a := menu.AddAction2(icon, "&Open...")
	a.ConnectTriggered(func(checked bool) {
		path, err := mw.openFile()
		if err != nil {
			msg := widgets.NewQErrorMessage(mw.QMainWindow)
			msg.ShowMessage(err.Error())
			return
		}
		cw, err := NewCentralWidget(path)
		if err != nil {
			msg := widgets.NewQErrorMessage(mw.QMainWindow)
			msg.ShowMessage(err.Error())
			return
		}
		mw := &MainWindow{widgets.NewQMainWindow(nil, 0)}
		mw.addMenu()
		mw.SetWindowTitle(filepath.Base(path))
		mw.SetCentralWidget(cw)
		mw.Resize2(defaultWidth, defaultHeight)
		mw.Show()
	})
	a.SetShortcuts2(gui.QKeySequence__Open)
}

func (mw *MainWindow) openFile() (string, error) {
	dialog := widgets.NewQFileDialog2(mw, "Open File...", "", "")
	dialog.SetAcceptMode(widgets.QFileDialog__AcceptOpen)
	dialog.SetFileMode(widgets.QFileDialog__ExistingFile)
	dialog.SetMimeTypeFilters([]string{"application/octet-stream"})
	if dialog.Exec() != int(widgets.QDialog__Accepted) {
		return "", errors.New("openFile failed")
	}
	files := dialog.SelectedFiles()
	if len(files) != 1 {
		return "", errors.New("openFile failed")
	}
	return files[0], nil
}

func NewCentralWidget(path string) (widgets.QWidget_ITF, error) {
	tab := widgets.NewQTabWidget(nil)

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	mf, err := macho.NewFile(f)
	if err != nil {
		return nil, err
	}

	strct, err := macho_widgets.NewStructWidget(mf)
	if err != nil {
		return nil, err
	}

	symtab, err := macho_widgets.NewSymtabWidget(mf)
	if err != nil {
		return nil, err
	}

	tab.AddTab(strct, "Struct")
	tab.AddTab(symtab, "Symtab")

	return tab, nil
}
