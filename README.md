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

```
godsect -i example.ad -o out.go
```

File: out.go contains:

```
type Count struct {
	One   [80]byte     // offset 0x0000 (0) type C
	Two   [80][1]byte  // offset 0x0050 (80) type C
	Three [6][4]uint32 // offset 0x00a0 (160) type F
	Four  [8]byte      // offset 0x00b8 (184) type D
	Five  [4][2]uint16 // offset 0x00c0 (192) type H
}

const CountSize = 200

type Mydseg struct {
	Itema  [7][8]uint64 // offset 0x0000 (0) type D
	Itemb  [6][8]byte   // offset 0x0038 (56) type D
	Itemc  [5][8]byte   // offset 0x0068 (104) type D
	_      [8]byte      // offset 0x0098 (152)
	Itemd  [3][4]uint32 // offset 0x0098 (152) type F
	Iteme  [2][2]uint16 // offset 0x00a4 (164) type H
	Itemf  [1]byte      // offset 0x00a8 (168) type C
	Itemf0 [1]byte      // offset 0x00a9 (169) type C
	Itemg  [21][1]byte  // offset 0x00aa (170) type C
	_      [1]byte      // offset 0x00c0 (192)
	Last   [0][2]uint16 // offset 0x00c0 (192) type H
}

const MydsegSize = 192

type Pstruct struct {
	// item Ifaargs of type:D size:8 at offset 0 with 0 count skipped
	Ifaargs_xprefix           [4]uint32      // offset 0x0000 (0) type F
	Ifaargs_xid               [8]byte        // offset 0x0004 (4) type C
	Ifaargs_xplistlen         [2]byte        // offset 0x000c (12) type X
	Ifaargs_xversion          [1]byte        // offset 0x000e (14) type X
	Ifaargs_xrequest          [1]byte        // offset 0x000f (15) type X
	Ifaargs_xprodowner        [16]byte       // offset 0x0010 (16) type C
	Ifaargs_xprodname         [16]byte       // offset 0x0020 (32) type C
	Ifaargs_xprodvers         [8]byte        // offset 0x0030 (48) type C
	Ifaargs_xprodqual         [8]byte        // offset 0x0038 (56) type C
	Ifaargs_xprodid           [8]byte        // offset 0x0040 (64) type C
	Ifaargs_xdomain           [1]byte        // offset 0x0048 (72) type X
	Ifaargs_xscope            [1]byte        // offset 0x0049 (73) type X
	Ifaargs_xrsv0001          [1]byte        // offset 0x004a (74) type C
	Ifaargs_xflags            [1]byte        // offset 0x004b (75) type B
	Ifaargs_xprtoken_addr     [4]byte        // offset 0x004c (76) type A
	Ifaargs_xbegtime_addr     [4]uint32      // offset 0x0050 (80) type A
	Ifaargs_xdata_addr        [4]byte        // offset 0x0054 (84) type A
	Ifaargs_xformat           [1]byte        // offset 0x0058 (88) type X
	Ifaargs_xunauthserv       [1]byte        // offset 0x0059 (89) type X
	Ifaargs_xrsv0002          [2]byte        // offset 0x005a (90) type C
	Ifaargs_xcurrentdata_addr [4]byte        // offset 0x005c (92) type A
	Ifaargs_xenddata_addr     [4]uint32      // offset 0x0060 (96) type A
	Ifaargs_xendtime_addr     [4]byte        // offset 0x0064 (100) type A
	Callprm                   [8][8]byte     // offset 0x0068 (104) type D
	Coinfo                    [4][4]uint32   // offset 0x00a8 (168) type F
	Cfspp                     [256][4]uint32 // offset 0x00b8 (184) type F
	Crcpp                     [4]uint32      // offset 0x04b8 (1208) type F
	Edoiflags                 [1]byte        // offset 0x04bc (1212) type B
	// item Edoi of type:F size:4 at offset 1212 with 0 count skipped
	_                     [3]byte   // offset 0x04c0 (1216)
	Edoineededfeatureslen [4]uint32 // offset 0x04c0 (1216) type F
	Edoiprodvers          [2]byte   // offset 0x04c4 (1220) type C
	Edoiprodversrelmod    [6]byte   // offset 0x04c4 (1220) type C
	Edoiprodrel           [2]byte   // offset 0x04c6 (1222) type C
	Edoiprodmod           [2]byte   // offset 0x04c8 (1224) type C
	// item List1 of type:D size:8 at offset 1232 with 0 count skipped
	List1_xprefix           [4]uint32 // offset 0x04d0 (1232) type F
	List1_xid               [8]byte   // offset 0x04d4 (1236) type C
	List1_xplistlen         [2]byte   // offset 0x04dc (1244) type X
	List1_xversion          [1]byte   // offset 0x04de (1246) type X
	List1_xrequest          [1]byte   // offset 0x04df (1247) type X
	List1_xprodowner        [16]byte  // offset 0x04e0 (1248) type C
	List1_xprodname         [16]byte  // offset 0x04f0 (1264) type C
	List1_xprodvers         [8]byte   // offset 0x0500 (1280) type C
	List1_xprodqual         [8]byte   // offset 0x0508 (1288) type C
	List1_xprodid           [8]byte   // offset 0x0510 (1296) type C
	List1_xdomain           [1]byte   // offset 0x0518 (1304) type X
	List1_xscope            [1]byte   // offset 0x0519 (1305) type X
	List1_xrsv0001          [1]byte   // offset 0x051a (1306) type C
	List1_xflags            [1]byte   // offset 0x051b (1307) type B
	List1_xprtoken_addr     [4]byte   // offset 0x051c (1308) type A
	List1_xbegtime_addr     [4]uint32 // offset 0x0520 (1312) type A
	List1_xdata_addr        [4]byte   // offset 0x0524 (1316) type A
	List1_xformat           [1]byte   // offset 0x0528 (1320) type X
	List1_xunauthserv       [1]byte   // offset 0x0529 (1321) type X
	List1_xrsv0002          [2]byte   // offset 0x052a (1322) type C
	List1_xcurrentdata_addr [4]byte   // offset 0x052c (1324) type A
	List1_xenddata_addr     [4]uint32 // offset 0x0530 (1328) type A
	List1_xendtime_addr     [4]byte   // offset 0x0534 (1332) type A
}

const PstructSize = 1336

type Cca struct {
	Ccaversion   [2]uint16 // offset 0x0000 (0) type H
	_            [2]byte   // offset 0x0004 (4)
	Ccamsglength [4]byte   // offset 0x0004 (4) type F
	// item Ccamsgptrg of type:C size:8 at offset 8 with 0 count skipped
	Ccamsgptr [4]uint32 // offset 0x0008 (8) type A
	_         [8]byte   // offset 0x0014 (20)
	// item Ccastartver2 of type:C size:52 at offset 20 with 0 count skipped
	// item Ccaendver1 of type:C size:1 at offset 20 with 0 count skipped
	_ [4]byte // offset 0x0018 (24)
	// item Ccawtoparms of type:C size:36 at offset 24 with 0 count skipped
	// item Ccaroutcdelistg of type:C size:8 at offset 24 with 0 count skipped
	Ccaroutcdelist [4]uint32 // offset 0x0018 (24) type A
	_              [4]byte   // offset 0x0020 (32)
	// item Ccadesclistg of type:C size:8 at offset 32 with 0 count skipped
	Ccadesclist [4]uint32 // offset 0x0020 (32) type A
	_           [4]byte   // offset 0x0028 (40)
	// item Ccawmcsflags of type:B size:4 at offset 40 with 0 count skipped
	_           [4]byte   // offset 0x002c (44)
	Ccawtotoken [4]byte   // offset 0x002c (44) type F
	Ccamsgidptr [4]uint32 // offset 0x0030 (48) type A
	// item Ccamsgidptrg of type:C size:8 at offset 48 with 0 count skipped
	_ [8]byte // offset 0x003c (60)
	// item Ccadomparms of type:C size:12 at offset 60 with 0 count skipped
	Ccadomtoken [4]byte // offset 0x003c (60) type F
	// item Ccamsgidlistg of type:C size:8 at offset 64 with 0 count skipped
	Ccamsgidlist [4]uint32 // offset 0x0040 (64) type A
	_            [4]byte   // offset 0x0048 (72)
	// item Ccastartver3 of type:C size:40 at offset 72 with 0 count skipped
	// item Ccamodcartptrg of type:C size:8 at offset 72 with 0 count skipped
	Ccamodcartptr [4]uint32 // offset 0x0048 (72) type A
	// item Ccaendver2 of type:C size:1 at offset 72 with 0 count skipped
	_ [4]byte // offset 0x0050 (80)
	// item Ccamodconsoleidptrg of type:C size:8 at offset 80 with 0 count skipped
	Ccamodconsoleidptr [4]uint32  // offset 0x0050 (80) type A
	_                  [4]byte    // offset 0x0058 (88)
	Ccamsgcart         [8]byte    // offset 0x0058 (88) type C
	Ccamsgconsoleid    [4]byte    // offset 0x0060 (96) type C
	_                  [12]byte   // offset 0x0070 (112)
	Ccaendver3         [0][1]byte // offset 0x0070 (112) type C
}

const CcaSize = 112

type Rdarea struct {
	_     [4]byte  // offset 0x0004 (4)
	Payno [6]byte  // offset 0x0004 (4) type C
	Name  [20]byte // offset 0x000a (10) type C
	// item Date of type:C size:6 at offset 30 with 0 count skipped
	Day    [2]byte  // offset 0x001e (30) type C
	Month  [2]byte  // offset 0x0020 (32) type C
	Year   [2]byte  // offset 0x0022 (34) type C
	_      [10]byte // offset 0x002e (46)
	Gross  [8]byte  // offset 0x002e (46) type C
	Fedtax [8]byte  // offset 0x0036 (54) type C
}

const RdareaSize = 80

type Fields struct {
	Field [4][10]byte // offset 0x0000 (0) type C
	Area  [100]byte   // offset 0x0028 (40) type C
}

const FieldsSize = 140
```
