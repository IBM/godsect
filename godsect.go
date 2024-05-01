package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
	"math/bits"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unsafe"
)

type RecordType struct {
	Typecode uint16
	Name     string
	Edition  byte
}

var recordTypes = map[uint16]RecordType{
	0x0000: {0x0000, "Job Idenfification", 1},
	0x0001: {0x0001, "ADATA Idenfification", 0},
	0x0002: {0x0002, "Compiilation Unit Start/End", 0},
	0x000a: {0x000a, "Output File information", 1},
	0x000b: {0x000b, "Option File information", 1},
	0x0010: {0x0010, "Options", 3},
	0x0020: {0x0020, "External Symbol Dictionary", 1},
	0x0030: {0x0030, "Source Analysis", 1},
	0x0032: {0x0032, "Source Error", 1},
	0x0034: {0x0034, "DC/DS", 1},
	0x0035: {0x0035, "DC Extension", 1},
	0x0036: {0x0036, "Machine Instruction", 1},
	0x0040: {0x0040, "Relocation Dictionary", 1},
	0x0042: {0x0042, "Symbol", 1},
	0x0044: {0x0044, "Symbol and Literal Cross Reference", 1},
	0x0045: {0x0045, "Register Cross Reference", 1},
	0x0060: {0x0060, "Macro and Copy Code Source Summary", 1},
	0x0062: {0x0062, "Macro and Copy Code Cross Reference", 1},
	0x0070: {0x0070, "User Data", 1},
	0x0080: {0x0080, "USING Map", 1},
	0x0090: {0x0090, "Assembly Statistics", 2},
}

var index int

type CommonHeaderData struct {
	LangCode     byte
	RecType      uint16
	Arch         byte
	Edition      byte
	DataLen      uint16
	LittleEndian bool
	Cont         bool
}
type SymbolInfo struct {
	Esdid         uint32
	Statement     uint32
	Location      uint32
	SymbolType    byte
	DupFactor     uint32
	TypeAttr      byte
	AsmType       string
	PgmType       uint32
	LenAttr       uint32
	IntAttr       uint16
	ScaleAttr     uint16
	SymbolFlags   byte
	SymNameOffset uint32
	Name          string
}

type Member struct {
	Name   string
	Offset uint32
	Type   string
	Size   uint32
	Dup    uint32
}
type Dsect struct {
	Esdid     uint32
	Name      string
	TotalSize uint32
	Mem       []Member
	Equs      []Equ
}
type Equ struct {
	Esdid uint32
	Name  string
	Value int32
}
type RegRepl struct {
	Re   *regexp.Regexp
	Repl string
}

type AdParse struct {
	In          *os.File
	Offset      uint64
	Inbytes     uint64
	Out         *os.File
	Buffer      []byte
	CHeader     CommonHeaderData
	Structs     map[uint32]Dsect
	Equs        []Equ
	HasGofmt    bool
	Nofmt       bool
	Verbose     bool
	Cmd         *exec.Cmd
	ReadHandle  *io.PipeReader
	WriteHandle *io.PipeWriter
	NameRegList []RegRepl
}

const BUFSIZE = 0x8000

func IsAlign(offset uint32, size uint32) bool {
	if bits.OnesCount32(size) > 1 {
		return false
	}
	if ((size - 1) & offset) == 0 {
		return true
	}
	return false
}

func (ad *AdParse) VerbosePrintf(format string, v ...any) {
	if ad.Verbose {
		fmt.Fprintf(os.Stderr, format, v...)
	}
}

func (ad *AdParse) ParseChangeExpr(s string) (lasterror error) {
	STokenize := func(s string, d byte) []string {
		var tokens []string
		var currentToken string
		escaped := false
		for _, char := range s {
			if byte(char) == '\\' && !escaped {
				escaped = true
			} else if byte(char) == d && !escaped {
				tokens = append(tokens, currentToken)
				currentToken = ""
			} else {
				currentToken += string(char)
				escaped = false
			}
		}
		if currentToken != "" {
			tokens = append(tokens, currentToken)
		}
		return tokens
	}

	tokens := STokenize(s, ';')
	for _, token := range tokens {
		res := STokenize(token, '/')
		if len(res) != 2 {
			lasterror = fmt.Errorf("change expression \"%s\" is not valid, it should be of the form \"from.../to...\"\n", token)
		} else {
			re, err := regexp.Compile(res[0])
			if re != nil {
				var r RegRepl
				r.Re = re
				r.Repl = res[1]
				ad.NameRegList = append(ad.NameRegList, r)
			} else {
				lasterror = fmt.Errorf("Regex changing \"%s\" to \"%s\" is not valid, error \"%s\"\n", res[0], res[1], err)
			}
		}
	}
	return
}

