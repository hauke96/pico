package main

import (
	"os"
	"regexp"
)

import (
//"github.com/hauke96/go-cape"
)

var FilePath string = ""

func main() {
	//TODO fix issues in go-cape to allow anonymous arguments (arguments without a identifier in front)
	//	parser := cape.NewParser()
	//	encodeFlagStatus := parser.RegisterArgument("encode", "e", "Setting this flag will encode (compress) the given file. This includes storing into an ipf file. Not setting this flag will load the ipf file and show it in a window.").Default("false").Bool()
	//	parser.Parse()

	encodeFlagStatus := getEncodeFlagStatus()

	verifyThatFileExists(encodeFlagStatus)
}

func getEncodeFlagStatus() bool {
	r, e := regexp.Compile("(-e)|(--encode)")
	if e != nil {
		Panic("Developement error, please report: %s", e.Error())
	}

	regexMatches := r.MatchString(os.Args[1])

	if len(os.Args) == 3 {
		if !regexMatches {
			Panic("Please specify the encode flag as second argument and the file name as third. Use -e oder --encode to do this.")
		}
		return true
	}

	// there're only 2 arguments, the regex is not allowed to match here
	if regexMatches {
		Panic("Please specify the filename as second argument! To enable encoding use -e or --enable first and THEN the filename.")
	}

	return false
}

func verifyThatFileExists(encodeFlagStatus bool) {
	filePathIndex := -1

	if encodeFlagStatus {
		filePathIndex = 3
	} else {
		filePathIndex = 2
	}

	if len(os.Args) == filePathIndex { // [0]=app path, [1]=encode flag, [2]=file path
		FilePath = os.Args[filePathIndex-1]
	} else {
		Panic("Please specify the file path as well.")
	}
}
