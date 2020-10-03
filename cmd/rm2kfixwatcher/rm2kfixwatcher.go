package main

import (
	"bufio"
	"fmt"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/silbinarywolf/rm2kpng"
	"gopkg.in/fsnotify.v1"
)

func convertFileInPlace(srcFilename string) error {
	const (
		afterSuffix  = ".afterRm2kFix"
		beforeSuffix = ".beforeRm2kFix"
	)
	srcFile, err := os.Open(srcFilename)
	if err != nil {
		return err
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

	argsWithoutProg := os.Args[1:]
	if len(argsWithoutProg) == 0 {
		log.Printf(`
rm2kfixwatcher is a tool for auto-fixing PNG files so they work in RPG Maker. It will "watch" an RPG Maker project folder for changes and automatically convert PNG files to an 8-bit PNG (if they do not exceed 255 colors)
		
How to use (beginners)
-------------------------
The easiest way to use this is to just drag your RPG Maker folder onto this exe file.

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
	path := argsWithoutProg[0]
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
	charsetPath := path + string(filepath.Separator) + "Charset"
	if _, err := os.Stat(charsetPath); err != nil {
		log.Fatalf("Unable to find \"Charset\" in given folder: %s", charsetPath)
	}
	chipsetPath := path + string(filepath.Separator) + "Chipset"
	if _, err := os.Stat(chipsetPath); err != nil {
		log.Fatalf("Unable to find \"Chipset\" in given folder: %s", chipsetPath)
	}

	// Setup watcher for all RPG Maker asset folders that support transparency
	// - CharSet
	// -
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("Unable to start file watcher: %v", err)
	}
	watcher.Add(charsetPath)
	watcher.Add(chipsetPath)

	log.Printf("Waiting for you to change files in Charset/Chipset folders...\n")

	//
	filesToUpdate := make([]string, 0, 16)
	for {
		filesToUpdate = filesToUpdate[:0]
		select {
		case event := <-watcher.Events:
			dir := event.Name
			// if already added to list of assets to update, continue the outer loop
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
			if filepath.Ext(dir) != ".png" {
				// only check for .png file changes
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
			for retryCount := 1; retryCount <= 10; retryCount++ {
				// todo(Jae): Add more error types to ignore in switch case
				if err := convertFileInPlace(path); err != nil {
					switch err := err.(type) {
					case rm2kpng.ErrRm2kCompatiblePNG:
						// if file is already compatible, ignore and move on
						continue FileUpdateLoop
					case *os.PathError:
						// NOTE(Jae): In my testing, saving in MS-Paint has a lock
						// on the file for an indeterminate amount of time, so we
						// try again a few times before giving up.
						//
						// In my testing, it takes ~3 retries on my machine with a
						// 10ms sleep
						retryCount++
						time.Sleep(10 * time.Millisecond)
						continue
					default:
						// unhandled error
						log.Fatalf("%T: Failed to convert changed file: %s\nerror: %v", err, path, err)
					}
				}
				log.Printf("Fixed file: %s", path)
				if retryCount > 1 {
					log.Printf("(retries taken: %d)", retryCount)
				}
				break
			}
		}
	}
}
