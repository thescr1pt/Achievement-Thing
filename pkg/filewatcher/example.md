package example

import (
	"fmt"
	"log"
	"time"

	"Achievement-Thing/pkg/filewatcher"
)

func exampleUsage() {
	// Create a new watcher with chokidar-like options
	watcher, err := filewatcher.NewWatcher(filewatcher.Options{
		Recursive:     true,                               // Watch subdirectories recursively
		IgnoreInitial: true,                               // Don't emit events for existing files on startup
		Ignored:       []string{"*.log", "*.tmp", ".git"}, // Patterns to ignore
	})
	if err != nil {
		log.Fatal("Error creating watcher:", err)
	}

	// Attach event handlers (like chokidar's .on() method)
	watcher.On(filewatcher.Add, func(event filewatcher.Event) {
		fmt.Printf("➕ File added: %s\n", event.Path)
		if event.Info != nil && event.Info.IsDir() {
			fmt.Printf("   📁 Directory\n")
		} else {
			fmt.Printf("   📄 File\n")
		}
	})

	watcher.On(filewatcher.Change, func(event filewatcher.Event) {
		fmt.Printf("✏️  File changed: %s\n", event.Path)
		if event.Info != nil {
			fmt.Printf("   📏 Size: %d bytes\n", event.Info.Size())
			fmt.Printf("   🕒 Modified: %s\n", event.Info.ModTime().Format(time.Kitchen))
		}
	})

	watcher.On(filewatcher.Remove, func(event filewatcher.Event) {
		fmt.Printf("🗑️  File removed: %s\n", event.Path)
	})

	watcher.On(filewatcher.Error, func(event filewatcher.Event) {
		fmt.Printf("❌ Error: %v\n", event.Err)
	})

	// Add files and folders to watch (like chokidar's .add() method)
	// You can pass a single path, multiple paths, or an array of paths
	err = watcher.Add("./test-folder", "./specific-file.txt", "./another-folder")
	if err != nil {
		log.Fatal("Error adding paths:", err)
	}

	// You can also add paths one by one
	// err = watcher.Add("./single-file.txt")
	// if err != nil {
	//     log.Fatal("Error adding single path:", err)
	// }

	// Do some work...
	fmt.Println("Watcher is running. Try creating, modifying, or deleting files!")
	time.Sleep(30 * time.Second)

	// Close the watcher when done (like chokidar's .close() method)
	fmt.Println("Closing watcher...")
	err = watcher.Close()
	if err != nil {
		log.Printf("Error closing watcher: %v", err)
	}
	fmt.Println("Watcher closed successfully!")
}
