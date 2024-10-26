package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"

	"github.com/natefinch/atomic"
	"github.com/urfave/cli/v2"
)

// Translate returns the cli command to scan for new translation strings.
//
// Since: 2.6
func Translate() *cli.Command {
	return &cli.Command{
		Name:      "translate",
		Usage:     "Scans for new translation strings.",
		ArgsUsage: "translationsFile [source ...]",
		Description: "Recursively scans the current or given directories/files for \n" +
			"translation strings, and creates or updates the translations file.",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "verbose",
				Aliases: []string{"v"},
				Usage:   "Show files that are being scanned etc.",
			},
			&cli.BoolFlag{
				Name:    "update",
				Aliases: []string{"u"},
				Usage:   "Update existing translations (use with care).",
			},
			&cli.BoolFlag{
				Name:    "dry-run",
				Aliases: []string{"n"},
				Usage:   "Scan without storing the results.",
			},
		},
		Action: func(ctx *cli.Context) error {
			translationsFile := ctx.Args().First()
			opts := translateOpts{
				DryRun:  ctx.Bool("dry-run"),
				Update:  ctx.Bool("update"),
				Verbose: ctx.Bool("verbose"),
			}

			if translationsFile == "" {
				fmt.Fprintln(os.Stderr, "Missing argument: translationsFile")
				fmt.Fprintf(os.Stderr, "usage: %s %s [command options] %s\n",
					ctx.App.Name,
					ctx.Command.Name,
					ctx.Command.ArgsUsage,
				)
				os.Exit(1)
			}

			sources, err := findSources(ctx.Args().Tail(), ".go", ".")
			if err != nil {
				return err
			}

			return updateTranslationsFile(translationsFile, sources, &opts)
		},
	}
}

// Takes the files/directories, recursively scans them for files with the given extension,
// and returns them. Uses the current directory when used with an empty list.
func findSources(files []string, ext, cur string) ([]string, error) {
	if len(files) == 0 {
		return findFilesExt(cur, ext)
	}

	sources := []string{}
	for _, file := range files {
		if filepath.Ext(file) == ext {
			sources = append(sources, file)
			continue
		}
		files, err := findFilesExt(file, ext)
		if err != nil {
			return nil, err
		}
		sources = append(sources, files...)
	}

	return sources, nil
}

// Recursively walk the given directory and return all files with the matching extension
func findFilesExt(dir, ext string) ([]string, error) {
	files := []string{}
	err := filepath.Walk(dir, func(path string, fi fs.FileInfo, err error) error {
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil
			}
			return err
		}

		if filepath.Ext(path) != ext || fi.IsDir() || !fi.Mode().IsRegular() {
			return nil
		}

		files = append(files, path)

		return nil
	})
	return files, err
}

type translateOpts struct {
	DryRun  bool
	Update  bool
	Verbose bool
}

// Create or add to translations file by scanning the given files for translation calls.
// Works with and without existing translations file.
func updateTranslationsFile(file string, files []string, opts *translateOpts) error {
	translations := make(map[string]interface{})

	f, err := os.Open(file)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	if f != nil {
		dec := json.NewDecoder(f)
		err := dec.Decode(&translations)
		f.Close()
		if err != nil {
			return err
		}
	}

	if opts.Verbose {
		fmt.Fprintf(os.Stderr, "scanning files: %v\n", files)
	}

	if err := updateTranslationsHash(translations, files, opts); err != nil {
		return err
	}

	if len(translations) == 0 {
		if opts.Verbose {
			fmt.Fprintln(os.Stderr, "no translations found")
		}
		return nil
	}

	b, err := json.MarshalIndent(translations, "", "    ")
	if err != nil {
		return err
	}

	if file == "-" {
		fmt.Printf("%s\n", string(b))
		return nil
	}

	if opts.DryRun {
		return nil
	}

	return writeTranslationsFile(b, file)
}

// Write data to given file and rename atomically to prevent file corruption
func writeTranslationsFile(b []byte, file string) error {
	nf, err := os.CreateTemp(filepath.Dir(file), filepath.Base(file)+"-*")
	if err != nil {
		return err
	}

	n, err := nf.Write(b)
	if err != nil {
		return err
	}

	if n < len(b) {
		return io.ErrShortWrite
	}

	if err := nf.Close(); err != nil {
		return err
	}

	return atomic.ReplaceFile(nf.Name(), file)
}

