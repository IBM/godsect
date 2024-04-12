# godsect

A utility to create Go structure types from DSECT information in ADATA file created by HLASM.

# Example

HLASM assembly source: `example.asm`

```
TRY CSECT
TRY AMODE 64
TRY RMODE ANY
 SYSSTATE AMODE64=YES
 BR 14
 BPXYCCA
RDAREA   DSECT
         DS         CL4
PAYNO    DS         CL6
NAME     DS         CL20
DATE     DS         0CL6
DAY      DS         CL2
MONTH    DS         CL2
YEAR     DS         CL2
         DS         CL10
GROSS    DS         CL8
FEDTAX   DS         CL8
         DS         CL18
FIELDS   DSECT
FIELD    DS              4CL10
AREA     DS              CL100
*
COUNT    DSECT
ONE      DS CL80 One 80 byte field, len attr of 80
TWO      DS 80C  Eighty 1 byte fields, length attribute of 1
THREE    DS 6F   6 fullwords, length attribute of 4
FOUR     DS D    1 doubleword, length attribute of 8
FIVE     DS 4H   4 halfwords, length attribute of 2
*
MYDSEG   DSECT
ITEMA    DS    7D
ITEMB    DS    6D
ITEMC    DS    5D
         DS    4H
ITEMD    DS    3F
ITEME    DS    2H
ITEMF    DS    1C
ITEMF0   DS    C
ITEMG    DS    21C
LAST     DS    0H

PSTRUCT  DSECT
REGARGS  IFAUSAGE MF=(L,IFAARGS)
Callprm  DS    8D
COINFO   DS    4F                        Output area for EDOI
CFSPP    DS    256F                      PPACK Features Info
CRCPP    DS    F                         IFAEDSTA return code
STAPRM   IFAEDIDF DSECT=NO,LIST=YES,TITLE=YES,EDOI=YES,                *
               EDAAHDR=COND,EDAAE=COND
PLEN     EQU *-PSTRUCT

ADDR     DSECT
A1@      DS    2AD
A2       DS    A
DONE     DS    0H
LENGTH#  DS    F

LISTFORM IFAUSAGE MF=(L,LIST1)
*---------------------------------------------------------------*
         END
```

- Create ADATA file example.ad

```
/bin/as --gadata=example.ad -m goff -a=example.lst example.asm`
 Assembler Done No Statements Flagged
```

- Run godsect on ADATA file example.ad
  -s option to change @ in symbol name to _ptr_ and # to _len_ (if needed)

```
godsect -s "@/_ptr_;#/_len_" -i example.ad -o out.go
```

File: out.go contains:

```
type Pstruct struct {
	// item Ifaargs of type:D size:8 at offset 0 with 0 count skipped
	Ifaargs_xprefix           uint32      // offset 0x0000 (0) type F, size/count: 4/1
	Ifaargs_xid               [8]byte     // offset 0x0004 (4) type C, size/count: 8/1
	Ifaargs_xplistlen         [2]byte     // offset 0x000c (12) type X, size/count: 2/1
	Ifaargs_xversion          byte        // offset 0x000e (14) type X, size/count: 1/1
	Ifaargs_xrequest          byte        // offset 0x000f (15) type X, size/count: 1/1
	Ifaargs_xprodowner        [16]byte    // offset 0x0010 (16) type C, size/count: 16/1
	Ifaargs_xprodname         [16]byte    // offset 0x0020 (32) type C, size/count: 16/1
	Ifaargs_xprodvers         [8]byte     // offset 0x0030 (48) type C, size/count: 8/1
	Ifaargs_xprodqual         [8]byte     // offset 0x0038 (56) type C, size/count: 8/1
	Ifaargs_xprodid           [8]byte     // offset 0x0040 (64) type C, size/count: 8/1
	Ifaargs_xdomain           byte        // offset 0x0048 (72) type X, size/count: 1/1
	Ifaargs_xscope            byte        // offset 0x0049 (73) type X, size/count: 1/1
	Ifaargs_xrsv0001          byte        // offset 0x004a (74) type C, size/count: 1/1
	Ifaargs_xflags            byte        // offset 0x004b (75) type B, size/count: 1/1
	Ifaargs_xprtoken_addr     uint32      // offset 0x004c (76) type A, size/count: 4/1
	Ifaargs_xbegtime_addr     uint32      // offset 0x0050 (80) type A, size/count: 4/1
	Ifaargs_xdata_addr        uint32      // offset 0x0054 (84) type A, size/count: 4/1
	Ifaargs_xformat           byte        // offset 0x0058 (88) type X, size/count: 1/1
	Ifaargs_xunauthserv       byte        // offset 0x0059 (89) type X, size/count: 1/1
	Ifaargs_xrsv0002          [2]byte     // offset 0x005a (90) type C, size/count: 2/1
	Ifaargs_xcurrentdata_addr uint32      // offset 0x005c (92) type A, size/count: 4/1
	Ifaargs_xenddata_addr     uint32      // offset 0x0060 (96) type A, size/count: 4/1
	Ifaargs_xendtime_addr     uint32      // offset 0x0064 (100) type A, size/count: 4/1
	Callprm                   [8]uint64   // offset 0x0068 (104) type D, size/count: 8/8
	Coinfo                    [4]uint32   // offset 0x00a8 (168) type F, size/count: 4/4
	Cfspp                     [256]uint32 // offset 0x00b8 (184) type F, size/count: 4/256
	Crcpp                     uint32      // offset 0x04b8 (1208) type F, size/count: 4/1
	Edoiflags                 byte        // offset 0x04bc (1212) type B, size/count: 1/1
	// item Edoi of type:F size:4 at offset 1212 with count 0 skipped because origin overlaps previous member.
	_                     [3]byte // offset 0x04c0 (1216), filler size 3
	Edoineededfeatureslen uint32  // offset 0x04c0 (1216) type F, size/count: 4/1
	Edoiprodversrelmod    [6]byte // offset 0x04c4 (1220) type C, size/count: 6/1
	// item Edoiprodvers of type:C size:2 at offset 1220 with count 1 skipped because origin overlaps previous member.
	// item Edoiprodrel of type:C size:2 at offset 1222 with count 1 skipped because origin overlaps previous member.
	// item Edoiprodmod of type:C size:2 at offset 1224 with count 1 skipped because origin overlaps previous member.
}

