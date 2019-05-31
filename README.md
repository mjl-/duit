[![GoDoc](https://godoc.org/github.com/mjl-/duit?status.svg)](https://godoc.org/github.com/mjl-/duit)
[![Build Status](https://travis-ci.org/mjl-/duit.svg?branch=master)](https://travis-ci.org/mjl-/duit)

# duit - developer ui toolkit

WARNING: this library is work in progress. backwards incompatible changes will be made.


## details

duit is a pure go (*), cross platform, MIT-licensed ui toolkit for developers. the api is small and uncomplicated.

duit works on the bsd's, linux and macos. it should be easy to get running on plan 9. for now, use the windows subsystem for linux on windows.

(*) duit currently needs a helper tool called devdraw, from plan9port (aka plan 9 from user space). plan9port is available for most unix systems, with devdraw in an x11 and native macos variant.


## screenshots

![duit screenshot](https://www.ueber.net/who/mjl/files/duit.png)

you should just try duit. using it and interacting with it gives a more complete impression.


## instructions

setting this up currently requires some effort:

- install plan9port, see https://9fans.github.io/plan9port/ (use their install instructions)
- install a nice font. i use & recommend lato for a modern look. duit will automatically pick it up through $font (through plan9port's fontsrv), e.g.: export font=/mnt/font/Lato-Regular/15a/font

you should now be able to run the code in examples/

devdraw is not yet available as a native binary for windows. for now, use the windows subsystem for linux (ubuntu) on windows along with Xming. see https://github.com/elrzn/acme-wsl for instructions.


## created with duit

see https://github.com/mjl- for applications.
applications created with duit by other developers:

- be the first to add your application here! (:


## more

- for context, see the [announcement blog post](https://www.ueber.net/who/mjl/blog/p/duit-developer-ui-toolkit/).
- for questions, first see FAQ.md.
- file an issue on this github repo if you found a bug.
- submit a PR if you wrote code (and see TODO.md).
- join #duit on freenode (irc).
- mail me at mechiel@ueber.net (no mailing list yet).
