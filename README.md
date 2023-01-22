# vivid
A golang bubbletea TUI wrapping around the [Colormind API](http://colormind.io/)

## Features
Provides all the same features as the original website, and more!:
- Generate a unique color scheme
- Lock colors to prevent them from changing when you generate more
- Move colors around
- Change each color's individual RGB values
- Output the RGB values for each color
- Select a custom style (UI, Fauvism, etc.)
- Scales to any terminal size

## Visuals

### Select a model
Menu:

![Image of the model selection menu](./assets/select_model.png "Selection menu")

### Scaling
Small:

![Image of a terminal scaling small](./assets/scales_small.png "Scales small")

Big:

![Image of a terminal scaling big](./assets/scales_big.png "Scales big")

### Locking
The lines signify being selected or locked.

![Image of some locked colors](./assets/locks.png "Lock visuals")

### Edit Colors
Color editing menu:

![Image of the color editing menu](./assets/color_edit_menu.png "Editing menu")

### Exporting:
Exporting looks like:

![Image of an export in the console](./assets/export_visual.png "Export in terminal")

## Controls

Quit - CTRL+C, Q, or ESC
Navigate selection/editing menu - W, Up Arrow, S, Down Arrow
Select option while in menu - Enter 
Reopen model selection - M
Toggle editing for selected color - E
Finish editing current color - Enter
Lock current selected color - Enter or Space
Move selected color - A, Left Arrow, D, Right Arrow
Swap selected color with color next to it - <, Comma, >, Period
Load new colors from API - R