func (ad *AdParse) Init(in, out *os.File, namechangeexpr string, nofmt, verbose bool) (err error) {
	ad.In = in
	ad.Out = out
	ad.Buffer = make([]byte, BUFSIZE)
	ad.Structs = make(map[uint32]Dsect)
	ad.Verbose = verbose
	ad.Nofmt = nofmt
	err = ad.ParseChangeExpr(namechangeexpr)
	if err != nil {
		return
	}
	if !nofmt {
		cmd := exec.Command("/bin/sh", "-c", "type gofmt >/dev/null 2>&1")
		if err := cmd.Run(); err == nil {
			ad.HasGofmt = true
			ad.Cmd = exec.Command("gofmt")
			ad.ReadHandle, ad.WriteHandle = io.Pipe()
			ad.Cmd.Stdout = ad.Out
			ad.Cmd.Stdin = ad.ReadHandle
			ad.Cmd.Start()
		}
	}
	return
}

func (ad *AdParse) OutPrintf(format string, v ...any) {
	s := fmt.Sprintf(format, v...)
	if ad.HasGofmt {
		ad.WriteHandle.Write([]byte(s))
	} else {
		ad.Out.WriteString(s)
	}
}

func (ad *AdParse) Term() {
	if ad.HasGofmt {
		ad.WriteHandle.Close()
		ad.Cmd.Wait()
		ad.ReadHandle.Close()
	}
	ad.In.Close()
	ad.Out.Close()
}

func (ad *AdParse) FillBuf(size int) (err error) {
	if size > BUFSIZE {
		log.Fatalf("Record body size %d > %d\n", size, BUFSIZE)
	}
	n, er := ad.In.Read(ad.Buffer[:size])
	if n != size {
		if er == nil {
			err = fmt.Errorf("Short record, expect %d received %d", size, n)
		} else {
			err = er
		}
	}
	return
}

func (ad *AdParse) CommonHeader() (err error) {
	er := ad.FillBuf(12)
	if er != nil {
		err = er
		return
	}
	ad.CHeader.LangCode = ad.Buffer[0]
	ad.CHeader.RecType = (uint16(ad.Buffer[1]) << 8) + uint16(ad.Buffer[2])
	ad.CHeader.Arch = ad.Buffer[3]
	ad.CHeader.Edition = ad.Buffer[5]
	ad.CHeader.LittleEndian = ((ad.Buffer[4] & 0x02) == 0x02)
	ad.CHeader.Cont = ((ad.Buffer[4] & 0x01) == 0x01)
	if ad.CHeader.LittleEndian {
		ad.CHeader.DataLen = (uint16(ad.Buffer[11]) << 8) + uint16(ad.Buffer[10])
	} else {
		ad.CHeader.DataLen = (uint16(ad.Buffer[10]) << 8) + uint16(ad.Buffer[11])
	}
	if ad.CHeader.LangCode != 16 {
		err = fmt.Errorf("Language code %d != 16", ad.CHeader.LangCode)
		return
	}
	if ad.CHeader.Arch != 3 {
		err = fmt.Errorf("Archtecture level %d != 3", ad.CHeader.Arch)
		return
	}
	_, ok := recordTypes[ad.CHeader.RecType]
	if !ok {
		err = fmt.Errorf("Record type 0x%04x not reconized", ad.CHeader.RecType)
		return
	}
	return
}

