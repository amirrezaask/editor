# Preditor
## Programmable Editor
Simple text editor implemented in Golang using Raylib with the goal of replacing Emacs for me, easier to extend and much faster and better language to work with than Elisp.
 
## Features So far

- reading/writing files
- simple keybindings (no key chords yet)
- left mouse click
- mouse scroll
- basic emacs bindings for movements
- Line numbers
- Basic status bar to show file name, file state
- Selecting text
- Incremental Buffer Search
- Support BeforeSaveHook to use formatting tools like Go fmt
- Auto format go code using pkg/format
- Copy/paste
- Basic Regex syntax highlighting for keywords and types.

## TODO:
- Zoom in/out (increase/decrease) font size
- Undo
- Fuzzy file finder
- Command output buffer ( run a command and see it's result, similar to *Compile Mode* in Emacs)

# Screenshot
![Main.go](assets/screenshot.png)
![Open File Menu](assets/files.png)
