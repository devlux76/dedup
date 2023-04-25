package main

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

type FileOp struct {
	Path string
	Hash string
}

const BufferSize = 64 * 1024 // 64 KB

func sha256File(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	buf := make([]byte, BufferSize)
	for {
		n, err := file.Read(buf)
		if n > 0 {
			hasher.Write(buf[:n])
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

type workerPool struct {
	wg           sync.WaitGroup
	fileOpChan   chan FileOp
	filePathChan chan string // Add a channel for sending file paths to workers
}

func newWorkerPool(numWorkers int, db *sql.DB, rootDir string) *workerPool {
	wp := &workerPool{
		fileOpChan:   make(chan FileOp),
		filePathChan: make(chan string), // Initialize the channel
	}

	wp.wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go wp.dbWorker(db)
		go wp.fileWorker() // Start file processing workers
	}

	wp.wg.Add(1)
	go wp.processDirectory(rootDir, rootDir)

	return wp
}

func (wp *workerPool) Wait() {
	wp.wg.Wait()
	close(wp.fileOpChan)
	close(wp.filePathChan) // Close the file path channel
}

func (wp *workerPool) fileWorker() {
	for filePath := range wp.filePathChan {
		wp.processFile(filePath)
	}
	wp.wg.Done()
}

func (wp *workerPool) processFile(filePath string) {
	// Removed the defer wp.wg.Done() line, as the Done() call is now in the fileWorker method
	fmt.Printf("Processing file %s\n", filePath)
	fileHash, err := sha256File(filePath)
	if err != nil {
		fmt.Printf("Error calculating hash for %s: %v\n", filePath, err)
		return
	}
	fmt.Printf("Hash for %s: %s\n", filePath, fileHash)
	wp.fileOpChan <- FileOp{
		Path: filePath,
		Hash: fileHash,
	}
}

func (wp *workerPool) processDirectory(rootDir string, dir string) {
	defer wp.wg.Done()
	fmt.Printf("Processing directory %s\n", dir)
	files, err := os.ReadDir(dir)
	if err != nil {
		fmt.Printf("Error reading directory %s: %v\n", dir, err)
		return
	}

	for _, file := range files {
		filePath := filepath.Join(dir, file.Name())

		// Get file information without following symlinks
		fileInfo, err := os.Lstat(filePath)
		if err != nil {
			fmt.Printf("Error reading file info for %s: %v\n", filePath, err)
			continue
		}

		// Check if the file is a regular file (not a symlink)
		if fileInfo.Mode().IsRegular() {
			// Send the file path to the channel for processing by the worker pool
			wp.filePathChan <- filePath
		} else if fileInfo.IsDir() {
			wp.wg.Add(1)
			go wp.processDirectory(rootDir, filePath)
		}
	}
}

func (wp *workerPool) dbWorker(db *sql.DB) {
	stmt, err := db.Prepare("SELECT file_path FROM file_hashes WHERE hash=?")
	if err != nil {
		fmt.Printf("Error preparing statement: %v\n", err)
		return
	}
	defer stmt.Close()

	for fileOp := range wp.fileOpChan {
		var originalFilePath string
		err := stmt.QueryRow(fileOp.Hash).Scan(&originalFilePath)
		if err != nil && err != sql.ErrNoRows {
			fmt.Printf("Error querying database: %v\n", err)
			continue
		}

		// Check if the file being processed is different from the original file in the database
		if originalFilePath != "" && originalFilePath != fileOp.Path {
			fmt.Printf("Found duplicate file %s\n", fileOp.Path)
			err = os.Remove(fileOp.Path)
			if err != nil {
				fmt.Printf("Error removing duplicate file %s: %v\n", fileOp.Path, err)
				continue
			}
			err = os.Symlink(originalFilePath, fileOp.Path)
			if err != nil {
				fmt.Printf("Error creating symlink for %s: %v\n", fileOp.Path, err)
				continue
			}
			fmt.Printf("Replaced duplicate file %s with symlink to %s\n", fileOp.Path, originalFilePath)
		} else if originalFilePath == "" {
			_, err = db.Exec("INSERT INTO file_hashes (hash, file_path) VALUES (?, ?)", fileOp.Hash, fileOp.Path)
			if err != nil {
				fmt.Printf("Error inserting into database: %v\n", err)
				continue
			}
		}
	}
	wp.wg.Done()
}

func scanAndReplaceDuplicates(rootDir string) error {
	db, err := sql.Open("sqlite3", "file_hashes.db")
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS file_hashes (hash TEXT, file_path TEXT)")
	if err != nil {
		return err
	}

	// Start the worker pool with 5 workers
	wp := newWorkerPool(5, db, rootDir)

	// Wait for all file and directory processing to finish
	wp.Wait()

	return nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <path_to_directory>")
		os.Exit(1)
	}

	rootDir := os.Args[1]
	err := scanAndReplaceDuplicates(rootDir)
	if err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Println("Done scanning and replacing duplicates.")
	}
}