func (ad *AdParse) Parse() (err error) {
	err = ad.CommonHeader()
	for err == nil {
		ad.VerbosePrintf("Header %+v \n", ad.CHeader)
		ad.VerbosePrintf("Record Type %+v\n", recordTypes[ad.CHeader.RecType])
		er := ad.FillBuf(int(ad.CHeader.DataLen))
		if er != nil {
			err = err
			return
		}
		ad.VerbosePrintf("Record Data for type %04x\n", ad.CHeader.RecType)
		ad.DiagDump(uintptr(unsafe.Pointer(&ad.Buffer[0])), uintptr(ad.CHeader.DataLen))
		switch ad.CHeader.RecType {
		case 0x0020:
			if ad.Buffer[0] == 0xff {
				dsectsize := binary.BigEndian.Uint32(ad.Buffer[20:24])
				namelen := binary.BigEndian.Uint32(ad.Buffer[40:44])
				esdid := binary.BigEndian.Uint32(ad.Buffer[4:8])
				var dsectname string
				if namelen > 0 {
					dsectname = string(e2a(ad.Buffer[52 : 52+namelen]))
				} else {
					dsectname = fmt.Sprintf("_unname_%d", index)
					index++
				}
				ds, ok := ad.Structs[esdid]
				if ok {
					ds.Esdid = esdid
					ds.Name = dsectname
					ds.TotalSize = dsectsize
					ad.Structs[esdid] = ds
				} else {
					var ds1 Dsect
					ds1.Esdid = esdid
					ds1.Name = dsectname
					ds1.TotalSize = dsectsize
					ad.Structs[esdid] = ds1
				}
				ad.VerbosePrintf("// DSECT %s, %+v\n", dsectname, ds)
			}

		case 0x0042:
			var sym SymbolInfo
			sym.Esdid = binary.BigEndian.Uint32(ad.Buffer[0:4])
			sym.Statement = binary.BigEndian.Uint32(ad.Buffer[4:8])
			sym.Location = binary.BigEndian.Uint32(ad.Buffer[8:12])
			sym.SymbolType = ad.Buffer[12]
			sym.DupFactor = binary.BigEndian.Uint32(ad.Buffer[13:17])
			sym.TypeAttr = ad.Buffer[17]
			sym.AsmType = string(e2a(ad.Buffer[18:22]))
			sym.PgmType = binary.BigEndian.Uint32(ad.Buffer[22:26])
			sym.LenAttr = binary.BigEndian.Uint32(ad.Buffer[26:30])
			sym.IntAttr = binary.BigEndian.Uint16(ad.Buffer[30:32])
			sym.ScaleAttr = binary.BigEndian.Uint16(ad.Buffer[32:34])
			sym.SymbolFlags = ad.Buffer[34]
			sym.SymNameOffset = binary.BigEndian.Uint32(ad.Buffer[42:46])
			namelen := binary.BigEndian.Uint32(ad.Buffer[46:50])
			sym.Name = string(e2a(ad.Buffer[50 : 50+namelen]))
			if sym.SymbolType == 0x02 {
				ds, ok := ad.Structs[sym.Esdid]
				if ok {
					ds.Esdid = sym.Esdid
					ds.Name = sym.Name
					ad.Structs[sym.Esdid] = ds
				} else {
					var ds1 Dsect
					ds1.Esdid = sym.Esdid
					ds1.Name = sym.Name
					ad.Structs[sym.Esdid] = ds1
				}
				ad.VerbosePrintf("// Dsect head %+v\n", sym)
			} else if sym.SymbolType == 0x0d {
				var m Member
				m.Name = sym.Name
				m.Offset = sym.Location
				m.Type = strings.TrimSpace(sym.AsmType)
				m.Size = sym.LenAttr
				m.Dup = sym.DupFactor

				ds, ok := ad.Structs[sym.Esdid]
				if ok {
					ds.Mem = append(ds.Mem, m)
					ad.Structs[sym.Esdid] = ds
				} else {
					var ds1 Dsect
					ds1.Mem = append(ds1.Mem, m)
					ad.Structs[sym.Esdid] = ds1
				}
				ad.VerbosePrintf("// Dsect member %+v Esdid %d\n", m, sym.Esdid)
			} else if sym.SymbolType == 0x0c {
				ad.VerbosePrintf("// Sym %+v\n", sym)
				ad.VerbosePrintf("EQU\n")
				ad.VerbosePrintf("Name %s\n", sym.Name)
				ad.VerbosePrintf("Offset %d 0x%x\n", int32(sym.Location), sym.Location)
				ad.VerbosePrintf("Type %s\n", strings.TrimSpace(sym.AsmType))
				ad.VerbosePrintf("Len attr %d\n", sym.LenAttr)
				ad.VerbosePrintf("Dup factor %d\n", sym.DupFactor)
				var e Equ
				e.Esdid = sym.Esdid
				e.Name = sym.Name
				e.Value = int32(sym.Location)
				ds, ok := ad.Structs[sym.Esdid]
				if ok {
					ds.Equs = append(ds.Equs, e)
					ad.Structs[sym.Esdid] = ds
				} else {
					ad.Equs = append(ad.Equs, e)
				}
			} else {
				ad.VerbosePrintf("// Sym %+v\n", sym)
			}

		}

		err = ad.CommonHeader()
	}
	if err != io.EOF {
		ad.VerbosePrintf("Read header Error %v \n", err)
	}
	return
}

