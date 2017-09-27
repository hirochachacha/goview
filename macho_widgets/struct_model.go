package macho_widgets

import (
	"debug/macho"
	"fmt"
	"strings"
	"time"

	"github.com/therecipe/qt/core"
	"github.com/therecipe/qt/gui"
)

type StructModel struct {
	Tree     core.QAbstractItemModel_ITF
	attrTabs []core.QAbstractItemModel_ITF
}

const StructItemRole = int(core.Qt__UserRole) + 1

func NewStructModel(f *macho.File) *StructModel {
	m := new(StructModel)

	setItemModel := func(data [][]string) *core.QVariant {
		tab := gui.NewQStandardItemModel(nil)
		tab.SetHorizontalHeaderItem(0, gui.NewQStandardItem2("Description"))
		tab.SetHorizontalHeaderItem(1, gui.NewQStandardItem2("Value"))
		for i, es := range data {
			for j, e := range es {
				tab.SetItem(i, j, gui.NewQStandardItem2(e))
			}
		}
		m.attrTabs = append(m.attrTabs, tab)
		return core.NewQVariant7(len(m.attrTabs))
	}

	tree := gui.NewQStandardItemModel(nil)

	root := tree.InvisibleRootItem()

	file := gui.NewQStandardItem2(fileString(f))
	file.SetData(setItemModel([][]string{
		{"Magic Number", fmt.Sprintf("%#08x (%s)", f.Magic, Magic(f.Magic))},
		{"CPU Type", fmt.Sprintf("%#08x (%s)", uint32(f.Cpu), CpuType(f.Cpu))},
		{"CPU Subtype", cpusubString(f.Cpu, f.SubCpu)},
		{"File Type", fmt.Sprintf("%#08x (%s)", uint32(f.Type), FileType(f.Type))},
		{"Number of Load Commands", fmt.Sprint(f.Ncmd)},
		{"Size of Load Commands", fmt.Sprint(f.Cmdsz)},
		{"File Flags", flagsString(f.Flags, fileFlagStrings[:])},
	}), StructItemRole)

	sectDone := make(map[*macho.Section]bool)

	loads := gui.NewQStandardItem2(fmt.Sprintf("Load Commands (%d)", len(f.Loads)))
	for _, lc := range f.Loads {
		raw := lc.Raw()
		cmd := f.ByteOrder.Uint32(raw[0:4])
		cmdsize := f.ByteOrder.Uint32(raw[4:8])

		switch lc := lc.(type) {
		case *macho.Rpath:
			item := gui.NewQStandardItem2("LC_RPATH")
			item.SetData(setItemModel([][]string{
				{"Command", fmt.Sprintf("%#08x (%s)", cmd, LoadCommand(cmd))},
				{"Command Size", fmt.Sprint(cmdsize)},
				{"RPath", lc.Path},
			}), StructItemRole)
			loads.AppendRow2(item)
		case *macho.Dylib:
			item := gui.NewQStandardItem2(fmt.Sprintf("LC_LOAD_DYLIB (%s)", lc.Name))
			item.SetData(setItemModel([][]string{
				{"Command", fmt.Sprintf("%#08x (%s)", cmd, LoadCommand(cmd))},
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
				{"Command", fmt.Sprintf("%#08x (%s)", cmd, LoadCommand(cmd))},
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
				{"Command", fmt.Sprintf("%#08x (%s)", cmd, LoadCommand(cmd))},
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
				{"Command", fmt.Sprintf("%#08x (%s)", cmd, LoadCommand(cmd))},
				{"Command Size", fmt.Sprint(cmdsize)},
				{"Name", lc.Name},
				{"VM Address", fmt.Sprintf("%#016x", lc.Addr)},
				{"VM Size", fmt.Sprintf("%d", lc.Memsz)},
				{"File Offset", fmt.Sprintf("%d", lc.Offset)},
				{"File Size", fmt.Sprintf("%d", lc.Filesz)},
				{"Maximum VM Protections", fmt.Sprintf("%#o", lc.Maxprot)},
				{"Initial VM Protections", fmt.Sprintf("%#o", lc.Prot)},
				{"Number of Sections", fmt.Sprint(lc.Nsect)},
				{"Segment Flags", flagsString(lc.Flag, segmentFlagStrings[:])},
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
						{"Address", fmt.Sprintf("%#016x", sect.Addr)},
						{"Size", fmt.Sprint(sect.Size)},
						{"Offset", fmt.Sprint(sect.Offset)},
						{"Alignment", fmt.Sprintf("%d (%d)", sect.Align, 1<<sect.Align)},
						{"Offset to the first Relocation", fmt.Sprint(sect.Reloff)},
						{"Number of Relocation entries", fmt.Sprint(sect.Nreloc)},
						{"Section Flags", sectionFlagsString(sect.Flags)},
					}), StructItemRole)

					segItem.AppendRow2(sectItem)
				}
			}

			if nsect != 0 {
				// TODO warning
			}

			loads.AppendRow2(segItem)
		default:
			item := gui.NewQStandardItem2(fmt.Sprintf("%s (?)", LoadCommand(cmd)))
			item.SetData(setItemModel([][]string{
				{"Command", fmt.Sprintf("%#08x (%s)", cmd, LoadCommand(cmd))},
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

	return m
}

func (m *StructModel) AttrTab(index *core.QModelIndex) core.QAbstractItemModel_ITF {
	if val := index.Data(StructItemRole); val.IsValid() {
		if i := val.ToInt(false); 0 < i && i <= len(m.attrTabs) {
			return m.attrTabs[i-1]
		}
	}
	return nil
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
		return fmt.Sprintf("%#08x (%s)", cpusub, CpuSubtypeX86(cpusub))
	case macho.CpuAmd64:
		var s string
		if cpusub&CPU_SUBTYPE_LIB64 != 0 {
			s = "0x80000000 (CPU_SUBTYPE_LIB64)\n"
			cpusub ^= CPU_SUBTYPE_LIB64
		}
		return s + fmt.Sprintf("%#08x (%s)", cpusub, CpuSubtypeX86_64(cpusub))
	case macho.CpuArm:
		return fmt.Sprintf("%#08x (%s)", cpusub, CpuSubtypeARM(cpusub))
	case macho.CpuArm | 0x01000000:
		var s string
		if cpusub&CPU_SUBTYPE_LIB64 != 0 {
			s = "0x80000000 (CPU_SUBTYPE_LIB64)\n"
			cpusub ^= CPU_SUBTYPE_LIB64
		}
		return s + fmt.Sprintf("%#08x (%s)", cpusub, CpuSubtypeARM64(cpusub))
	case macho.CpuPpc:
		return fmt.Sprintf("%#08x (%s)", cpusub, CpuSubtypePPC(cpusub))
	case macho.CpuPpc64:
		var s string
		if cpusub&CPU_SUBTYPE_LIB64 != 0 {
			s = "0x80000000 (CPU_SUBTYPE_LIB64)\n"
			cpusub ^= CPU_SUBTYPE_LIB64
		}
		return s + fmt.Sprintf("%#08x (%s)", cpusub, CpuSubtypePPC(cpusub))
	default:
		var s string
		if cpusub&CPU_SUBTYPE_LIB64 != 0 {
			s = "0x80000000 (CPU_SUBTYPE_LIB64)\n"
			cpusub ^= CPU_SUBTYPE_LIB64
		}
		return s + fmt.Sprintf("%#08x (?)", cpusub)
	}
}

func flagsString(f uint32, strtab []string) string {
	var flags []string
	for i := 0; f != 0; i++ {
		if f&1 != 0 {
			flags = append(flags, fmt.Sprintf("%#08x (%s)", 1<<uint(i), strtab[i]))
		}
		f >>= 1
	}
	if len(flags) == 0 {
		return "0x00000000"
	}
	return strings.Join(flags, "\n")
}

func versionString(v uint32) string {
	return fmt.Sprintf("%#08x (%d.%d.%d)", v, v>>16, (v>>8)&0xff, v&0xff)
}

func sectionFlagsString(f uint32) string {
	var flags []string
	if f&SECTION_TYPE != 0 {
		flags = append(flags, fmt.Sprintf("%#08x (%s)", f&SECTION_TYPE, SectionType(f&SECTION_TYPE)))
	}
	if f&SECTION_ATTRIBUTES_SYS != 0 {
		if f&S_ATTR_LOC_RELOC != 0 {
			flags = append(flags, "0x00000100 (S_ATTR_LOC_RELOC)")
		}
		if f&S_ATTR_EXT_RELOC != 0 {
			flags = append(flags, "0x00000200 (S_ATTR_EXT_RELOC)")
		}
		if f&S_ATTR_SOME_INSTRUCTIONS != 0 {
			flags = append(flags, "0x00000400 (S_ATTR_SOME_INSTRUCTIONS)")
		}
	}
	if f&SECTION_ATTRIBUTES_USR != 0 {
		if f&S_ATTR_DEBUG != 0 {
			flags = append(flags, "0x02000000 (S_ATTR_DEBUG)")
		}
		if f&S_ATTR_SELF_MODIFYING_CODE != 0 {
			flags = append(flags, "0x04000000 (S_ATTR_SELF_MODIFYING_CODE)")
		}
		if f&S_ATTR_LIVE_SUPPORT != 0 {
			flags = append(flags, "0x08000000 (S_ATTR_LIVE_SUPPORT)")
		}
		if f&S_ATTR_NO_DEAD_STRIP != 0 {
			flags = append(flags, "0x10000000 (S_ATTR_NO_DEAD_STRIP)")
		}
		if f&S_ATTR_STRIP_STATIC_SYMS != 0 {
			flags = append(flags, "0x20000000 (S_ATTR_STRIP_STATIC_SYMS)")
		}
		if f&S_ATTR_NO_TOC != 0 {
			flags = append(flags, "0x40000000 (S_ATTR_NO_TOC)")
		}
		if f&S_ATTR_PURE_INSTRUCTIONS != 0 {
			flags = append(flags, "0x80000000 (S_ATTR_PURE_INSTRUCTIONS)")
		}
	}
	if len(flags) == 0 {
		return "0x00000000"
	}
	return strings.Join(flags, "\n")
}
