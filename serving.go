package main

import (
	//"encoding/json"
    "crypto/md5"
    "fmt"
    "github.com/n3integration/epub"
    "io"
    "io/ioutil"
    "log"
    "regexp"
    "strconv"
    "strings"
    "os"
)

type Ebook struct {
	Author      string `json:"author"`
	Title       string `json:"title"`
	Rights      string `json:"rights"`
	Description string `json:"description"`
	Filename    string `json:"filename"`
}

var searchDirs []string = []string{
    "/home/kim/e/search",
}

var storageDir string = "/home/kim/e/store"

func getEbook(ebookFilename string) (ebook []byte, err error) {
    ebookPath := storageDir + "/" + ebookFilename
    ebook, err = ioutil.ReadFile(ebookPath)
    return ebook, err
}

func getEbooks() (ebooks []Ebook) {
    checkForNewBooks()
    ebookPaths := getEbookPaths([]string{storageDir})
    log.Printf("Getting details from %d ebooks", len(ebookPaths))
	for _, ebookPath := range ebookPaths {
        //grab author, title, rights, description, filename (not whole path!)
        log.Printf("Sending book %s details", ebookPath)
        ebookDetails, err := getBookDetails(ebookPath)
        if err != nil {
            log.Printf("error trying to open %s : %s 😱", ebookPath)
            log.Print(err)
        }
        ebooks = append(ebooks, ebookDetails)
    }
    return
}

func getBookDetails(ebookPath string) (ebook Ebook, err error) {
    openEpub, err := epub.Open(ebookPath)
    if err != nil {
        return ebook, err
    }
    metadata := openEpub.Opf.Metadata // .opf file contains the ebook's metadata
    if len(metadata.Creator) > 0 {
        ebook.Author = strings.TrimSpace(metadata.Creator[0].Data)
    } else {
        ebook.Author = "unknown author"
    }
    if len(metadata.Title) > 0 {
        ebook.Title = strings.TrimSpace(metadata.Title[0].Data)
    } else {
        ebook.Title = "unknown title"
    }
    if len(metadata.Rights) > 0 {
        ebook.Rights = strings.TrimSpace(metadata.Rights[0])
    } else {
        ebook.Rights = ""
    }
    if len(metadata.Description) > 0 {
        ebook.Description = strings.TrimSpace(metadata.Description[0])
    } else {
        ebook.Description = ""
    }

    splitEbookPath := strings.Split(ebookPath, `/`)
    ebook.Filename = splitEbookPath[len(splitEbookPath)-1]

    defer openEpub.Close()

    return ebook, nil
}

func checkForNewBooks() {
    log.Print("Checking for new ebooks...")
    log.Printf("Searching in: %s", strings.Join(searchDirs, "; "))
    ebookSearchPaths := getEbookPaths(searchDirs)
    log.Printf("Ebooks from search path: %s", strings.Join(ebookSearchPaths, "; "))
    ebookStoragePaths := getEbookPaths([]string{storageDir})
    log.Print("Hashing ebooks for comparison...")
    hashedSearchEbooks := hashEbooks(ebookSearchPaths)
    hashedStorageEbooks := hashEbooks(ebookStoragePaths)
    log.Print("Comparing hashes...")
    // find hashes unique in the search path only
    searchEbooksUnique := hashedPathsDifference(hashedSearchEbooks, hashedStorageEbooks)
    if len(searchEbooksUnique) > 0 {
        copyEbooks(searchEbooksUnique, storageDir)
        log.Print("...finished copying.")
    } else {
        log.Print("...no new ebooks found.")
    }
}

func hashEbooks(ebookPaths []string) (hashedPaths map[string]string) {
    // hash all the ebooks at the given paths
    hashedPaths = make(map[string]string)
	for _, path := range ebookPaths {
        ebookData, err := os.Open(path)
        if err != nil {
            log.Print(err)
        }
        hash := md5.New()
        if _, err := io.Copy(hash, ebookData); err != nil {
            log.Print(err)
        }
        hashString := fmt.Sprintf("%x", hash.Sum(nil))
        hashedPaths[hashString] = path
    }
    return
}

func hashedPathsDifference(a map[string]string, b map[string]string) (difference map[string]string) {
    // returns a - b (i.e. what's in a but not in b)
    // O(n^2) but good enough for now
    difference = make(map[string]string)
    for hash_a, path_a := range a {
        matchFound := false
        for hash_b, _ := range b {
            if hash_a == hash_b {
                matchFound = true
                break
            }
        }
        if !matchFound {
            difference[hash_a] = path_a
        }
    }
    return
}

func copyEbooks(searchEbooksUnique map[string]string, storageDir string) {
    log.Printf("Found %d new ebooks!", len(searchEbooksUnique))
    for _, path := range searchEbooksUnique {
        filename := generateNewFilename()
        log.Printf("Copying %s to %s/%s", path, storageDir, filename)
        Copy(path, storageDir + "/" + filename)
        //err := Copy(path, storageDir + "/" + filename)
        // do something with err I guess
    }
}

func generateNewFilename() string {
    // generate new filename of format `e00001.epub`
    ebookPaths := getEbookPaths([]string{storageDir})
    highestNum := 0
	ebookRe, _ := regexp.Compile(`^[\w/]+([\d]{6})\.epub$`)
    for _, ebookPath := range ebookPaths {
        reMatch := ebookRe.FindStringSubmatch(ebookPath)
        if len(reMatch) > 0 {
            digit, _ := strconv.Atoi(reMatch[1])
            if digit > highestNum {
                highestNum = digit
            }
        }
    }
    return fmt.Sprintf("e%06d.epub", highestNum + 1)
}

func getEbookPaths(dirs []string) (ebookPaths []string) {
    // grab all the .epub paths in the storage dir
	for _, dir := range dirs {
        files, err := ioutil.ReadDir(dir)
        //log.Printf("Files found in dir %s: %s", dir, strings.Join(files, "; "))
        if err != nil {
            log.Print(err)
        }
        for _, file := range files {
            filename := file.Name()
            // would be better to check type in another way but this will do for now
            // e.g. from wikipedia: "The first file in the archive must be the mimetype file. It must be unencrypted and uncompressed so that non-ZIP utilities can read the mimetype. The mimetype file must be an ASCII file that contains the string "application/epub+zip". This file provides a more reliable way for applications to identify the mimetype of the file than just the .epub extension."
            plopp := filename[len(filename)-5:]
            if plopp == ".epub" {
                ebookPaths = append(ebookPaths, dir + "/" + filename)
            }
        }
    }
    return
}

// https://stackoverflow.com/a/21061062
// Copy the src file to dst. Any existing file will be overwritten and will not
// copy file attributes.
func Copy(src, dst string) error {
    in, err := os.Open(src)
    if err != nil {
        return err
    }
    defer in.Close()

    out, err := os.Create(dst)
    if err != nil {
        return err
    }
    defer out.Close()

    _, err = io.Copy(out, in)
    if err != nil {
        return err
    }
    return out.Close()
}
