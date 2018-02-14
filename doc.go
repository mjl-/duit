/*
Package duit is a pure go, cross-platform, MIT-licensed, UI toolkit for developers.

The examples/ directory has small code examples for working with duit and its UIs. Examples are the recommended starting point.

Instructions and information

Start with NewDUI to create a DUI: essentially a window and all the UI state.

The user interface consists of a hierarchy of "UIs" like Box, Scroll, Button, Label, etc. They are called UIs, after the interface UI they all implement.  The zero structs for UIs have sane default behaviour so you only have to fill in the fields you need.

UIs are kept/wrapped in a Kid, to track their layout/draw state. Use NewKids() to build up the UIs for your application. You won't see much of the Kid-types/functions otherwise, unless you implement a new UI.

You are in charge of the main event loop, receiving mouse/keyboard/window events from the dui.Inputs channel, and typically passing them on unchanged to dui.Input. All callbacks and functions on UIs are called from inside dui.Input. From there you can also safely change the the UIs, no locking required. After changing a UI you are responsible for calling MarkLayout or MarkDraw to tell duit the UI needs a new layout or draw. This may sound like more work, but this tradeoff keeps the API small and easy to use. If you need to change the UI from a goroutine outside of the main loop, e.g. for blocking calls, you can send a function that makes those modifications on the dui.Call channel, which will be run on the main channel through dui.Inputs. After handling an input, duit will layout or draw as necessary, no need to render explicitly.

Embedding a UI into your own data structure is often an easy way to build up UI hiearchies.

Scrolling

Scroll and Edit show a scrollbar. Use button 1 on the scrollbar to scroll up, button 3 to scroll down. If you click more near the top, you scroll less. More near the bottom, more. Button 2 scrolls to the absolute place, where you clicked. Button 4 and 5 are wheel up and wheel down, and also scroll less/more depending on position in the UI.
*/
package duit