func (ad *AdParse) PrintGoStructs() {
	for _, v := range ad.Structs {
		if len(v.Name) > 0 {
			Name := ad.ToVarName(v.Name)
			ad.OutPrintf("type %s struct {\n", Name)
			sort.Slice(v.Mem, func(i, j int) bool {
				return v.Mem[i].Offset < v.Mem[j].Offset
			})
			cursor := uint32(0)
			for i, x := range v.Mem {
				XName := ad.ToVarName(x.Name)
				gap := int(x.Offset) - int(cursor)
				if gap > 0 {
					ad.OutPrintf("  _ [%d]byte // offset 0x%04x (%d), filler size %d\n", gap, x.Offset, x.Offset, gap)
					cursor += uint32(gap)
				} else if gap < 0 {
					ad.OutPrintf(" // item %s of type:%s size:%d at offset %d with count %d skipped because origin overlaps previous member.\n", XName, x.Type, x.Size, x.Offset, x.Dup)
					continue
				}
				XType := TypeNameInStruct(x.Offset, x.Size, x.Dup, x.Type)
				if x.Dup == 0 && ((i + 1) < len(v.Mem)) {
					// Zero length array is allowed for the last element of a structure
					ad.OutPrintf(" // item %s of type:%s size:%d at offset %d with 0 count skipped\n", XName, x.Type, x.Size, x.Offset)
				} else {
					ad.OutPrintf("  %s %s // offset 0x%04x (%d) type %s, size/count: %d/%d\n", XName, XType, x.Offset, x.Offset, x.Type, x.Size, x.Dup)
					cursor += (x.Size * x.Dup)
				}
			}
			ad.OutPrintf("}\nconst %sSize = %d\n\n", Name, v.TotalSize)
			if len(v.Equs) > 0 {
				ad.OutPrintf("const (\n")
				for _, e := range v.Equs {
					if len(e.Name) > 0 {
						Name := ad.ToVarName(e.Name)
						ad.OutPrintf(" %s = %d\n", Name, e.Value)
					}
				}
				ad.OutPrintf(")\n")
			}
		}
	}
	if len(ad.Equs) > 0 {
		ad.OutPrintf("const (\n")
		for _, e := range ad.Equs {
			if len(e.Name) > 0 {
				Name := ad.ToVarName(e.Name)
				ad.OutPrintf(" %s = %d\n", Name, e.Value)
			}
		}
		ad.OutPrintf(")\n")
	}
}

func TypeNameInStruct(offset uint32, typesize uint32, dupfactor uint32, typecode string) (result string) {
	var name string
	if typesize > 1 {
		name = fmt.Sprintf("[%d]byte", typesize)
	} else {
		name = "byte"
	}
	if IsAlign(offset, typesize) {
		switch typesize {
		case 2:
			switch typecode {
			case "H":
				name = "uint16"
			case "y":
				name = "uint16"
			}
		case 4:
			switch typecode {
			case "F":
				name = "uint32"
			case "A":
				name = "uint32"
			}
		case 8:
			switch typecode {
			case "D":
				name = "uint64"
			case "AD":
				name = "unsafe.Pointer"
			}
		}
	}
	// fall thru
	if dupfactor == 1 {
		result = fmt.Sprintf("%s", name)
	} else {
		result = fmt.Sprintf("[%d]%s", dupfactor, name)
	}
	return
}

