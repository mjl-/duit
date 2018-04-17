## duit faq


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

for container-like UI's, the hard function is Layout, for the others you can probably just use duits Kid*-functions. for non-container UI's (like buttons, labels), the layout is often much easier, but you'll put more effort in the Draw, Key and Mouse-functions.

one last tip: the function keys toggle various debug modes. like logging all mouse/key events, or printing the current UI hierarchy, or forcing a redraw. look at the code to learn which key does what.


#### q: how to pronounce duit

either pronounce it as the dutch word, or otherwise as "do it".
DO IT.


#### q: can i contribute?

absolutely, there is a big non-exhaustive list of todo's at the bottom of this page. but the easiest way to help is to just use duit, create programs, let me know about the bugs and missing features.


#### q: how to communicate?

file an issue or PR on github. on freenode, join #duit. there is no mailing list or website (yet). send email to mechiel@ueber.net.
