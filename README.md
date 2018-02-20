[![GoDoc](https://godoc.org/github.com/mjl-/duit?status.svg)](https://godoc.org/github.com/mjl-/duit)

# duit - developer ui toolkit


WARNING: this library is unstable. i will break your code with backwards incompatible changes to the api, even for seemingly trivial cosmetic api changes. it's published now for interested fosdem2018 attendants. lots of code needs cleaning up, and there is virtually no documentation yet.


## details

duit is a pure go (*), cross platform, MIT-licensed ui toolkit for developers. the api is small and uncomplicated.

duit works on the bsd's, linux and macos. it should be easy to get running on plan 9. for now, use the windows subsystem for linux on windows.

(*) duit currently needs a helper tool called devdraw, from plan9port (aka plan 9 from user space). plan9port is available for most unix systems, with devdraw in an x11 and native macos variant.


## screenshots

![duit screenshot](https://www.ueber.net/who/mjl/files/duit.png)

you should just try duit. using it and interacting with it gives a more complete impression.


## instructions

setting this up currently requires some effort:

- clone github.com/mjl-/go as 9fans.net/go in your go source tree
- install plan9port, see https://9fans.github.io/plan9port/ (use their install instructions)
- install a nice font. i use & recommend lato for a modern look. duit will automatically pick it up through $font (through plan9port's fontsrv), e.g.: export font=/mnt/font/Lato-Regular/15a/font

you should now be able to run the code in examples/

devdraw is not yet available as a native  binary for windows. for now, use the windows subsystem for linux (ubuntu) on windows along with Xming. see https://github.com/elrzn/acme-wsl for instructions.


## faq

#### q: why write a ui toolkit?

i didn't find the ui toolkit i was looking for.
creating one turned out to be surprisingly easy due to the pre-existence of devdraw and its go library (shoulders of giants).

alternative ways to implement ui's would be:
- use go bindings for existing (native) ui libraries.  upsides: native look & feel. downsides: non-pure go, so not getting the advantages of go. complicated/non-go-like library, or not cross-platform.
- make a web app. upsides: many people are familiar with web api's. downsides: gigantic software stack and api surface for just showing some ui elements: bad for security, agility, maintainability, etc. (unfortunately, most developers don't care). browsers themselves only provide severely limited access to the system, and embedding an entire browser in each application is just insane.

i don't particularly like writing applications with a ui. or writing a ui toolkit.  i use a lot of command-line tools, separate commands on stdin/stdout/stderr that do one thing well. i don't like how we are all still emulating ancient terminals on modern machines. sometimes an interactive user interface is needed.  it's better to use a proper graphical ui for those cases.

#### q: does duit have any new/interesting/noteworthy features?

duit tries to get out of the way of the developer as much as possible.

focus & hover is unified with mouse warping (also a concept from acme). focus follows the mouse. changing focus to the next ui element by hitting "tab" simply warps the mouse to the new UI: focussing it, and making it easy to continue working with the mouse.

most UI's in duit work if you just create them as zero structs, they have sensible default behaviour. this makes the api very easy to use and to get started with. it also means you can easily embed UIs in your own data structures, making them usable as UIs. for examples, see the duitsql code.

there is no need to do locking when changing the ui. you are in control of the main loop, so you know where all ui code (like click handlers)  is being executed. you only modify ui-state from your main loop code.  if you need to run blocking go code, just spawn a go routine, and afterwards just send a closure that performs the ui modifications on the dui.Call channel to get it executed by the main loop.

some of duit's design and UI elements are inspired by plan 9 and may feel unfamiliar ("unintuitive") to you.  scrollbars work like in acme, with mouse buttons 1,2,3 doing different scrolling: b1 scrolls up, b3 down, b2 absolutely (use the alt and cmd modifier keys to simulate b2,b3 clicks).  the scrollbar is positioned on the left side of the window (where your mouse often is). text selection in an editor (Edit) works like in acme.

low & high dpi dispays both just work. if you need to specify sizes on UI fields, specify them in low dpi pixels. the UI's convert those to high dpi as needed.

#### q: any tips for writing programs?

start by taking a look at the examples/.
you might also look at more "real" applications, see the duit* repos at https://github.com/mjl-.
documentation will be available at https://godoc.org/github.com/mjl-/duit.

#### q: how usable is duit?

duit is still very much work in progress. the api will change. you will run into bugs. there will likely be major refactoring. but you can already start to write real applications. and most of the important UI elements exist, at least partially.

#### q: how can i implement my ui own ui widget?

it's surprisingly easy. just implement the UI interface. btw, they aren't called "widgets", just UI's, after the interface name.

for container-like UI's, the hard function is Layout, for the others you can probably just use the Kid*-functions. for non-container UI's (like buttons, labels), the layout is often much easier, but you'll put more effort in the Draw, Key and Mouse-functions.

one last tip: the function keys toggle various debug modes. like logging all mouse/key events, or printing the current UI hierarchy, or forcing a redraw. look at the code to learn which keys does what.

#### q: how to pronounce duit

either pronounce it as the dutch word, or otherwise as "do it".
DO IT.

#### q: can i contribute?

absolutely, there is a big non-exhaustive list of todo's at the bottom of this page. but the easiest way to help is to just use duit, create programs, let me know about the bugs and missing features.

#### q: how to communicate?

file an issue or PR on github. on freenode, join #duit. there is no mailing list or website (yet). send email to mechiel@ueber.net.


## created with duit

a list of applications created with duit:

- be the first to add your application here! (:


## todo

- edit: scrollbar remains drawn with hover after moving mouse out of scrollbar
- edit: more vi commands
- edit: do not trash history in Saved(), but adjust the offsets to the new file contents
- edit: fix ScrollCursor so it knows about linewraps. for forward reading, have to start at ui.offset, then read forward, and keep adjusting ui.offset.
- edit: don't do disk i/o in main thread
- edit: provide function to save to same file as one already open.  should we write new file, then replace old one?
- edit: plumbing? look
- edit: render tab with configurable width
- lots of code cleanup
- kids* drawing: should allocate image for kid to draw on if it is larger than available size.  can put child size & image in Kid.
- figure out which keyboard shortcuts can be safely used across all the system
- place: should draw overlapping children on their own image, so it doesn't have to redraw children all the time. initially, we'll just keep images for all the children.
- try draw lib for plan9, https://github.com/mortdeus/draw9 or https://bitbucket.org/mischief/draw9; probably needs some modification
- option for devdraw for windows: https://bitbucket.org/mtrS/pf9; the binaries don't seem to work. code may be old.
- gridlist: draw own scrollbar, draw header fixed at the top
- gridlist: change to take a dynamic source of rows, so we can read on demand
- gridlist: implement rows where a cell has multiple lines
- warp: a mechanism to suppress warp on click. having a key pressed would be good (not currently possible with devdraw).
- label: text selection with mouse, with cmd+a/n, cmd+c for copying selection.
- need to find a solution for having field take up only as much as is available, not entire width.
- scroll: do not draw entire child UI if it is big, but perhaps only 2x scroll size so some scroll can be done, but ask child to redraw at some point. saves image memory.
- field: more like edit. perhaps even merge them. or make a field a special case of edit. would give it the same vi key editing, mouse selection, etc. major difference is rendering: field renders different part of content based on cursor.
- attempt to write a json encoder for entire ui. would need a marshal/unmarshal on kid, for the type of the UI. requires changes to UIs that require functions to layout: what do to for place? horizontal/vertical can just get a default split function.
- horizontal scrolling. or should uis implement that themselves when they think it is necessary?
- more ui elements?
- maybe: separate scrollbar from other uis, where they interact with function calls. so we can have a scroll bar that scrolls two other ui's. tk has this.
- learn from other UI toolkits
- make duitmap a UI on its own?
- devdraw for windows. should start with plan9port code base. use windows UI support from inferno-os, perhaps also a drawterm. inferno-os's build system works and is clean, but might as well go for some glue code in go, probably easier and with fewer dependencies.
- future: replace dependencies on devdraw. eg with x11 library on unix. some sort of low-level code for macos and windows? find libraries, they might already exist. easiest if it is just a drop-in replacement for 9fans.net/go/draw.
- tooltips?  requires overlays, on top of other UIs. requires deeper modifications, to redraw the appropriate UIs.
- overlays?
- think about how to show animations, eg animated gifs. a UI needs to tell it wants to redraw? or perhaps it can just redraw? (no, because of scrolling with an image copy). should probably just have a timer, and call markdraw.
- figure out how to do proper font selection. eg bigger/smaller, bold/italic, styles (monowidth, serif, sans-serif). currently outside of duit. you need to configure fonts manually at the moment.
- tab-focus: handle shift-tab to go backwards. also requires a LastFocus. devdraw does not support this though...
- text selection with shift-arrows. devdraw doesn't tell us about separate shift events, or shift+arrow keys, so not possible currently.
- shortcut for "focus next" in edit?  tab is just inserted as tab. the edit doesn't know where to warp the pointer to, and cannot tell its caller currently. probably needs change to duit.Result.
- tip: test live resizing with label="page". devdraw treats those windows differently. should change devdraw to make this runtime configurable.
