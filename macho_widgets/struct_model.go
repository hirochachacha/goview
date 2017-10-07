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
	Tree         core.QAbstractItemModel_ITF
	attrTabFuncs []func() core.QAbstractItemModel_ITF
	attrTabCache []core.QAbstractItemModel_ITF
}

const StructItemRole = int(core.Qt__UserRole) + 1

func (f *File) NewStructModel() *StructModel {
	m := new(StructModel)

	setItemModel := func(data [][]string) (*core.QVariant, int) {
		m.attrTabFuncs = append(m.attrTabFuncs, func() core.QAbstractItemModel_ITF {
			tab := gui.NewQStandardItemModel(nil)
			tab.SetHorizontalHeaderItem(0, gui.NewQStandardItem2("Field"))
			tab.SetHorizontalHeaderItem(1, gui.NewQStandardItem2("Value"))
			for i, es := range data {
				for j, e := range es {
					tab.SetItem(i, j, gui.NewQStandardItem2(e))
				}
			}
			return tab
		})
		return core.NewQVariant7(len(m.attrTabFuncs)), StructItemRole
	}

	tree := gui.NewQStandardItemModel(nil)

	root := tree.InvisibleRootItem()

	file := gui.NewQStandardItem2(f.fileString())
	file.SetData(setItemModel([][]string{
		{"magic", fmt.Sprintf("%#08x (%s)", f.Magic, Magic(f.Magic))},
		{"cputype", fmt.Sprintf("%#08x (%s)", uint32(f.Cpu), CpuType(f.Cpu))},
		{"cpusubtype", f.cpusubString(true)},
		{"filetype", fmt.Sprintf("%#08x (%s)", uint32(f.Type), FileType(f.Type))},
		{"ncmds", fmt.Sprint(f.Ncmd)},
		{"sizeofcmds", fmt.Sprintf("%#08x", f.Cmdsz)},
		{"flags", f.flagsString(f.Flags, fileFlagStrings[:], true)},
	}))

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
				{"cmd", fmt.Sprintf("%#08x (%s)", cmd, LoadCommand(cmd))},
				{"cmdsize", fmt.Sprintf("%#08x", cmdsize)},
				{"path", lc.Path},
			}))
			loads.AppendRow2(item)
		case *macho.Dylib:
			item := gui.NewQStandardItem2(fmt.Sprintf("LC_LOAD_DYLIB (%s)", lc.Name))
			item.SetData(setItemModel([][]string{
				{"cmd", fmt.Sprintf("%#08x (%s)", cmd, LoadCommand(cmd))},
				{"cmdsize", fmt.Sprintf("%#08x", cmdsize)},
				{"name", lc.Name},
				{"timestamp", time.Unix(int64(lc.Time), 0).String()},
				{"current_version", f.versionString(lc.CurrentVersion)},
				{"compatibility_version", f.versionString(lc.CompatVersion)},
			}))
			loads.AppendRow2(item)
		case *macho.Symtab:
			item := gui.NewQStandardItem2("LC_SYMTAB")
			item.SetData(setItemModel([][]string{
				{"cmd", fmt.Sprintf("%#08x (%s)", cmd, LoadCommand(cmd))},
				{"cmdsize", fmt.Sprintf("%#08x", cmdsize)},
				{"symoff", fmt.Sprintf("%#08x", lc.SymtabCmd.Symoff)},
				{"nsyms", fmt.Sprint(lc.SymtabCmd.Nsyms)},
				{"stroff", fmt.Sprintf("%#08x", lc.SymtabCmd.Stroff)},
				{"strsize", fmt.Sprintf("%#08x", lc.SymtabCmd.Strsize)},
			}))
			loads.AppendRow2(item)
		case *macho.Dysymtab:
			item := gui.NewQStandardItem2("LC_DYSYMTAB")
			item.SetData(setItemModel([][]string{
				{"cmd", fmt.Sprintf("%#08x (%s)", cmd, LoadCommand(cmd))},
				{"cmdsize", fmt.Sprintf("%#08x", cmdsize)},
				{"ilocalsym", fmt.Sprint(lc.DysymtabCmd.Ilocalsym)},
				{"nlocalsym", fmt.Sprint(lc.DysymtabCmd.Nlocalsym)},
				{"iextdefsym", fmt.Sprint(lc.DysymtabCmd.Iextdefsym)},
				{"nextdefsym", fmt.Sprint(lc.DysymtabCmd.Nextdefsym)},
				{"iundefsym", fmt.Sprint(lc.DysymtabCmd.Iundefsym)},
				{"nundefsym", fmt.Sprint(lc.DysymtabCmd.Nundefsym)},
				{"tocoff", fmt.Sprintf("%#08x", lc.DysymtabCmd.Tocoffset)},
				{"ntoc", fmt.Sprint(lc.DysymtabCmd.Ntoc)},
				{"modtaboff", fmt.Sprintf("%#08x", lc.DysymtabCmd.Modtaboff)},
				{"nmodtab", fmt.Sprint(lc.DysymtabCmd.Modtaboff)},
				{"extrefsymoff", fmt.Sprintf("%#08x", lc.DysymtabCmd.Extrefsymoff)},
				{"nextrefsyms", fmt.Sprint(lc.DysymtabCmd.Nextrefsyms)},
				{"indirectsymoff", fmt.Sprintf("%#08x", lc.DysymtabCmd.Indirectsymoff)},
				{"nindirectsyms", fmt.Sprint(lc.DysymtabCmd.Nindirectsyms)},
				{"extreloff", fmt.Sprintf("%#08x", lc.DysymtabCmd.Extreloff)},
				{"nextrel", fmt.Sprint(lc.DysymtabCmd.Nextrel)},
				{"locreloff", fmt.Sprintf("%#08x", lc.DysymtabCmd.Locreloff)},
				{"nlocrel", fmt.Sprintf("%d", lc.DysymtabCmd.Nlocrel)},
			}))
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
				{"cmd", fmt.Sprintf("%#08x (%s)", cmd, LoadCommand(cmd))},
				{"cmdsize", fmt.Sprintf("%#08x", cmdsize)},
				{"segname", lc.Name},
				{"vmaddr", fmt.Sprintf("%#016x", lc.Addr)},
				{"vmsize", fmt.Sprintf("%#016x", lc.Memsz)},
				{"fileoff", fmt.Sprintf("%#016x", lc.Offset)},
				{"filesize", fmt.Sprintf("%#016x", lc.Filesz)},
				{"maxprot", f.vmprotString(lc.Maxprot)},
				{"initprot", f.vmprotString(lc.Prot)},
				{"nsects", fmt.Sprint(lc.Nsect)},
				{"flags", f.flagsString(lc.Flag, segmentFlagStrings[:], true)},
			}))

			nsect := lc.Nsect

			for i, sect := range f.Sections {
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

					sectItem := gui.NewQStandardItem2(fmt.Sprintf("Section %d (%s,%s)", i+1, sect.Seg, sect.Name))
					sectItem.SetData(setItemModel([][]string{
						{"sectname", sect.Name},
						{"segname", sect.Seg},
						{"addr", fmt.Sprintf("%#016x", sect.Addr)},
						{"size", fmt.Sprintf("%#016x", sect.Size)},
						{"offset", fmt.Sprintf("%#016x", sect.Offset)},
						{"align", fmt.Sprintf("%d (%d)", sect.Align, 1<<sect.Align)},
						{"reloff", fmt.Sprintf("%#016x", sect.Reloff)},
						{"nreloc", fmt.Sprint(sect.Nreloc)},
						{"flags", f.sectionFlagsString(sect.Flags, true)},
					}))

					// TODO use tree instead of table, so relocation will be visible

					s := sect

					m.attrTabFuncs = append(m.attrTabFuncs, func() core.QAbstractItemModel_ITF {
						return f.NewSectionModel(f.guessSectType(s), s, 0, 0)
					})

					sectDataItem := gui.NewQStandardItem2("Data")
					sectDataItem.SetData(core.NewQVariant7(len(m.attrTabFuncs)), StructItemRole)

					sectItem.AppendRow2(sectDataItem)
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
				{"cmd", fmt.Sprintf("%#08x (%s)", cmd, LoadCommand(cmd))},
				{"cmdsize", fmt.Sprintf("%#08x", cmdsize)},
			}))
			loads.AppendRow2(item)
		}
	}

	m.attrTabCache = make([]core.QAbstractItemModel_ITF, len(m.attrTabFuncs))

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
		if i := val.ToInt(false); 0 < i && i <= len(m.attrTabFuncs) {
			if cache := m.attrTabCache[i-1]; cache != nil {
				return cache
			}
			tab := m.attrTabFuncs[i-1]()
			m.attrTabCache[i-1] = tab
			return tab
		}
	}
	return nil
}