var atbl [256]byte = [256]byte{
	46, 46, 46, 46, 46, 46, 46, 46, 46,
	46, 46, 46, 46, 46, 46, 46, 46,
	46, 46, 46, 46, 46, 46, 46, 46,
	46, 46, 46, 46, 46, 46, 46, 32,
	33, 34, 35, 36, 37, 38, 39, 40,
	41, 42, 43, 44, 45, 46, 47, 48,
	49, 50, 51, 52, 53, 54, 55, 56,
	57, 58, 59, 60, 61, 62, 63, 64,
	65, 66, 67, 68, 69, 70, 71, 72,
	73, 74, 75, 76, 77, 78, 79, 80,
	81, 82, 83, 84, 85, 86, 87, 88,
	89, 90, 91, 92, 93, 94, 95, 96,
	97, 98, 99, 100, 101, 102, 103, 104,
	105, 106, 107, 108, 109, 110, 111, 112,
	113, 114, 115, 116, 117, 118, 119, 120,
	121, 122, 123, 124, 125, 126, 46, 46,
	46, 46, 46, 46, 46, 46, 46, 46,
	46, 46, 46, 46, 46, 46, 46, 46,
	46, 46, 46, 46, 46, 46, 46, 46,
	46, 46, 46, 46, 46, 46, 46, 46,
	46, 46, 46, 46, 46, 46, 46, 46,
	46, 46, 46, 46, 46, 46, 46, 46,
	46, 46, 46, 46, 46, 46, 46, 46,
	46, 46, 46, 46, 46, 46, 46, 46,
	46, 46, 46, 46, 46, 46, 46, 46,
	46, 46, 46, 46, 46, 46, 46, 46,
	46, 46, 46, 46, 46, 46, 46, 46,
	46, 46, 46, 46, 46, 46, 46, 46,
	46, 46, 46, 46, 46, 46, 46, 46,
	46, 46, 46, 46, 46, 46, 46, 46,
	46, 46, 46, 46, 46, 46, 46, 46,
	46, 46, 46, 46, 46, 46, 46,
}

var etbl [256]byte = [256]byte{
	46, 46, 46, 46, 46, 46, 46, 46,
	46, 46, 46, 46, 46, 46, 46, 46,
	46, 46, 46, 46, 46, 46, 46, 46,
	46, 46, 46, 46, 46, 46, 46, 46,
	46, 46, 46, 46, 46, 46, 46, 46,
	46, 46, 46, 46, 46, 46, 46, 46,
	46, 46, 46, 46, 46, 46, 46, 46,
	46, 46, 46, 46, 46, 46, 46, 46,
	32, 46, 46, 46, 46, 46, 46, 46,
	46, 46, 46, 46, 60, 40, 43, 124,
	38, 46, 46, 46, 46, 46, 46, 46,
	46, 46, 33, 36, 42, 41, 59, 94,
	45, 47, 46, 46, 46, 46, 46, 46,
	46, 46, 46, 44, 37, 95, 62, 63,
	46, 46, 46, 46, 46, 46, 46, 46,
	46, 96, 58, 35, 64, 39, 61, 34,
	46, 97, 98, 99, 100, 101, 102, 103,
	104, 105, 46, 46, 46, 46, 46, 46,
	46, 106, 107, 108, 109, 110, 111, 112,
	113, 114, 46, 46, 46, 46, 46, 46,
	46, 126, 115, 116, 117, 118, 119, 120,
	121, 122, 46, 46, 46, 91, 46, 46,
	46, 46, 46, 46, 46, 46, 46, 46,
	46, 46, 46, 46, 46, 93, 46, 46,
	123, 65, 66, 67, 68, 69, 70, 71,
	72, 73, 46, 46, 46, 46, 46, 46,
	125, 74, 75, 76, 77, 78, 79, 80,
	81, 82, 46, 46, 46, 46, 46, 46,
	92, 46, 83, 84, 85, 86, 87, 88,
	89, 90, 46, 46, 46, 46, 46, 46,
	48, 49, 50, 51, 52, 53, 54, 55,
	56, 57, 46, 46, 46, 46, 46, 46,
}

var hexchar [16]byte = [16]byte{
	'0', '1', '2', '3',
	'4', '5', '6', '7',
	'8', '9', 'a', 'b',
	'c', 'd', 'e', 'f',
}