const PstructSize = 1228

type Addr struct {
	A1_ptr_ [2]unsafe.Pointer // offset 0x0000 (0) type AD, size/count: 8/2
	A2      uint32            // offset 0x0010 (16) type A, size/count: 4/1
	// item Done of type:H size:2 at offset 20 with 0 count skipped
	Length_len_ uint32 // offset 0x0014 (20) type F, size/count: 4/1
	// item List1 of type:D size:8 at offset 24 with 0 count skipped
	List1_xprefix           uint32   // offset 0x0018 (24) type F, size/count: 4/1
	List1_xid               [8]byte  // offset 0x001c (28) type C, size/count: 8/1
	List1_xplistlen         [2]byte  // offset 0x0024 (36) type X, size/count: 2/1
	List1_xversion          byte     // offset 0x0026 (38) type X, size/count: 1/1
	List1_xrequest          byte     // offset 0x0027 (39) type X, size/count: 1/1
	List1_xprodowner        [16]byte // offset 0x0028 (40) type C, size/count: 16/1
	List1_xprodname         [16]byte // offset 0x0038 (56) type C, size/count: 16/1
	List1_xprodvers         [8]byte  // offset 0x0048 (72) type C, size/count: 8/1
	List1_xprodqual         [8]byte  // offset 0x0050 (80) type C, size/count: 8/1
	List1_xprodid           [8]byte  // offset 0x0058 (88) type C, size/count: 8/1
	List1_xdomain           byte     // offset 0x0060 (96) type X, size/count: 1/1
	List1_xscope            byte     // offset 0x0061 (97) type X, size/count: 1/1
	List1_xrsv0001          byte     // offset 0x0062 (98) type C, size/count: 1/1
	List1_xflags            byte     // offset 0x0063 (99) type B, size/count: 1/1
	List1_xprtoken_addr     uint32   // offset 0x0064 (100) type A, size/count: 4/1
	List1_xbegtime_addr     uint32   // offset 0x0068 (104) type A, size/count: 4/1
	List1_xdata_addr        uint32   // offset 0x006c (108) type A, size/count: 4/1
	List1_xformat           byte     // offset 0x0070 (112) type X, size/count: 1/1
	List1_xunauthserv       byte     // offset 0x0071 (113) type X, size/count: 1/1
	List1_xrsv0002          [2]byte  // offset 0x0072 (114) type C, size/count: 2/1
	List1_xcurrentdata_addr uint32   // offset 0x0074 (116) type A, size/count: 4/1
	List1_xenddata_addr     uint32   // offset 0x0078 (120) type A, size/count: 4/1
	List1_xendtime_addr     uint32   // offset 0x007c (124) type A, size/count: 4/1
}

const AddrSize = 128