// Update translations hash by scanning the given files, then parsing and walking the AST
func updateTranslationsHash(m map[string]interface{}, srcs []string, opts *translateOpts) error {
	for _, src := range srcs {
		fset := token.NewFileSet()
		af, err := parser.ParseFile(fset, src, nil, parser.AllErrors)
		if err != nil {
			return err
		}

		ast.Walk(&visitor{opts: opts, m: m}, af)
	}

	return nil
}

// Visitor pattern with state machine to find translations fallback (and key)
type visitor struct {
	opts     *translateOpts
	state    stateFn
	name     string
	key      string
	fallback string
	m        map[string]interface{}
}

// Method to walk AST using interface for ast.Walk
func (v *visitor) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return nil
	}

	if v.state == nil {
		v.state = translateNew
		v.name = ""
		v.key = ""
		v.fallback = ""
	}

	v.state = v.state(v, node)

	return v
}

// State machine to pick out translation key and fallback from AST
type stateFn func(*visitor, ast.Node) stateFn

// All translation calls need to start with the literal "lang"
func translateNew(v *visitor, node ast.Node) stateFn {
	ident, ok := node.(*ast.Ident)
	if !ok {
		return nil
	}

	if ident.Name != "lang" {
		return nil
	}

	return translateCall
}

// A known translation method needs to be used. The two supported cases are:
// - simple cases (L, N): only the first argument is relevant
// - more complex cases (X, XN): first two arguments matter
func translateCall(v *visitor, node ast.Node) stateFn {
	ident, ok := node.(*ast.Ident)
	if !ok {
		return nil
	}

	v.name = ident.Name

	switch ident.Name {
	case "L", "Localize":
		return translateLocalize
	case "N", "LocalizePlural":
		return translateLocalize
	case "X", "LocalizeKey":
		return translateKey
	case "XN", "LocalizePluralKey":
		return translateKey
	}

	return nil
}

// Parse first argument, use string as key and fallback, and finish
func translateLocalize(v *visitor, node ast.Node) stateFn {
	basiclit, ok := node.(*ast.BasicLit)
	if !ok {
		return nil
	}

	val, err := strconv.Unquote(basiclit.Value)
	if err != nil {
		return nil
	}

	v.key = val
	v.fallback = val

	return translateFinish(v, node)
}

// Parse first argument and use as key
func translateKey(v *visitor, node ast.Node) stateFn {
	basiclit, ok := node.(*ast.BasicLit)
	if !ok {
		return nil
	}

	val, err := strconv.Unquote(basiclit.Value)
	if err != nil {
		return nil
	}

	v.key = val

	return translateKeyFallback
}

// Parse second argument and use as fallback, and finish
func translateKeyFallback(v *visitor, node ast.Node) stateFn {
	basiclit, ok := node.(*ast.BasicLit)
	if !ok {
		return nil
	}

	val, err := strconv.Unquote(basiclit.Value)
	if err != nil {
		return nil
	}

	v.fallback = val

	return translateFinish(v, node)
}

// Finish scan for translation and add to translation hash with the right type (singular or plural).
// Only adding new keys, ignoring changed or removed ones.
// Removing is potentially dangerous as there could be dynamic keys that get removed.
// By default ignore existing translations to prevent accidental overwriting.
func translateFinish(v *visitor, node ast.Node) stateFn {
	_, found := v.m[v.key]
	if found {
		if !v.opts.Update {
			if v.opts.Verbose {
				fmt.Fprintf(os.Stderr, "ignoring: %s\n", v.key)
			}
			return nil
		}
		if v.opts.Verbose {
			fmt.Fprintf(os.Stderr, "updating: %s\n", v.key)
		}
	} else {
		if v.opts.Verbose {
			fmt.Fprintf(os.Stderr, "adding: %s\n", v.key)
		}
	}

	switch v.name {
	case "LocalizePlural", "LocalizePluralKey", "N", "XN":
		m := make(map[string]string)
		m["other"] = v.fallback
		v.m[v.key] = m
	default:
		v.m[v.key] = v.fallback
	}

	return nil
}
