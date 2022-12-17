package main

import (
	"bufio"
	"flag"
	"fmt"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/karrick/godirwalk"
	"github.com/silbinarywolf/rm2kpng"
	"gopkg.in/fsnotify.v1"
)

var (
	hasDebug      bool
	isFixOnlyMode bool
)

const (
	convertedFileText = "Converted file to 8-bit PNG: %s"
)

func init() {
	flag.BoolVar(&hasDebug, "debug", false, "this flag to get additional debugging information")
	flag.BoolVar(&isFixOnlyMode, "fix", false, "this flag will only run the fixing tool once and won't watch the directory")
}

type errOpenFile struct {
	err error
}

func (err errOpenFile) Error() string {
	return err.err.Error()
}

func convertFileInPlace(srcFilename string) error {
	const (
		afterSuffix  = ".afterRm2kFix"
		beforeSuffix = ".beforeRm2kFix"
	)
	srcFile, err := os.Open(srcFilename)
	if err != nil {
		return errOpenFile{err: err}
	}

	convertedImage, err := rm2kpng.ConvertPNGToRm2kPNG(srcFile)
	if err != nil {
		srcFile.Close()
		return err
	}
	srcFile.Close()

	// Create file for new converted image
	dstFile, err := os.Create(srcFilename + afterSuffix)
	if err != nil {
		return err
	}
	if err := png.Encode(dstFile, convertedImage); err != nil {
		dstFile.Close()
		return err
	}
	// Wait for file to sync on hard-drive so it's definitely saved
	if err := dstFile.Sync(); err != nil {
		return err
	}
	// Close it
	if err := dstFile.Close(); err != nil {
		return err
	}
	// Swap original file name to have ".beforeRm2kFix" suffix
	if err := os.Rename(srcFilename, srcFilename+beforeSuffix); err != nil {
		return fmt.Errorf("Unable to backup original file: %v", err)
	}
	// Swap new fixed file to no longer have a suffix
	if err := os.Rename(srcFilename+afterSuffix, srcFilename); err != nil {
		return err
	}
	// Delete ".beforeRm2kFix" suffix file
	if err := os.Remove(srcFilename + beforeSuffix); err != nil {
		return fmt.Errorf("Unable to remove original: %v", err)
	}
	return nil
}