type Cca struct {
	Ccaversion   uint16  // offset 0x0000 (0) type H, size/count: 2/1
	_            [2]byte // offset 0x0004 (4), filler size 2
	Ccamsglength uint32  // offset 0x0004 (4) type F, size/count: 4/1
	// item Ccamsgptrg of type:C size:8 at offset 8 with 0 count skipped
	Ccamsgptr uint32  // offset 0x0008 (8) type A, size/count: 4/1
	_         [8]byte // offset 0x0014 (20), filler size 8
	// item Ccastartver2 of type:C size:52 at offset 20 with 0 count skipped
	// item Ccaendver1 of type:C size:1 at offset 20 with 0 count skipped
	_ [4]byte // offset 0x0018 (24), filler size 4
	// item Ccawtoparms of type:C size:36 at offset 24 with 0 count skipped
	// item Ccaroutcdelistg of type:C size:8 at offset 24 with 0 count skipped
	Ccaroutcdelist uint32  // offset 0x0018 (24) type A, size/count: 4/1
	_              [4]byte // offset 0x0020 (32), filler size 4
	// item Ccadesclistg of type:C size:8 at offset 32 with 0 count skipped
	Ccadesclist uint32  // offset 0x0020 (32) type A, size/count: 4/1
	_           [4]byte // offset 0x0028 (40), filler size 4
	// item Ccawmcsflags of type:B size:4 at offset 40 with 0 count skipped
	_           [4]byte // offset 0x002c (44), filler size 4
	Ccawtotoken uint32  // offset 0x002c (44) type F, size/count: 4/1
	Ccamsgidptr uint32  // offset 0x0030 (48) type A, size/count: 4/1
	// item Ccamsgidptrg of type:C size:8 at offset 48 with count 0 skipped because origin overlaps previous member.
	_ [8]byte // offset 0x003c (60), filler size 8
	// item Ccadomparms of type:C size:12 at offset 60 with 0 count skipped
	Ccadomtoken uint32 // offset 0x003c (60) type F, size/count: 4/1
	// item Ccamsgidlistg of type:C size:8 at offset 64 with 0 count skipped
	Ccamsgidlist uint32  // offset 0x0040 (64) type A, size/count: 4/1
	_            [4]byte // offset 0x0048 (72), filler size 4
	// item Ccastartver3 of type:C size:40 at offset 72 with 0 count skipped
	// item Ccamodcartptrg of type:C size:8 at offset 72 with 0 count skipped
	Ccamodcartptr uint32 // offset 0x0048 (72) type A, size/count: 4/1
	// item Ccaendver2 of type:C size:1 at offset 72 with count 0 skipped because origin overlaps previous member.
	_ [4]byte // offset 0x0050 (80), filler size 4
	// item Ccamodconsoleidptrg of type:C size:8 at offset 80 with 0 count skipped
	Ccamodconsoleidptr uint32   // offset 0x0050 (80) type A, size/count: 4/1
	_                  [4]byte  // offset 0x0058 (88), filler size 4
	Ccamsgcart         [8]byte  // offset 0x0058 (88) type C, size/count: 8/1
	Ccamsgconsoleid    [4]byte  // offset 0x0060 (96) type C, size/count: 4/1
	_                  [12]byte // offset 0x0070 (112), filler size 12
	Ccaendver3         [0]byte  // offset 0x0070 (112) type C, size/count: 1/0
}

const CcaSize = 112

type Rdarea struct {
	_     [4]byte  // offset 0x0004 (4), filler size 4
	Payno [6]byte  // offset 0x0004 (4) type C, size/count: 6/1
	Name  [20]byte // offset 0x000a (10) type C, size/count: 20/1
	// item Date of type:C size:6 at offset 30 with 0 count skipped
	Day    [2]byte  // offset 0x001e (30) type C, size/count: 2/1
	Month  [2]byte  // offset 0x0020 (32) type C, size/count: 2/1
	Year   [2]byte  // offset 0x0022 (34) type C, size/count: 2/1
	_      [10]byte // offset 0x002e (46), filler size 10
	Gross  [8]byte  // offset 0x002e (46) type C, size/count: 8/1
	Fedtax [8]byte  // offset 0x0036 (54) type C, size/count: 8/1
}

const RdareaSize = 80

type Fields struct {
	Field [4][10]byte // offset 0x0000 (0) type C, size/count: 10/4
	Area  [100]byte   // offset 0x0028 (40) type C, size/count: 100/1
}

const FieldsSize = 140

type Count struct {
	One   [80]byte  // offset 0x0000 (0) type C, size/count: 80/1
	Two   [80]byte  // offset 0x0050 (80) type C, size/count: 1/80
	Three [6]uint32 // offset 0x00a0 (160) type F, size/count: 4/6
	Four  uint64    // offset 0x00b8 (184) type D, size/count: 8/1
	Five  [4]uint16 // offset 0x00c0 (192) type H, size/count: 2/4
}

const CountSize = 200

type Mydseg struct {
	Itema  [7]uint64 // offset 0x0000 (0) type D, size/count: 8/7
	Itemb  [6]uint64 // offset 0x0038 (56) type D, size/count: 8/6
	Itemc  [5]uint64 // offset 0x0068 (104) type D, size/count: 8/5
	_      [8]byte   // offset 0x0098 (152), filler size 8
	Itemd  [3]uint32 // offset 0x0098 (152) type F, size/count: 4/3
	Iteme  [2]uint16 // offset 0x00a4 (164) type H, size/count: 2/2
	Itemf  byte      // offset 0x00a8 (168) type C, size/count: 1/1
	Itemf0 byte      // offset 0x00a9 (169) type C, size/count: 1/1
	Itemg  [21]byte  // offset 0x00aa (170) type C, size/count: 1/21
	_      [1]byte   // offset 0x00c0 (192), filler size 1
	Last   [0]uint16 // offset 0x00c0 (192) type H, size/count: 2/0
}

const MydsegSize = 192

```