func (ad *AdParse) DiagDump(ptr uintptr, size uintptr) {
	var line [90]byte
	var i uintptr
	for size > 0 {
		lbl := ptr
		bits := (unsafe.Sizeof(lbl) * 8)
		for bits > 0 {
			if 0 == (0x0f & (lbl >> (bits - 4))) {
				bits -= 4
			} else {
				break
			}
		}
		for bits > 0 {
			line[i] = hexchar[0x0f&(lbl>>(bits-4))]
			i++
			bits -= 4
		}
		line[i] = ':'
		i++
		line[i] = ' '
		i++

		ascptr := ptr
		ascsize := size
		ebcptr := ptr
		ebcsize := size

		fmt1 := 0
		fmt2 := 16

		for ; fmt1 < fmt2; fmt1++ {
			if size > 0 {
				b := *(*byte)(unsafe.Pointer(ptr))
				line[i] = hexchar[0x0f&(b>>4)]
				i++
				line[i] = hexchar[0x0f&(b)]
				i++
				size--
				ptr++
			} else {
				line[i] = ' '
				i++
				line[i] = ' '
				i++
			}
			if fmt1 > 0 && ((fmt1+1)%4 == 0) {
				line[i] = ' '
				i++
			}
		}
		for fmt1 = 0; fmt1 < fmt2; fmt1++ {
			if ascsize > 0 {
				b := *(*byte)(unsafe.Pointer(ascptr))
				line[i] = atbl[0xff&b]
				ascsize--
				ascptr++
			} else {
				line[i] = ' '
			}
			i++
		}
		line[i] = ' '
		i++
		for fmt1 = 0; fmt1 < fmt2; fmt1++ {
			if ebcsize > 0 {
				b := *(*byte)(unsafe.Pointer(ebcptr))
				line[i] = etbl[0xff&b]
				ebcsize--
				ebcptr++
			} else {
				line[i] = ' '
			}
			i++
		}
		line[i] = 0
		ad.VerbosePrintf("\t%s\n", string(line[0:i]))
		i = 0
	}
}

var e2atable = []byte{
	/* 00 */ 0x00, 0x01, 0x02, 0x03, 0x9c, 0x09, 0x86, 0x7f,
	/* 08 */ 0x97, 0x8d, 0x8e, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f,
	/* 10 */ 0x10, 0x11, 0x12, 0x13, 0x9d, 0x0a, 0x08, 0x87,
	/* 18 */ 0x18, 0x19, 0x92, 0x8f, 0x1c, 0x1d, 0x1e, 0x1f,
	/* 20 */ 0x80, 0x81, 0x82, 0x83, 0x84, 0x85, 0x17, 0x1b,
	/* 28 */ 0x88, 0x89, 0x8a, 0x8b, 0x8c, 0x05, 0x06, 0x07,
	/* 30 */ 0x90, 0x91, 0x16, 0x93, 0x94, 0x95, 0x96, 0x04,
	/* 38 */ 0x98, 0x99, 0x9a, 0x9b, 0x14, 0x15, 0x9e, 0x1a,
	/* 40 */ 0x20, 0xa0, 0xe2, 0xe4, 0xe0, 0xe1, 0xe3, 0xe5,
	/* 48 */ 0xe7, 0xf1, 0xa2, 0x2e, 0x3c, 0x28, 0x2b, 0x7c,
	/* 50 */ 0x26, 0xe9, 0xea, 0xeb, 0xe8, 0xed, 0xee, 0xef,
	/* 58 */ 0xec, 0xdf, 0x21, 0x24, 0x2a, 0x29, 0x3b, 0x5e,
	/* 60 */ 0x2d, 0x2f, 0xc2, 0xc4, 0xc0, 0xc1, 0xc3, 0xc5,
	/* 68 */ 0xc7, 0xd1, 0xa6, 0x2c, 0x25, 0x5f, 0x3e, 0x3f,
	/* 70 */ 0xf8, 0xc9, 0xca, 0xcb, 0xc8, 0xcd, 0xce, 0xcf,
	/* 78 */ 0xcc, 0x60, 0x3a, 0x23, 0x40, 0x27, 0x3d, 0x22,
	/* 80 */ 0xd8, 0x61, 0x62, 0x63, 0x64, 0x65, 0x66, 0x67,
	/* 88 */ 0x68, 0x69, 0xab, 0xbb, 0xf0, 0xfd, 0xfe, 0xb1,
	/* 90 */ 0xb0, 0x6a, 0x6b, 0x6c, 0x6d, 0x6e, 0x6f, 0x70,
	/* 98 */ 0x71, 0x72, 0xaa, 0xba, 0xe6, 0xb8, 0xc6, 0xa4,
	/* a0 */ 0xb5, 0x7e, 0x73, 0x74, 0x75, 0x76, 0x77, 0x78,
	/* a8 */ 0x79, 0x7a, 0xa1, 0xbf, 0xd0, 0x5b, 0xde, 0xae,
	/* b0 */ 0xac, 0xa3, 0xa5, 0xb7, 0xa9, 0xa7, 0xb6, 0xbc,
	/* b8 */ 0xbd, 0xbe, 0xdd, 0xa8, 0xaf, 0x5d, 0xb4, 0xd7,
	/* c0 */ 0x7b, 0x41, 0x42, 0x43, 0x44, 0x45, 0x46, 0x47,
	/* c8 */ 0x48, 0x49, 0xad, 0xf4, 0xf6, 0xf2, 0xf3, 0xf5,
	/* d0 */ 0x7d, 0x4a, 0x4b, 0x4c, 0x4d, 0x4e, 0x4f, 0x50,
	/* d8 */ 0x51, 0x52, 0xb9, 0xfb, 0xfc, 0xf9, 0xfa, 0xff,
	/* e0 */ 0x5c, 0xf7, 0x53, 0x54, 0x55, 0x56, 0x57, 0x58,
	/* e8 */ 0x59, 0x5a, 0xb2, 0xd4, 0xd6, 0xd2, 0xd3, 0xd5,
	/* f0 */ 0x30, 0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37,
	/* f8 */ 0x38, 0x39, 0xb3, 0xdb, 0xdc, 0xd9, 0xda, 0x9f,
}

