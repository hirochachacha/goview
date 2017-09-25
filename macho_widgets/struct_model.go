package macho_widgets

import (
	"debug/macho"
	"fmt"
	"time"

	"github.com/therecipe/qt/core"
	"github.com/therecipe/qt/gui"
)

type StructModel struct {
	Tree   core.QAbstractItemModel_ITF
	Tables []core.QAbstractItemModel_ITF
}

func NewStructModel(f *macho.File) (*StructModel, error) {
	m := new(StructModel)

	tree := gui.NewQStandardItemModel(nil)

	setItemModel := func(data [][]string) *core.QVariant {
		table := gui.NewQStandardItemModel(nil)
		table.SetHorizontalHeaderItem(0, gui.NewQStandardItem2("Description"))
		table.SetHorizontalHeaderItem(1, gui.NewQStandardItem2("Value"))
		for i, es := range data {
			for j, e := range es {
				table.SetItem(i, j, gui.NewQStandardItem2(e))
			}
		}
		m.Tables = append(m.Tables, table)
		return core.NewQVariant7(len(m.Tables))
	}

	root := tree.InvisibleRootItem()

	file := gui.NewQStandardItem2(fileString(f))
	file.SetData(setItemModel([][]string{
		{"Magic Number", Magic(f.Magic).String()},
		{"CPU Type", CpuType(f.Cpu).String()},
		{"CPU Subtype", cpusubString(f.Cpu, f.SubCpu)},
		{"File Type", FileType(f.Type).String()},
		{"Number of Load Commands", fmt.Sprint(f.Ncmd)},
		{"Size of Load Commands", fmt.Sprint(f.Cmdsz)},
		{"File Flags", FileFlag(f.Flags).String()},
	}), StructItemRole)

	sectDone := make(map[*macho.Section]bool)

	loads := gui.NewQStandardItem2(fmt.Sprintf("Load Commands (%d)", len(f.Loads)))
	for _, lc := range f.Loads {
		raw := lc.Raw()
		cmd := macho.LoadCmd(f.ByteOrder.Uint32(raw[0:4]))
		cmdsize := f.ByteOrder.Uint32(raw[4:8])

		switch lc := lc.(type) {
		case *macho.Rpath:
			item := gui.NewQStandardItem2("LC_RPATH")
			item.SetData(setItemModel([][]string{
				{"Command", LoadCommand(cmd).String()},
				{"Command Size", fmt.Sprint(cmdsize)},
				{"RPath", lc.Path},
			}), StructItemRole)
			loads.AppendRow2(item)
		case *macho.Dylib:
			item := gui.NewQStandardItem2(fmt.Sprintf("LC_LOAD_DYLIB (%s)", lc.Name))
			item.SetData(setItemModel([][]string{
				{"Command", LoadCommand(cmd).String()},
				{"Command Size", fmt.Sprint(cmdsize)},
				{"Name", lc.Name},
				{"Timestamp", time.Unix(int64(lc.Time), 0).String()},
				{"Current Version", versionString(lc.CurrentVersion)},
				{"Compatibility Version", versionString(lc.CompatVersion)},
			}), StructItemRole)
			loads.AppendRow2(item)
		case *macho.Symtab:
			item := gui.NewQStandardItem2("LC_SYMTAB")
			item.SetData(setItemModel([][]string{
				{"Command", LoadCommand(cmd).String()},
				{"Command Size", fmt.Sprint(cmdsize)},
				{"Symbol Table Offset", fmt.Sprint(lc.SymtabCmd.Symoff)},
				{"Number of Symbols", fmt.Sprint(lc.SymtabCmd.Nsyms)},
				{"String Table Offset", fmt.Sprint(lc.SymtabCmd.Stroff)},
				{"String Table Size", fmt.Sprint(lc.SymtabCmd.Strsize)},
			}), StructItemRole)
			loads.AppendRow2(item)
		case *macho.Dysymtab:
			item := gui.NewQStandardItem2("LC_DYSYMTAB")
			item.SetData(setItemModel([][]string{
				{"Command", LoadCommand(cmd).String()},
				{"Command Size", fmt.Sprint(cmdsize)},
				{"Index of The First Local Symbol", fmt.Sprint(lc.DysymtabCmd.Ilocalsym)},
				{"Number of Local Symbols", fmt.Sprint(lc.DysymtabCmd.Nlocalsym)},
				{"Index of the first External Symbol", fmt.Sprint(lc.DysymtabCmd.Iextdefsym)},
				{"Number of External Symbols", fmt.Sprint(lc.DysymtabCmd.Nextdefsym)},
				{"Index of the first Undefined Symbol", fmt.Sprint(lc.DysymtabCmd.Iundefsym)},
				{"Number of Undefined Symbols", fmt.Sprint(lc.DysymtabCmd.Nundefsym)},
				{"Offset to the TOC", fmt.Sprintf("%d", lc.DysymtabCmd.Tocoffset)},
				{"Number of TOC entries", fmt.Sprint(lc.DysymtabCmd.Ntoc)},
				{"Offset to the Module Table", fmt.Sprintf("%d", lc.DysymtabCmd.Modtaboff)},
				{"Number of the Module Table entries", fmt.Sprint(lc.DysymtabCmd.Modtaboff)},
				{"Offset to the External Reference Table", fmt.Sprintf("%d", lc.DysymtabCmd.Extrefsymoff)},
				{"Number of the External Reference Table entries", fmt.Sprint(lc.DysymtabCmd.Nextrefsyms)},
				{"Offset to the Indirect Symbol Table", fmt.Sprintf("%d", lc.DysymtabCmd.Indirectsymoff)},
				{"Number of the Indirect Symbol Table entries", fmt.Sprint(lc.DysymtabCmd.Nindirectsyms)},
				{"Offset to the External Relocation Table", fmt.Sprintf("%d", lc.DysymtabCmd.Extreloff)},
				{"Number of the External Relocation Table entries", fmt.Sprint(lc.DysymtabCmd.Nextrel)},
				{"Offset to the Local Relocation Table", fmt.Sprint(lc.DysymtabCmd.Locreloff)},
				{"Number of the Local Relocation Table entries", fmt.Sprintf("%d", lc.DysymtabCmd.Nlocrel)},
			}), StructItemRole)
			loads.AppendRow2(item)
		case *macho.Segment:
			var segItem *gui.QStandardItem
			switch lc.Cmd {
			case macho.LoadCmdSegment:
				segItem = gui.NewQStandardItem2(fmt.Sprintf("LC_SEGMENT (%s) (%d)", lc.Name, lc.Nsect))
			case macho.LoadCmdSegment64:
				segItem = gui.NewQStandardItem2(fmt.Sprintf("LC_SEGMENT_64 (%s) (%d)", lc.Name, lc.Nsect))
			default:
				panic("unreachable")
			}

			segItem.SetData(setItemModel([][]string{
				{"Command", LoadCommand(cmd).String()},
				{"Command Size", fmt.Sprint(cmdsize)},
				{"Name", lc.Name},
				{"VM Address", fmt.Sprintf("%#x", lc.Addr)},
				{"VM Size", fmt.Sprintf("%d", lc.Memsz)},
				{"File Offset", fmt.Sprintf("%d", lc.Offset)},
				{"File Size", fmt.Sprintf("%d", lc.Filesz)},
				{"Maximum VM Protections", fmt.Sprintf("%#o", lc.Maxprot)},
				{"Initial VM Protections", fmt.Sprintf("%#o", lc.Prot)},
				{"Number of Sections", fmt.Sprint(lc.Nsect)},
				{"Segment Flags", SegmentFlag(lc.Flag).String()},
			}), StructItemRole)

			nsect := lc.Nsect

			for _, sect := range f.Sections {
				if lc.Addr <= sect.Addr && sect.Addr+sect.Size <= lc.Addr+lc.Memsz {
					if lc.Name == sect.Seg {
						nsect--
					} else {
						// TODO warning
					}
					if sectDone[sect] {
						// TODO warning
					} else {
						sectDone[sect] = true
					}

					sectItem := gui.NewQStandardItem2(fmt.Sprintf("Section (%s,%s)", sect.Seg, sect.Name))
					sectItem.SetData(setItemModel([][]string{
						{"Name", sect.Name},
						{"Segment Name", sect.Seg},
						{"Address", fmt.Sprintf("%#x", sect.Addr)},
						{"Size", fmt.Sprint(sect.Size)},
						{"Offset", fmt.Sprint(sect.Offset)},
						{"Alignment", fmt.Sprint(sect.Align)},
						{"Offset to the first Relocation", fmt.Sprint(sect.Reloff)},
						{"Number of Relocation entries", fmt.Sprint(sect.Nreloc)},
					}), StructItemRole)

					segItem.AppendRow2(sectItem)
				}
			}

			if nsect != 0 {
				// TODO warning
			}

			loads.AppendRow2(segItem)
		default:
			item := gui.NewQStandardItem2("?")
			item.SetData(setItemModel([][]string{
				{"Command", LoadCommand(cmd).String()},
				{"Command Size", fmt.Sprint(cmdsize)},
			}), StructItemRole)
			loads.AppendRow2(item)
		}
	}

	for _, sect := range f.Sections {
		if !sectDone[sect] {
			// TODO warning
		}
	}

	file.AppendRow2(loads)

	root.AppendRow2(file)

	m.Tree = tree

	return m, nil
}