func (f *File) fileString() string {
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

func (f *File) cpusubString(html bool) string {
	var s string

	cpusub := f.SubCpu

	switch f.Cpu {
	case macho.Cpu386:
		s = fmt.Sprintf("%#08x (%s)", cpusub, CpuSubtypeX86(cpusub))
	case macho.CpuAmd64:
		if cpusub&CPU_SUBTYPE_LIB64 != 0 {
			s = "0x80000000 (CPU_SUBTYPE_LIB64)\n"
			cpusub ^= CPU_SUBTYPE_LIB64
		}
		s += fmt.Sprintf("%#08x (%s)", cpusub, CpuSubtypeX86_64(cpusub))
	case macho.CpuArm:
		s = fmt.Sprintf("%#08x (%s)", cpusub, CpuSubtypeARM(cpusub))
	case macho.CpuArm | 0x01000000:
		if cpusub&CPU_SUBTYPE_LIB64 != 0 {
			s = "0x80000000 (CPU_SUBTYPE_LIB64)\n"
			cpusub ^= CPU_SUBTYPE_LIB64
		}
		s += fmt.Sprintf("%#08x (%s)", cpusub, CpuSubtypeARM64(cpusub))
	case macho.CpuPpc:
		s = fmt.Sprintf("%#08x (%s)", cpusub, CpuSubtypePPC(cpusub))
	case macho.CpuPpc64:
		if cpusub&CPU_SUBTYPE_LIB64 != 0 {
			s = "0x80000000 (CPU_SUBTYPE_LIB64)\n"
			cpusub ^= CPU_SUBTYPE_LIB64
		}
		s += fmt.Sprintf("%#08x (%s)", cpusub, CpuSubtypePPC(cpusub))
	default:
		if cpusub&CPU_SUBTYPE_LIB64 != 0 {
			s = "0x80000000 (CPU_SUBTYPE_LIB64)\n"
			cpusub ^= CPU_SUBTYPE_LIB64
		}
		s += fmt.Sprintf("%#08x (?)", cpusub)
	}
	if html {
		s = "<body>" + s + "</body>"
	}
	return s
}

func (_ *File) flagsString(f uint32, strtab []string, html bool) string {
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
	s := strings.Join(flags, "\n")
	if html {
		s = "<body>" + s + "</body>"
	}
	return s
}

func (f *File) versionString(v uint32) string {
	return fmt.Sprintf("%#08x (%d.%d.%d)", v, v>>16, (v>>8)&0xff, v&0xff)
}

func (f *File) vmprotString(prot uint32) string {
	s := ""
	if prot&4 != 0 {
		s += "r"
	} else {
		s += "-"
	}
	if prot&2 != 0 {
		s += "w"
	} else {
		s += "-"
	}
	if prot&1 != 0 {
		s += "x"
	} else {
		s += "-"
	}
	return fmt.Sprintf("%#o (%s)", prot, s)
}

func (_ *File) sectionFlagsString(f uint32, html bool) string {
	var flags []string

	flags = append(flags, fmt.Sprintf("%#08x (%s)", f&SECTION_TYPE, SectionType(f&SECTION_TYPE)))

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

	s := strings.Join(flags, "\n")
	if html {
		s = "<body>" + s + "</body>"
	}
	return s
}