func e2a(e []byte) (a []byte) {
	a = make([]byte, len(e))
	for i := range a {
		a[i] = e2atable[e[i]]
	}
	return
}

func (ad *AdParse) ToVarName(input string) string {
	for _, r := range ad.NameRegList {
		input = (r.Re).ReplaceAllString(input, (r.Repl))
	}
	input = strings.TrimSpace(input)
	input = strings.ReplaceAll(input, " ", "_")
	input = strings.ToUpper(input[:1]) + strings.ToLower(input[1:])
	isValidRune := func(r rune) bool {
		return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
	}
	var validName strings.Builder
	for _, r := range input {
		if isValidRune(r) {
			validName.WriteRune(r)
		} else {
			validName.WriteString(fmt.Sprintf("_x%02X", r))
		}
	}
	return validName.String()
}

func main() {
	var output, input string
	var in, out *os.File
	var err error
	var nofmt bool
	var namechangeexpr string
	var verbose bool
	flag.StringVar(&output, "o", "-", "output file name")
	flag.StringVar(&input, "i", "-", "input file name")
	flag.StringVar(&namechangeexpr, "s", "", "a series of regexs separated by ';' to change characters in symbol name. e.g. \"@/_ptr_;$/_D_/;#/_size\"")
	flag.BoolVar(&verbose, "v", false, "verbose with lots of diagnostic messages")
	flag.BoolVar(&nofmt, "n", false, "dont format with gofmt")
	flag.Parse()
	if input != "-" {
		in, err = os.Open(input)
		if err != nil {
			log.Fatalf("Cannot open %s for read %v\n", input, err)
		}
	} else {
		in = os.Stdin
	}
	defer in.Close()
	if output != "-" {
		out, err = os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			log.Fatalf("Cannot optn %s for read %v\n", output, err)
		}
	} else {
		out = os.Stdout
	}
	var adp AdParse
	err = adp.Init(in, out, namechangeexpr, nofmt, verbose)
	if err != nil {
		log.Fatalf("Init error: %v\n", err)
	}
	defer adp.Term()
	err = adp.Parse()
	if err != nil && err != io.EOF {
		log.Fatalf("Parsing file %v\n", err)
	}
	adp.PrintGoStructs()
}