func fileString(f *macho.File) string {
	typeString := func(typ macho.Type) string {
		switch typ {
		case macho.TypeObj:
			return "Object"
		case macho.TypeExec:
			return "Executable"
		case macho.TypeDylib:
			return "Dynamic Library"
		case macho.TypeBundle:
			return "Bundle"
		default:
			return "?"
		}
	}
	cpuString := func(cpu macho.Cpu) string {
		switch cpu {
		case macho.Cpu386:
			return "386"
		case macho.CpuAmd64:
			return "AMD64"
		case macho.CpuArm:
			return "ARM"
		case macho.CpuArm | 0x01000000:
			return "ARM64"
		case macho.CpuPpc:
			return "PPC"
		case macho.CpuPpc64:
			return "PPC64"
		default:
			return "?"
		}
	}
	return fmt.Sprintf("%s (%s)", typeString(f.Type), cpuString(f.Cpu))
}

func cpusubString(cpu macho.Cpu, cpusub uint32) string {
	switch cpu {
	case macho.Cpu386:
		return CpuSubtypeX86(cpusub).String()
	case macho.CpuAmd64:
		return CpuSubtypeX86_64(cpusub).String()
	case macho.CpuArm:
		return CpuSubtypeARM(cpusub).String()
	case macho.CpuArm | 0x01000000:
		return CpuSubtypeARM64(cpusub).String()
	case macho.CpuPpc:
		return CpuSubtypePPC(cpusub).String()
	case macho.CpuPpc64:
		return CpuSubtypePPC(cpusub).String()
	default:
		return "?"
	}
}

func versionString(v uint32) string {
	return fmt.Sprintf("%d.%d.%d", v>>16, (v>>8)&0xff, v&0xff)
}