func main() {
	/* defer func() {
		if r := recover(); r != nil {
			log.Printf("%v", r)
			log.Printf("Press any key to close")
			for {
				input := bufio.NewScanner(os.Stdin)
				input.Scan()
			}
		}
	}() */

	// Disable time logging for this app
	log.SetFlags(0)

	// Parse flags
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		log.Printf(`
rm2kfixwatcher is a tool for auto-fixing PNG files so they work in RPG Maker. It will "watch" an RPG Maker project folder for changes and automatically convert PNG files to an 8-bit PNG (if they do not exceed 256 colors)

How it works
-------------------------
If your PNG files are not already an 8-bit PNG, it will attempt to convert any PNG format to an 8-bit PNG by:
- Iterating over every pixel and building up a palette of colors
- It will decide that the top-left corner is the transparent pixel (except for Chipsets, it picks from the transparent tile)

If your PNG file exceeds 256 colors, it will give up on the conversion process and do nothing.

How to use (beginners)
-------------------------
First, make sure you *backup* your RPG Maker project files to avoid any images becoming corrupted, then to use this drag your RPG Maker folder onto this exe file.

How to use (nerds)
-------------------------
rm2kfixwatcher <file_or_folder>
	
Why does this tool exist?
-------------------------
This tool exists so that users can work in paint tools they're comfortable in without needing to think about managing their palette (until they exceed the max 256 colors that is!).
`)
		log.Printf("\nPress any key to close")
		// wait for input before closing, more beginner friendly
		// i remember being like, 9 years old trying to use a CLI tool. Like what the heck is this.
		// why does it instant close.
		input := bufio.NewScanner(os.Stdin)
		input.Scan()
		return
	}

	// Get path
	var rm2kAssetPathList []string
	{
		path := args[0]
		fileinfo, err := os.Stat(path)
		if os.IsNotExist(err) {
			fileinfo, err = os.Stat(path)
			if os.IsNotExist(err) {
				log.Fatal("File or folder does not exist.")
			}
		}
		if !fileinfo.IsDir() {
			log.Fatal("Must be a folder.")
		}

		// Validate that the user has given an RPG Maker folder
		rpgRuntimePath := path + string(filepath.Separator) + "RPG_RT.exe"
		if _, err := os.Stat(rpgRuntimePath); err != nil {
			log.Fatalf("Unable to find RPG_RT in given folder: %s", rpgRuntimePath)
		}

		// Get asset folders
		assetBaseNameList := []string{
			// Rm2k
			"Battle",
			"Charset",
			"Chipset",
			"FaceSet",
			"Panorama",
			"Picture",
			"Monster",
			"System",
			// Rm2k3
			"Backdrop",
			"Battle2",
			"BattleCharSet",
			"BattleWeapon",
			"Frame",
			"System2",
		}
		for _, baseName := range assetBaseNameList {
			assetFolderPath, err := filepath.Abs(path + string(filepath.Separator) + baseName)
			if err != nil {
				log.Fatalf("Cannot get absolute path: %s", err)
			}
			if _, err := os.Stat(assetFolderPath); err != nil {
				continue
			}
			rm2kAssetPathList = append(rm2kAssetPathList, assetFolderPath)
		}
	}

	// Fix files at start-up
	{
		filesToUpdate := make([]string, 0, 100)
		for _, assetDir := range rm2kAssetPathList {
			err := godirwalk.Walk(assetDir, &godirwalk.Options{
				Callback: func(osPathname string, de *godirwalk.Dirent) error {
					if !de.IsRegular() {
						// ignore directories / symlinks / etc
						return godirwalk.SkipThis
					}
					if filepath.Ext(osPathname) != ".png" {
						// ignore non-png
						return godirwalk.SkipThis
					}
					filesToUpdate = append(filesToUpdate, osPathname)
					return nil
				},
				Unsorted: true,
			})
			if err != nil {
				log.Fatalf("Cannot find \"%s\", err: %s", assetDir, err.Error())
			}
		}
		filesConverted := make([]string, 0, len(filesToUpdate))
		for _, path := range filesToUpdate {
			if err := convertFileInPlace(path); err != nil {
				switch err := err.(type) {
				case rm2kpng.ErrRm2kCompatiblePNG:
					// if file is already compatible, ignore and move on
					continue
				case rm2kpng.ErrRm2kPaletteTooBig:
					// if palette too big,
					log.Printf("Skipping file: %s, error: %s", path, err)
					continue
				default:
					// unhandled error
					log.Printf("Skipping file: %s, error: %s", path, err)
					continue
				}
			}
			filesConverted = append(filesConverted, path)
		}
		if len(filesConverted) == 0 {
			log.Printf("No files converted.")
		} else {
			for _, path := range filesConverted {
				log.Printf(convertedFileText, path)
			}
		}
		if isFixOnlyMode {
			// Exit early if only fixing
			return
		}
	}

	// Setup watcher for all RPG Maker asset folders that support transparency
	// - CharSet
	// -
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("Unable to start file watcher: %v", err)
	}
	for _, assetDir := range rm2kAssetPathList {
		if err := watcher.Add(assetDir); err != nil {
			log.Fatalf(`Unable to watch "%s" folder: %v`, filepath.Base(assetDir), err)
		}
	}

	//
	log.Printf("Waiting for you to change files in asset folders:\n")
	for _, assetDir := range rm2kAssetPathList {
		log.Printf("- %s", filepath.Base(assetDir))
	}

	//
	filesToUpdate := make([]string, 0, 16)
	for {
		filesToUpdate = filesToUpdate[:0]
		select {
		case event := <-watcher.Events:
			dir := event.Name
			// Ignore if not a create/write event for a file, ignore.
			// We don't care about removed/renamed/chmoded file changes.
			if event.Op != fsnotify.Create &&
				event.Op != fsnotify.Write {
				continue
			}
			// Check If already added to list of Rm2k assets to convert
			hasFound := false
			for _, otherDir := range filesToUpdate {
				if otherDir == dir {
					hasFound = true
					break
				}
			}
			if hasFound {
				continue
			}
			// if not a .png, ignore
			if filepath.Ext(dir) != ".png" {
				continue
			}
			filesToUpdate = append(filesToUpdate, dir)
		case err := <-watcher.Errors:
			log.Printf("Watcher error: %s", err.Error())
			continue
		}
		if len(filesToUpdate) == 0 {
			continue
		}
	FileUpdateLoop:
		for _, path := range filesToUpdate {
			var err error
			var retryCount int
			for ; retryCount <= 10; retryCount++ {
				if err = convertFileInPlace(path); err != nil {
					switch err := err.(type) {
					case rm2kpng.ErrRm2kCompatiblePNG:
						// if file is already compatible, ignore and move on
						continue FileUpdateLoop
					case rm2kpng.ErrRm2kPaletteTooBig:
						// If file is incompatible, skip but give info to user
						log.Printf("Failed to convert changed file: %s, error: %s", path, err)
						continue FileUpdateLoop
					case
						// When testing on Windows and saving with MS-Paint
						// we get a file in use error for san indeterminate amount of time,
						// so we try again a few times before giving up.
						// In my testing, it takes ~3 retries on my machine with a
						// 10ms sleep
						errOpenFile,
						// When testing on Mac and saving with Pinta, we get an issue
						// where the image hasn't finished saving yet, so a decoding error
						// occurs. It takes ~5 retries with 10ms sleep
						rm2kpng.ErrRm2kDecode:
						{
							retryCount++
							time.Sleep(10 * time.Millisecond)
							continue
						}
					default:
						// unhandled error
						log.Fatalf("Failed to convert changed file: %s\ninternal error: %s", path, err)
					}
				}
				// exit loop if succeeded or retries failed
				break
			}
			if err != nil {
				// if retries failed
				log.Printf("Was unable to fix file: %s, internal error: %s", path, err)
				continue
			}
			log.Printf(convertedFileText, path)
			if hasDebug && retryCount > 1 {
				log.Printf("(retries taken: %d)", retryCount)
			}
		}
	}
}
