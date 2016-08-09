

# vm
`import "github.com/db47h/ngaro/vm"`

* [Overview](#pkg-overview)
* [Index](#pkg-index)
* [Examples](#pkg-examples)

## <a name="pkg-overview">Overview</a>
Package vm implements the Ngaro VM.

Please visit <a href="http://forthworks.com/retro/">http://forthworks.com/retro/</a> to get you started about the Retro
language and the Ngaro Virtual Machine.

The main purpose of this implementation is to allow communication between
Retro programs and Go programs via custom I/O handlers (i.e. scripting Go
programs in Retro). The package examples demonstrate various use cases. For
more details on I/O handling in the Ngaro VM, please refer to
<a href="http://retroforth.org/docs/The_Ngaro_Virtual_Machine.html">http://retroforth.org/docs/The_Ngaro_Virtual_Machine.html</a>.

This implementation passes all tests from the retro-language test suite and
its performance when running tests/core.rx is slightly better than with the
reference implementations:


	1.20s for this implementation, compiled with Go 1.7rc6.
	1.30s for the reference Go implementation, compiled with Go 1.7rc6
	2.22s for the reference C implementation, compiled with gcc-5.4 -O3 -fomit-frame-pointer

For all intents and purposes, the VM behaves according to the specification.
With one exception: if you venture into hacking the VM code itself, be aware
that for performance reasons, the PC (aka. Instruction Pointer) is not
incremented in a single place, rather each opcode deals with the PC as
needed. This should be of no concern to any other users, even with custom I/O
handlers. Should you find that the VM does not behave according to the spec,
please file a bug report.

There's a caveat common to all Ngaro implementations: use of IN, OUT and WAIT
from the listener (the Retro interactive prompt) will not work as expected.
This is because the listener uses the same mechanism to read user input and
write to the terminal and will clear port 0 before you get a chance to
read/clear response values. This is of particular importance for users of
custom IO handlers. To work around this issue, a synchronous OUT-WAIT-IN IO
sequence must be compiled in a word, so that it will run atomically without
interference from the listener. For example, to read VM capabilities, you can
do this:


	( io sends value n to port p, does a wait and puts response back on the stack )
	: io ( np-n ) dup push out 0 0 out wait pop in ;
	
	-1 5 io putn

should give you the size of the image.

Regarding I/O, reading console width and height will only work if the
io.Writer set as output with vm.Output implements the Fd method. So this will
only work if the output is os.Stdout or a pty (and NOT wrapped in a
bufio.Writer).




## <a name="pkg-index">Index</a>
* [type Cell](#Cell)
* [type Image](#Image)
  * [func Load(fileName string, capacity int) (Image, error)](#Load)
  * [func (i Image) DecodeString(start Cell) string](#Image.DecodeString)
  * [func (i Image) Disassemble(pc int) (next int, disasm string)](#Image.Disassemble)
  * [func (i Image) EncodeString(start Cell, s string)](#Image.EncodeString)
  * [func (i Image) Save(fileName string, shrink bool) error](#Image.Save)
* [type InHandler](#InHandler)
* [type Instance](#Instance)
  * [func New(image Image, imageFile string, opts ...Option) (*Instance, error)](#New)
  * [func (i *Instance) Address() []Cell](#Instance.Address)
  * [func (i *Instance) Data() []Cell](#Instance.Data)
  * [func (i *Instance) Drop(v Cell)](#Instance.Drop)
  * [func (i *Instance) Dump(w io.Writer) error](#Instance.Dump)
  * [func (i *Instance) In(port Cell) error](#Instance.In)
  * [func (i *Instance) InstructionCount() int64](#Instance.InstructionCount)
  * [func (i *Instance) Out(v, port Cell) error](#Instance.Out)
  * [func (i *Instance) Pop() Cell](#Instance.Pop)
  * [func (i *Instance) Push(v Cell)](#Instance.Push)
  * [func (i *Instance) PushInput(r io.Reader)](#Instance.PushInput)
  * [func (i *Instance) Rpop() Cell](#Instance.Rpop)
  * [func (i *Instance) Rpush(v Cell)](#Instance.Rpush)
  * [func (i *Instance) Run() (err error)](#Instance.Run)
  * [func (i *Instance) SetOptions(opts ...Option) error](#Instance.SetOptions)
  * [func (i *Instance) Tos() Cell](#Instance.Tos)
  * [func (i *Instance) Wait(v, port Cell) error](#Instance.Wait)
  * [func (i *Instance) WaitReply(v, port Cell)](#Instance.WaitReply)
* [type Option](#Option)
  * [func AddressSize(size int) Option](#AddressSize)
  * [func BindInHandler(port Cell, handler InHandler) Option](#BindInHandler)
  * [func BindOutHandler(port Cell, handler OutHandler) Option](#BindOutHandler)
  * [func BindWaitHandler(port Cell, handler WaitHandler) Option](#BindWaitHandler)
  * [func DataSize(size int) Option](#DataSize)
  * [func Input(r io.Reader) Option](#Input)
  * [func Output(t Terminal) Option](#Output)
  * [func Shrink(shrink bool) Option](#Shrink)
* [type OutHandler](#OutHandler)
* [type Terminal](#Terminal)
  * [func NewVT100Terminal(w io.Writer, flush func() error, size func() (width int, height int)) Terminal](#NewVT100Terminal)
* [type WaitHandler](#WaitHandler)

#### <a name="pkg-examples">Examples</a>
* [BindOutHandler](#example_BindOutHandler)
* [BindWaitHandler](#example_BindWaitHandler)
* [BindWaitHandler (Async)](#example_BindWaitHandler_async)
* [Image.Disassemble](#example_Image_Disassemble)
* [Instance.Run](#example_Instance_Run)

#### <a name="pkg-files">Package files</a>
[core.go](/src/github.com/db47h/ngaro/vm/core.go) [doc.go](/src/github.com/db47h/ngaro/vm/doc.go) [image.go](/src/github.com/db47h/ngaro/vm/image.go) [io.go](/src/github.com/db47h/ngaro/vm/io.go) [io_helpers.go](/src/github.com/db47h/ngaro/vm/io_helpers.go) [opcode_string.go](/src/github.com/db47h/ngaro/vm/opcode_string.go) [vm.go](/src/github.com/db47h/ngaro/vm/vm.go) 






## <a name="Cell">type</a> [Cell](/src/target/vm.go?s=776:791#L16)
``` go
type Cell int32
```
Cell is the raw type stored in a memory location.


``` go
const (
    OpNop Cell = iota
    OpLit
    OpDup
    OpDrop
    OpSwap
    OpPush
    OpPop
    OpLoop
    OpJump
    OpReturn
    OpGtJump
    OpLtJump
    OpNeJump
    OpEqJump
    OpFetch
    OpStore
    OpAdd
    OpSub
    OpMul
    OpDimod
    OpAnd
    OpOr
    OpXor
    OpShl
    OpShr
    OpZeroExit
    OpInc
    OpDec
    OpIn
    OpOut
    OpWait
)
```
Ngaro Virtual Machine Opcodes.










## <a name="Image">type</a> [Image](/src/target/image.go?s=798:815#L19)
``` go
type Image []Cell
```
Image encapsulates a VM's memory







### <a name="Load">func</a> [Load](/src/target/image.go?s=1200:1255#L26)
``` go
func Load(fileName string, capacity int) (Image, error)
```
Load loads an image from file fileName. The returned slice should have its
length equal to the number of cells in the file and its capacity equal to the
maximum of the requested capacity and the image file size + 1024 free cells.
When using this slice to create a new VM, New will get the lenght to track
the image file size and expand the slice to its full capacity.





### <a name="Image.DecodeString">func</a> (Image) [DecodeString](/src/target/image.go?s=2440:2486#L72)
``` go
func (i Image) DecodeString(start Cell) string
```
DecodeString returns the string starting at position start in the image.
Strings stored in the image must be zero terminated. The trailing '\0' is
not returned.




### <a name="Image.Disassemble">func</a> (Image) [Disassemble](/src/target/image.go?s=3068:3128#L97)
``` go
func (i Image) Disassemble(pc int) (next int, disasm string)
```
Disassemble disassembles the cells at position pc and returns the position of
the next valid opcode and the disassembly string.




### <a name="Image.EncodeString">func</a> (Image) [EncodeString](/src/target/image.go?s=2787:2836#L86)
``` go
func (i Image) EncodeString(start Cell, s string)
```
EncodeString writes the given string at postion start in the Image and
terminates it with a '\0' Cell.




### <a name="Image.Save">func</a> (Image) [Save](/src/target/image.go?s=2014:2069#L56)
``` go
func (i Image) Save(fileName string, shrink bool) error
```
Save saves the image. If the shrink parameter is true, only the portion of
the image from offset 0 to HERE will be saved.




## <a name="InHandler">type</a> [InHandler](/src/target/vm.go?s=2821:2870#L98)
``` go
type InHandler func(i *Instance, port Cell) error
```
InHandler is the function prototype for custom IN handlers.










## <a name="Instance">type</a> [Instance](/src/target/vm.go?s=909:1373#L25)
``` go
type Instance struct {
    PC    int    // Program Counter (aka. Instruction Pointer)
    Image Image  // Memory image
    Ports []Cell // I/O ports
    // contains filtered or unexported fields
}
```
Instance represents an Ngaro VM instance.







### <a name="New">func</a> [New](/src/target/vm.go?s=5537:5611#L174)
``` go
func New(image Image, imageFile string, opts ...Option) (*Instance, error)
```
New creates a new Ngaro Virtual Machine instance.

The image parameter is the Cell array used as memory by the VM. Usually
loaded from file with the Load function. Note that New expects the lenght of
the slice to be the actual image file size (in Cells) and its capacity set to
the run-time image size, so New will expand the slice to its full capacity
before using it.

The imageFile parameter is the fileName that will be used to dump the
contents of the memory image. It does not have to exist or even be writable
as long as no user program requests an image dump.

Options will be set by calling SetOptions.





### <a name="Instance.Address">func</a> (\*Instance) [Address](/src/target/vm.go?s=6823:6858#L220)
``` go
func (i *Instance) Address() []Cell
```
Address returns the address stack. Note that value changes will be reflected
in the instance's stack, but re-slicing will not affect it. To add/remove
values on the address stack, use the Rpush and Rpop functions.




### <a name="Instance.Data">func</a> (\*Instance) [Data](/src/target/vm.go?s=6494:6526#L210)
``` go
func (i *Instance) Data() []Cell
```
Data returns the data stack. Note that value changes will be reflected in the
instance's stack, but re-slicing will not affect it. To add/remove values on
the data stack, use the Push and Pop functions.




### <a name="Instance.Drop">func</a> (\*Instance) [Drop](/src/target/core.go?s=1165:1196#L52)
``` go
func (i *Instance) Drop(v Cell)
```
Drop removes the top item from the data stack.




### <a name="Instance.Dump">func</a> (\*Instance) [Dump](/src/target/vm.go?s=7631:7673#L261)
``` go
func (i *Instance) Dump(w io.Writer) error
```
Dump dumps the virtual machine stacks and image to the specified io.Writer.




### <a name="Instance.In">func</a> (\*Instance) [In](/src/target/io.go?s=2711:2749#L89)
``` go
func (i *Instance) In(port Cell) error
```
In is the default IN handler for all ports.




### <a name="Instance.InstructionCount">func</a> (\*Instance) [InstructionCount](/src/target/vm.go?s=7015:7058#L228)
``` go
func (i *Instance) InstructionCount() int64
```
InstructionCount returns the number of instructions executed so far.




### <a name="Instance.Out">func</a> (\*Instance) [Out](/src/target/io.go?s=2858:2900#L96)
``` go
func (i *Instance) Out(v, port Cell) error
```
Out is the default OUT handler for all ports.




### <a name="Instance.Pop">func</a> (\*Instance) [Pop](/src/target/core.go?s=1390:1419#L63)
``` go
func (i *Instance) Pop() Cell
```
Pop pops the value on top of the data stack and returns it.




### <a name="Instance.Push">func</a> (\*Instance) [Push](/src/target/core.go?s=1264:1295#L57)
``` go
func (i *Instance) Push(v Cell)
```
Push pushes the argument on top of the data stack.




### <a name="Instance.PushInput">func</a> (\*Instance) [PushInput](/src/target/io.go?s=2320:2361#L73)
``` go
func (i *Instance) PushInput(r io.Reader)
```
PushInput sets r as the current input RuneReader for the VM. When this reader
reaches EOF, the previously pushed reader will be used.




### <a name="Instance.Rpop">func</a> (\*Instance) [Rpop](/src/target/core.go?s=1658:1688#L76)
``` go
func (i *Instance) Rpop() Cell
```
Rpop pops the value on top of the address stack and returns it.




### <a name="Instance.Rpush">func</a> (\*Instance) [Rpush](/src/target/core.go?s=1522:1554#L70)
``` go
func (i *Instance) Rpush(v Cell)
```
Rpush pushes the argument on top of the address stack.




### <a name="Instance.Run">func</a> (\*Instance) [Run](/src/target/core.go?s=2143:2179#L92)
``` go
func (i *Instance) Run() (err error)
```
Run starts execution of the VM.

If an error occurs, the PC will will point to the instruction that triggered
the error.

If the VM was exited cleanly from a user program with the `bye` word, the PC
will be equal to len(i.Image) and err will be nil.

If the last input stream gets closed, the VM will exit and return io.EOF.
This is a normal exit condition in most use cases.




### <a name="Instance.SetOptions">func</a> (\*Instance) [SetOptions](/src/target/vm.go?s=4738:4789#L152)
``` go
func (i *Instance) SetOptions(opts ...Option) error
```
SetOptions sets the provided options.




### <a name="Instance.Tos">func</a> (\*Instance) [Tos](/src/target/core.go?s=1059:1088#L47)
``` go
func (i *Instance) Tos() Cell
```
Tos returns the top stack item.




### <a name="Instance.Wait">func</a> (\*Instance) [Wait](/src/target/io.go?s=3379:3422#L116)
``` go
func (i *Instance) Wait(v, port Cell) error
```
Wait is the default WAIT handler bound to ports 1, 2, 4, 5 and 8. It can be
called manually by custom handlers that override default behaviour.




### <a name="Instance.WaitReply">func</a> (\*Instance) [WaitReply](/src/target/io.go?s=3146:3188#L109)
``` go
func (i *Instance) WaitReply(v, port Cell)
```
WaitReply writes the value v to the given port and sets port 0 to 1. This
should only be used by WAIT port handlers.




## <a name="Option">type</a> [Option](/src/target/vm.go?s=1395:1428#L47)
``` go
type Option func(*Instance) error
```
Option interface







### <a name="AddressSize">func</a> [AddressSize](/src/target/vm.go?s=1933:1966#L65)
``` go
func AddressSize(size int) Option
```
AddressSize sets the address stack size. It will not erase the stack, but data nay
be lost if set to a smaller size. The default is 1024 cells.


### <a name="BindInHandler">func</a> [BindInHandler](/src/target/vm.go?s=3544:3599#L114)
``` go
func BindInHandler(port Cell, handler InHandler) Option
```
BindInHandler binds the porvided IN handler to the given port.

The default IN handler behaves according to the specification: it reads the
corresponding port value from Ports[port] and pushes it to the data stack.
After reading, the value of Ports[port] is reset to 0.

Custom hamdlers do not strictly need to interract with Ports field. It is
however recommended that they behave the same as the default.


### <a name="BindOutHandler">func</a> [BindOutHandler](/src/target/vm.go?s=4016:4073#L127)
``` go
func BindOutHandler(port Cell, handler OutHandler) Option
```
BindOutHandler binds the porvided OUT handler to the given port.

The default OUT handler just stores the given value in Ports[port].
A common use of OutHandler when using buffered I/O is to flush the output
writer when anything is written to port 3. Such handler just ignores the
written value, leaving Ports[3] as is.


### <a name="BindWaitHandler">func</a> [BindWaitHandler](/src/target/vm.go?s=4556:4615#L144)
``` go
func BindWaitHandler(port Cell, handler WaitHandler) Option
```
BindWaitHandler binds the porvided WAIT handler to the given port.

WAIT handlers are called only if the value the following conditions are both
true:


	- the value of the bound I/O port is not 0
	- the value of I/O port 0 is not 1

Upon completion, a WAIT handler should call the WaitReply method which will
set the value of the bound port and set the value of port 0 to 1.


### <a name="DataSize">func</a> [DataSize](/src/target/vm.go?s=1574:1604#L51)
``` go
func DataSize(size int) Option
```
DataSize sets the data stack size. It will not erase the stack, but data nay
be lost if set to a smaller size. The default is 1024 cells.


### <a name="Input">func</a> [Input](/src/target/vm.go?s=2219:2249#L78)
``` go
func Input(r io.Reader) Option
```
Input pushes the given RuneReader on top of the input stack.


### <a name="Output">func</a> [Output](/src/target/vm.go?s=2467:2497#L84)
``` go
func Output(t Terminal) Option
```
Output configures the output Terminal. For simple I/O, the helper function
NewVT100Terminal will build a Terminal wrapper around an io.Writer.


### <a name="Shrink">func</a> [Shrink](/src/target/vm.go?s=2655:2686#L93)
``` go
func Shrink(shrink bool) Option
```
Shrink enables or disables image shrinking when saving it. The default is
false.





## <a name="OutHandler">type</a> [OutHandler](/src/target/vm.go?s=2937:2990#L101)
``` go
type OutHandler func(i *Instance, v, port Cell) error
```
OutHandler is the function prototype for custom OUT handlers.










## <a name="Terminal">type</a> [Terminal](/src/target/io.go?s=1566:1737#L36)
``` go
type Terminal interface {
    io.Writer
    Flush() error
    Size() (width int, height int)
    Clear()
    MoveCursor(x, y int)
    FgColor(fg int)
    BgColor(bg int)
    Port8Enabled() bool
}
```
Terminal encapsulates methods provided by a terminal output. Apart from
WriteRune, all methods can be implemented as no-ops if the underlying output
does not support the corresponding functionality.

WriteRune writes a single Unicode code point, returning the number of bytes written and any error.

Flush writes any buffered unwritten output.

Size returns the width and height of the terminal window. Should return 0, 0
if unsupported.

Clear clears the terminal window and moves the cursor to the top left.

MoveCursor moves the cursor to the specified column and row.

FgColor and BgColor respectively set the foreground and background color of
all characters subsequently written.

Port8Enabled should return true if the MoveCursor, FgColor and BgColor
methods have any effect.







### <a name="NewVT100Terminal">func</a> [NewVT100Terminal](/src/target/io_helpers.go?s=2555:2655#L88)
``` go
func NewVT100Terminal(w io.Writer, flush func() error, size func() (width int, height int)) Terminal
```
NewVT100Terminal returns a new Terminal implementation that uses VT100 escape
sequences to implement the Clear, CusrosrPos, FgColor and BgColor methods.

The caller only needs to provide the functions implementing Flush and Size.
Either of these functions may be nil, in which case they will be implemented
as no-ops.





## <a name="WaitHandler">type</a> [WaitHandler](/src/target/vm.go?s=3059:3113#L104)
``` go
type WaitHandler func(i *Instance, v, port Cell) error
```
WaitHandler is the function prototype for custom WAIT handlers.














- - -
Generated by [godoc2md](http://godoc.org/github.com/davecheney/godoc2md)
