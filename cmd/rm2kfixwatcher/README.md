# Rm2k Fix Watcher

This is a tool that will watch your RPG Maker project directory for file changes and convert any PNG images that aren't in a compatible RPG Maker 8-bit PNG format to that format, if your PNG does not contain more than 256 colors.

## Installation

Visit the [releases page here](https://github.com/silbinarywolf/rm2kpng/releases) and download it.

## How does it work?

If your PNG file is not already an 8-bit PNG, it will attempt to convert any PNG format to an 8-bit PNG by:
- Iterating over every pixel and building up a palette of colors
- It will decide that the top-left corner is the transparent pixel (except for Chipsets, it picks from the transparent tile)

## How to use

1) Follow **Installation** instructions above.

2) **Backup** your RPG Maker project into a safe place. This application will change your files and has the potential to break your work.

3) Drag your RPG Maker project folder onto the EXE / binary file, this will start the app and it'll watch for file changes
