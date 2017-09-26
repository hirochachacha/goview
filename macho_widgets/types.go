//go:generate stringer -type=CpuType,CpuSubtypeX86,CpuSubtypeX86_64,CpuSubtypePPC,CpuSubtypeARM,CpuSubtypeARM64,Magic,FileType,LoadCommand,ReferenceType -output types_string.go
package macho_widgets

type CpuType uint32

const (
	CPU_TYPE_VAX       CpuType = 0x1
	CPU_TYPE_MC680x0   CpuType = 0x6
	CPU_TYPE_X86       CpuType = 0x7
	CPU_TYPE_I386      CpuType = 0x7
	CPU_TYPE_X86_64    CpuType = 0x1000007
	CPU_TYPE_MC98000   CpuType = 0xa
	CPU_TYPE_HPPA      CpuType = 0xb
	CPU_TYPE_ARM       CpuType = 0xc
	CPU_TYPE_ARM64     CpuType = 0x100000c
	CPU_TYPE_MC88000   CpuType = 0xd
	CPU_TYPE_SPARC     CpuType = 0xe
	CPU_TYPE_I860      CpuType = 0xf
	CPU_TYPE_POWERPC   CpuType = 0x12
	CPU_TYPE_POWERPC64 CpuType = 0x1000012
)

type (
	CpuSubtypeX86    uint32
	CpuSubtypeX86_64 uint32
	CpuSubtypePPC    uint32
	CpuSubtypeARM    uint32
	CpuSubtypeARM64  uint32
)

const (
	CPU_SUBTYPE_LIB64 = 0x80000000

	CPU_SUBTYPE_X86_ALL   CpuSubtypeX86 = 0x3
	CPU_SUBTYPE_X86_ARCH1 CpuSubtypeX86 = 0x4

	CPU_SUBTYPE_X86_64_ALL CpuSubtypeX86_64 = 0x3
	CPU_SUBTYPE_X86_64_H   CpuSubtypeX86_64 = 0x8

	CPU_SUBTYPE_POWERPC_ALL   CpuSubtypePPC = 0x0
	CPU_SUBTYPE_POWERPC_601   CpuSubtypePPC = 0x1
	CPU_SUBTYPE_POWERPC_602   CpuSubtypePPC = 0x2
	CPU_SUBTYPE_POWERPC_603   CpuSubtypePPC = 0x3
	CPU_SUBTYPE_POWERPC_603e  CpuSubtypePPC = 0x4
	CPU_SUBTYPE_POWERPC_603ev CpuSubtypePPC = 0x5
	CPU_SUBTYPE_POWERPC_604   CpuSubtypePPC = 0x6
	CPU_SUBTYPE_POWERPC_604e  CpuSubtypePPC = 0x7
	CPU_SUBTYPE_POWERPC_620   CpuSubtypePPC = 0x8
	CPU_SUBTYPE_POWERPC_750   CpuSubtypePPC = 0x9
	CPU_SUBTYPE_POWERPC_7400  CpuSubtypePPC = 0xa
	CPU_SUBTYPE_POWERPC_7450  CpuSubtypePPC = 0xb
	CPU_SUBTYPE_POWERPC_970   CpuSubtypePPC = 0x64

	CPU_SUBTYPE_ARM_ALL    CpuSubtypeARM = 0x0
	CPU_SUBTYPE_ARM_V4T    CpuSubtypeARM = 0x5
	CPU_SUBTYPE_ARM_V6     CpuSubtypeARM = 0x6
	CPU_SUBTYPE_ARM_V5TEJ  CpuSubtypeARM = 0x7
	CPU_SUBTYPE_ARM_XSCALE CpuSubtypeARM = 0x8
	CPU_SUBTYPE_ARM_V7     CpuSubtypeARM = 0x9
	CPU_SUBTYPE_ARM_V7F    CpuSubtypeARM = 0xa
	CPU_SUBTYPE_ARM_V7S    CpuSubtypeARM = 0xb
	CPU_SUBTYPE_ARM_V7K    CpuSubtypeARM = 0xc
	CPU_SUBTYPE_ARM_V6M    CpuSubtypeARM = 0xe
	CPU_SUBTYPE_ARM_V7M    CpuSubtypeARM = 0xf
	CPU_SUBTYPE_ARM_V7EM   CpuSubtypeARM = 0x10
	CPU_SUBTYPE_ARM_V8     CpuSubtypeARM = 0xd

	CPU_SUBTYPE_ARM64_ALL CpuSubtypeARM64 = 0x0
	CPU_SUBTYPE_ARM64_V8  CpuSubtypeARM64 = 0x1
)

type Magic uint32

const (
	MH_MAGIC     Magic = 0xfeedface
	MH_CIGAM     Magic = 0xcefaedfe
	MH_MAGIC_64  Magic = 0xfeedfacf
	MH_CIGAM_64  Magic = 0xcffaedfe
	FAT_MAGIC    Magic = 0xcafebabe
	FAT_CIGAM    Magic = 0xbebafeca
	FAT_MAGIC_64 Magic = 0xcafebabf
	FAT_CIGAM_64 Magic = 0xbfbafeca
)

