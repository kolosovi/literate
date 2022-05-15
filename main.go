package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

func main() {
	args, err := parseArgs()
	if err != nil {
		log.Println(err.Error())
		printUsage()
		os.Exit(1)
	}
	err = literate(args)
	if err != nil {
		log.Println(err.Error())
		os.Exit(1)
	}
}

func parseArgs() (Args, error) {
	var args Args
	flag.StringVar(&(args.Lexer), "lexer", "", "Chroma Highlight lexer name")
	flag.StringVar(
		&(args.Anchor),
		"anchor",
		"",
		"Anchor to detect literate comments. Everything to the left of anchor including itself & its modifiers is discarded, the rest is included in the output",
	)
	flag.Parse()
	args.Filename = flag.Arg(0)
	if args.Lexer == "" {
		return args, fmt.Errorf("--lexer is a required option")
	}
	if args.Anchor == "" {
		return args, fmt.Errorf("--anchor is a required option")
	}
	if args.Filename == "" {
		return args, fmt.Errorf("<filename> is a required argument")
	}
	return args, nil
}

func printUsage() {
	fmt.Println("Usage: literate --lexer <Chroma Highlight lexer name> --anchor <anchor> <filename>")
}

type Args struct {
	Lexer    string
	Anchor   string
	Filename string
}

func literate(args Args) error {
	file, err := os.Open(args.Filename)
	if err != nil {
		return fmt.Errorf("cannot open file '%v' for reading: %w", args.Filename, err)
	}
	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		return fmt.Errorf("cannot read from file '%v': %w", args.Filename, err)
	}
	content := string(fileBytes)
	lines := strings.Split(content, "\n")
	lineno := 1
	var outputBuffer []string
	state := StateFree
	startAnchor := args.Anchor + " " + startModifier
	endAnchor := args.Anchor + " " + endModifier
	literateAnchor := args.Anchor
	anchors := []string{startAnchor, endAnchor, literateAnchor}
	for i := range lines {
		anchor, pastAnchorIndex := detectAnchor(lines[i], anchors)
		switch state {
		case StateFree:
			switch anchor {
			case startAnchor:
				state = StateLiterate
			case literateAnchor:
				return wrapErr(i, "unexpected anchor '%v' outside of literate block", anchor)
			case endAnchor:
				return wrapErr(i, "unexpected anchor '%v' outside of literate block", anchor)
			default:
				lineno++
			}
		case StateLiterate:
			switch anchor {
			case startAnchor:
				return wrapErr(i, "unexpected anchor '%v' inside of literate block", anchor)
			case literateAnchor:
				outputBuffer = append(outputBuffer, linePastAnchor(lines[i], pastAnchorIndex))
			case endAnchor:
				state = StateFree
			default:
				outputBuffer = append(outputBuffer, "")
				outputBuffer = append(outputBuffer, fmtCodeBlockPrologue(args.Lexer, lineno))
				outputBuffer = append(outputBuffer, lines[i])
				lineno++
				state = StateCode
			}
		case StateCode:
			switch anchor {
			case startAnchor:
				return wrapErr(i, "unexpected anchor '%v' inside of code block", anchor)
			case literateAnchor:
				outputBuffer = append(outputBuffer, codeBlockEpilogue)
				outputBuffer = append(outputBuffer, linePastAnchor(lines[i], pastAnchorIndex))
				state = StateLiterate
			case endAnchor:
				outputBuffer = append(outputBuffer, codeBlockEpilogue)
				state = StateFree
			default:
				outputBuffer = append(outputBuffer, lines[i])
				lineno++
			}
		default:
			return fmt.Errorf("unknown state '%v'", state)
		}
	}
	fmt.Println(strings.Join(outputBuffer, "\n"))
	return nil
}

func linePastAnchor(line string, pastAnchorIndex int) string {
	if pastAnchorIndex == len(line) {
		return ""
	}
	return line[pastAnchorIndex:]
}

func fmtCodeBlockPrologue(lexer string, lineno int) string {
	return fmt.Sprintf(
		"{{< highlight %v \"linenos=table,linenostart=%v\" >}}",
		lexer,
		lineno,
	)
}

const codeBlockEpilogue = "{{< / highlight >}}"

func wrapErr(numFromZero int, format string, args ...interface{}) error {
	return fmt.Errorf("line %v: %w", numFromZero+1, fmt.Errorf(format, args))
}

func detectAnchor(line string, anchors []string) (string, int) {
	for i := range anchors {
		start := strings.Index(line, anchors[i])
		if start == -1 {
			continue
		}
		pastAnchorIndex := start + len(anchors[i])
		if pastAnchorIndex < len(line) && line[pastAnchorIndex] == ' ' {
			pastAnchorIndex++
		}
		return anchors[i], pastAnchorIndex
	}
	return "", -1
}

type State string

const (
	StateFree     State = "FREE"
	StateLiterate State = "LITERATE"
	StateCode     State = "CODE"
)

const (
	startModifier = "START"
	endModifier   = "END"
)
