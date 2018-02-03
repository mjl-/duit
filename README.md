duit - developer ui toolkit

WARNING: this library is very unstable. i will break your code with backwards incompatible changes to the api, even  just because i will make seemingly trivial cosmetic changes. it's published now for interested fosdem2018 attendants. lots of code needs cleaning up, and there is virtually no documentation yet.

# details

duit is a pure go (*), cross platform, MIT-licensed ui toolkit for developers. the api is small and uncomplicated.

duit works on the bsd's, linux and macos. it should be easy to get running on plan 9. windows support is work in progress.

(*) duit currently needs a helper tool called devdraw, from plan9port (aka plan 9 from user space).

# instructions

setting this up currently requires some effort:

- clone github.com/mjl-/go/draw as 9fans.net/go/draw (you might need to clone the entire "go" directory)
- install plan9port, see https://9fans.github.io/plan9port/
- install a nice font. i use & recommend lato for a modern look. duit will automatically pick it up through $font (through plan9port's fontsrv), e.g.: export font=/mnt/font/Lato-Regular/15a/font

you should now be able to run the code in examples/

# todo

- add  todo list here...
- publish the example applications