type FileType uint32

const (
	MH_OBJECT      FileType = 0x1
	MH_EXECUTE     FileType = 0x2
	MH_FVMLIB      FileType = 0x3
	MH_CORE        FileType = 0x4
	MH_PRELOAD     FileType = 0x5
	MH_DYLIB       FileType = 0x6
	MH_DYLINKER    FileType = 0x7
	MH_BUNDLE      FileType = 0x8
	MH_DYLIB_STUB  FileType = 0x9
	MH_DSYM        FileType = 0xa
	MH_KEXT_BUNDLE FileType = 0xb
)

type FileFlag uint32

const (
	MH_NOUNDEFS                FileFlag = 0x1
	MH_INCRLINK                FileFlag = 0x2
	MH_DYLDLINK                FileFlag = 0x4
	MH_BINDATLOAD              FileFlag = 0x8
	MH_PREBOUND                FileFlag = 0x10
	MH_SPLIT_SEGS              FileFlag = 0x20
	MH_LAZY_INIT               FileFlag = 0x40
	MH_TWOLEVEL                FileFlag = 0x80
	MH_FORCE_FLAT              FileFlag = 0x100
	MH_NOMULTIDEFS             FileFlag = 0x200
	MH_NOFIXPREBINDING         FileFlag = 0x400
	MH_PREBINDABLE             FileFlag = 0x800
	MH_ALLMODSBOUND            FileFlag = 0x1000
	MH_SUBSECTIONS_VIA_SYMBOLS FileFlag = 0x2000
	MH_CANONICAL               FileFlag = 0x400
	MH_WEAK_DEFINES            FileFlag = 0x8000
	MH_BINDS_TO_WEAK           FileFlag = 0x10000
	MH_ALLOW_STACK_EXECUTION   FileFlag = 0x20000
	MH_ROOT_SAFE               FileFlag = 0x40000
	MH_SETUID_SAFE             FileFlag = 0x80000
	MH_NO_REEXPORTED_DYLIBS    FileFlag = 0x100000
	MH_PIE                     FileFlag = 0x200000
	MH_DEAD_STRIPPABLE_DYLIB   FileFlag = 0x400000
	MH_HAS_TLV_DESCRIPTORS     FileFlag = 0x800000
	MH_NO_HEAP_EXECUTION       FileFlag = 0x1000000
	MH_APP_EXTENSION_SAFE      FileFlag = 0x02000000
)

var fileFlagStrings = [...]string{
	"MH_NOUNDEFS",
	"MH_INCRLINK",
	"MH_DYLDLINK",
	"MH_BINDATLOAD",
	"MH_PREBOUND",
	"MH_SPLIT_SEGS",
	"MH_LAZY_INIT",
	"MH_TWOLEVEL",
	"MH_FORCE_FLAT",
	"MH_NOMULTIDEFS",
	"MH_NOFIXPREBINDING",
	"MH_PREBINDABLE",
	"MH_ALLMODSBOUND",
	"MH_SUBSECTIONS_VIA_SYMBOLS",
	"MH_CANONICAL",
	"MH_WEAK_DEFINES",
	"MH_BINDS_TO_WEAK",
	"MH_ALLOW_STACK_EXECUTION",
	"MH_ROOT_SAFE",
	"MH_SETUID_SAFE",
	"MH_NO_REEXPORTED_DYLIBS",
	"MH_PIE",
	"MH_DEAD_STRIPPABLE_DYLIB",
	"MH_HAS_TLV_DESCRIPTORS",
	"MH_NO_HEAP_EXECUTION",
	"MH_APP_EXTENSION_SAFE",
}

type SegmentFlag uint32

const (
	SG_HIGHVM              SegmentFlag = 0x1
	SG_FVMLIB              SegmentFlag = 0x2
	SG_NORELOC             SegmentFlag = 0x4
	SG_PROTECTED_VERSION_1 SegmentFlag = 0x8
)

var segmentFlagStrings = [...]string{
	"SG_HIGHVM",
	"SG_FVMLIB",
	"SG_NORELOC",
	"SG_PROTECTED_VERSION_1",
}

type LoadCommand uint32

