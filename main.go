package main

import (
	//"bytes"
	//"bufio"
	"fmt"
	"io"

	//"io/ioutil"
	"flag"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"

	//"regexp"
	//"strings"
	//"github.com/adrg/frontmatter"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
	hashtag "go.abhg.dev/goldmark/hashtag"
)

var md goldmark.Markdown

func init() {
	md = goldmark.New(
		goldmark.WithExtensions(&hashtag.Extender{}),
	)
}

func copyFile(srcPath, destPath string) error {
	// Open the source file
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Create the destination file
	destFile, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer destFile.Close()

	// Copy the contents of the source file to the destination file
	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return err
	}

	return nil
}

func assignFrontMatter(path string, content []byte) (string, bool) {

	// Getting filename without extension
	fileName := filepath.Base(path)
	extension := filepath.Ext(path)
	title := fileName[:len(fileName)-len(extension)]

	// front-matter template
	template := `---
title: %s
tags: %s
---
`
	// Parse the Markdown content
	doc := md.Parser().Parse(text.NewReader(content))
	// List the tags.
	hashtags := make(map[string]struct{})
	ast.Walk(doc, func(node ast.Node, enter bool) (ast.WalkStatus, error) {
		if n, ok := node.(*hashtag.Node); ok && enter {
			hashtags[string(n.Tag)] = struct{}{}
		}
		return ast.WalkContinue, nil
	})

	//fmt.Println(hashtags)

	_, present := hashtags["publish"]
	if present {
		var tags string

		for tagName := range hashtags {
			if tagName != "publish" {
				tags += ("\n- " + tagName)
			}
		}
		return fmt.Sprintf(template, title, tags), present
	} else {
		return "", present
	}
}

func createIndex(indexPath string, index string) (error, string) {
	indexContent, err := ioutil.ReadFile(indexPath)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return err, ""
	}

	frontMatter, _ := assignFrontMatter(indexPath, indexContent)

	finalContent := frontMatter + string(indexContent) + index

	return nil, finalContent

}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	in := flag.String("input", "", "Notes folder location within obsidian vault")
	out := flag.String("output", "", "Quartz notes location, must append /contents ")
	flag.Parse()

	if *in == "" {
		fmt.Fprintln(os.Stderr, "Error: Missing required flag -input")
		flag.Usage()
		os.Exit(1)
	}

	if *out == "" {
		fmt.Fprintln(os.Stderr, "Error: Missing required flag -output")
		flag.Usage()
		os.Exit(1)
	}
	indexPath := ""
	index := ""
	err := filepath.WalkDir(*in, func(path string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		// path normalize fp
		//s := filepath.ToSlash(fp)

		if d.IsDir() {
			fmt.Println("Directory:", path)
			return nil
		}

		// Create the corresponding destination path
		relativePath, _ := filepath.Rel(*in, path)
		destPath := filepath.Join(*out, relativePath)

		// Create the directory structure in the destination directory
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}
		if filepath.Ext(d.Name()) == ".md" && filepath.Base(d.Name()) != "_index.md" {
			fmt.Println(filepath.Base(d.Name()))
			// Read the existing file content
			content, err := ioutil.ReadFile(path)
			if err != nil {
				fmt.Println("Error reading file:", err)
				return err
			}

			frontMatter, publish := assignFrontMatter(path, content)
			if publish {
				index += "\n[[" + filepath.Base(d.Name())[:len(filepath.Base(d.Name()))-len(filepath.Ext(d.Name()))] + "]]"
				// Prepend the template to the content
				finalContent := frontMatter + string(content)

				// Write the modified content back to the file
				err = ioutil.WriteFile(destPath, []byte(finalContent), 0644)
				if err != nil {
					fmt.Println("Error writing file:", err)
					return err
				}

				fmt.Println("Metadata appended to", destPath)

			} else {
				fmt.Println("Skipping " + path)
			}

		} else if filepath.Base(d.Name()) == "_index.md" {
			indexPath = path
			fmt.Println(indexPath + "saved")
		} else {
			// Copy the file
			if err := copyFile(path, destPath); err != nil {
				return err
			}

			//fmt.Println("Copied:", path, "to", destPath)
		}

		return nil
	})

	// Create index file
	indexErr, createdIndex := createIndex(indexPath, index)

	if indexErr != nil {
		fmt.Println("Index file error: ", err)
	} else {
		// Create the corresponding destination path
		relativePath, _ := filepath.Rel(*in, indexPath)
		destPath := filepath.Join(*out, relativePath)
		fmt.Println("PATHS: ", relativePath, destPath)

		err = ioutil.WriteFile(destPath, []byte(createdIndex), 0644)
		if err != nil {
			fmt.Println("Error writing file:", err)
		}

		fmt.Println("Index created at:", destPath)

	}

	if err != nil {
		fmt.Println("Error:", err)
	}
}