const (
	LC_REQ_DYLD                 LoadCommand = 0x80000000
	LC_SEGMENT                  LoadCommand = 0x1
	LC_SYMTAB                   LoadCommand = 0x2
	LC_SYMSEG                   LoadCommand = 0x3
	LC_THREAD                   LoadCommand = 0x4
	LC_UNIXTHREAD               LoadCommand = 0x5
	LC_LOADFVMLIB               LoadCommand = 0x6
	LC_IDFVMLIB                 LoadCommand = 0x7
	LC_IDENT                    LoadCommand = 0x8
	LC_FVMFILE                  LoadCommand = 0x9
	LC_PREPAGE                  LoadCommand = 0xa
	LC_DYSYMTAB                 LoadCommand = 0xb
	LC_LOAD_DYLIB               LoadCommand = 0xc
	LC_ID_DYLIB                 LoadCommand = 0xd
	LC_LOAD_DYLINKER            LoadCommand = 0xe
	LC_ID_DYLINKER              LoadCommand = 0xf
	LC_PREBOUND_DYLIB           LoadCommand = 0x10
	LC_ROUTINES                 LoadCommand = 0x11
	LC_SUB_FRAMEWORK            LoadCommand = 0x12
	LC_SUB_UMBRELLA             LoadCommand = 0x13
	LC_SUB_CLIENT               LoadCommand = 0x14
	LC_SUB_LIBRARY              LoadCommand = 0x15
	LC_TWOLEVEL_HINTS           LoadCommand = 0x16
	LC_PREBIND_CKSUM            LoadCommand = 0x17
	LC_LOAD_WEAK_DYLIB          LoadCommand = 0x80000018
	LC_SEGMENT_64               LoadCommand = 0x19
	LC_ROUTINES_64              LoadCommand = 0x1a
	LC_UUID                     LoadCommand = 0x1b
	LC_RPATH                    LoadCommand = 0x8000001c
	LC_CODE_SIGNATURE           LoadCommand = 0x1d
	LC_SEGMENT_SPLIT_INFO       LoadCommand = 0x1e
	LC_REEXPORT_DYLIB           LoadCommand = 0x8000001f
	LC_LAZY_LOAD_DYLIB          LoadCommand = 0x20
	LC_ENCRYPTION_INFO          LoadCommand = 0x21
	LC_DYLD_INFO                LoadCommand = 0x22
	LC_DYLD_INFO_ONLY           LoadCommand = 0x80000022
	LC_LOAD_UPWARD_DYLIB        LoadCommand = 0x80000023
	LC_VERSION_MIN_MACOSX       LoadCommand = 0x24
	LC_VERSION_MIN_IPHONEOS     LoadCommand = 0x25
	LC_FUNCTION_STARTS          LoadCommand = 0x26
	LC_DYLD_ENVIRONMENT         LoadCommand = 0x27
	LC_MAIN                     LoadCommand = 0x80000028
	LC_DATA_IN_CODE             LoadCommand = 0x29
	LC_SOURCE_VERSION           LoadCommand = 0x2a
	LC_DYLIB_CODE_SIGN_DRS      LoadCommand = 0x2b
	LC_ENCRYPTION_INFO_64       LoadCommand = 0x2c
	LC_LINKER_OPTION            LoadCommand = 0x2d
	LC_LINKER_OPTIMIZATION_HINT LoadCommand = 0x2e
	LC_VERSION_MIN_TVOS         LoadCommand = 0x2f
	LC_VERSION_MIN_WATCHOS      LoadCommand = 0x30
)

const (
	N_STAB = 0xe0
	N_PEXT = 0x10
	N_TYPE = 0x0e
	N_EXT  = 0x01
)

const (
	N_UNDF = 0x0
	N_ABS  = 0x2
	N_SECT = 0xe
	N_PBUD = 0xc
	N_INDR = 0xa
)

type SymbolType uint8

type ReferenceType uint8

const REFERENCE_TYPE = 0x7 // for undefined

const (
	REFERENCE_FLAG_UNDEFINED_NON_LAZY         ReferenceType = 0
	REFERENCE_FLAG_UNDEFINED_LAZY             ReferenceType = 1
	REFERENCE_FLAG_DEFINED                    ReferenceType = 2
	REFERENCE_FLAG_PRIVATE_DEFINED            ReferenceType = 3
	REFERENCE_FLAG_PRIVATE_UNDEFINED_NON_LAZY ReferenceType = 4
	REFERENCE_FLAG_PRIVATE_UNDEFINED_LAZY     ReferenceType = 5
)

const REFERENCED_DYNAMICALLY = 0x0010 // for extern

const (
	N_NO_DEAD_STRIP   = 0x0020 // for .o
	N_DESC_DISCARDED  = 0x0020 // for non .o
	N_WEAK_REF        = 0x0040 // for undefined
	N_WEAK_DEF        = 0x0080 // for extern
	N_REF_TO_WEAK     = 0x0080 // for undefined
	N_ARM_THUMB_DEF   = 0x0008 // for arm
	N_SYMBOL_RESOLVER = 0x0100 // for .o
	N_ALT_ENTRY       = 0x0200
)

const (
	SELF_LIBRARY_ORDINAL   = 0x0
	MAX_LIBRARY_ORDINAL    = 0xfd
	DYNAMIC_LOOKUP_ORDINAL = 0xfe
	EXECUTABLE_ORDINAL     = 0xff
)
