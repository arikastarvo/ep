package main

import(
	"fmt"
	"flag"
	//"log"
	//"strconv"
	"os"
	"bufio"
	"encoding/json"

	"ep/elog"
	"ep/parser"
)

var logger elog.ELogger

/** argparse **/
type arrayFlags []string

func (i *arrayFlags) String() string {
	var ret string
	for _,val := range *i {
		ret += val
	}
	return ret
    //return strings.Join(*i, ",")
}

func (i *arrayFlags) Set(value string) error {
    *i = append(*i, value)
    return nil
}

var patternsArg arrayFlags

var patternConfFile = flag.String("conf", "patterns.yaml", "set patterns file")
	
func init() {
    // example with short version for long flag
	flag.Var(&patternsArg, "pattern", "set pattern inline (if set, this is used instead of -conf)")
    flag.Var(&patternsArg, "p", "short version of -pattern")
}

func fileExists(filename string) bool {
    info, err := os.Stat(filename)
    if os.IsNotExist(err) {
        return false
    }
    return !info.IsDir()
}

func main() {

	logToFile := flag.String("log", "", "enable logging. \"-\" for stdout, filename otherwise")
	logDebug := flag.Bool("debug", false, "enable deug logging.")
	outputConfSimple := flag.Bool("os", false, "output pattern conf (short format)")
	flag.Parse()

	/**
		set up logging
	**/
	logger = elog.GetELogger(*logToFile, "ep", *logDebug)
	parser.SetELogger(*logToFile, *logDebug)
	

	logger.Println("starting with conf values - pattern:", patternsArg, "; conf:", *patternConfFile)

	var p parser.Parser

	if len(patternsArg) > 0 {
		confStr := "event:\n"
		for _, pat := range patternsArg {
			confStr += "  - " + pat + "\n"
		}
		p = parser.ParserFromBytes([]byte(confStr))
	} else {
		p = parser.ParserFromFile(*patternConfFile)
	}
    
	if *outputConfSimple {
        p.PrettyPrintPatterns()
        os.Exit(0)
    }
	/*jsondata,_ := json.Marshal(p)
	fmt.Println(string(jsondata))*/

	scanner := bufio.NewScanner(os.Stdin)
	
	fileInputTypeSet := false
	fileInput := false

	for scanner.Scan() {
		// read line from stdin
		var line = scanner.Text()
		if !fileInputTypeSet {
			fileInput = fileExists(line)
		}
		
		if fileInput {	// handle files
			file, err := os.Open(line)
			defer file.Close()
			if err != nil {
				logger.Fatal(err)
			} else {
				subScanner := bufio.NewScanner(file)
				// optionally, resize scanner's capacity for lines over 64K, see next example
				for subScanner.Scan() {

					var subline = subScanner.Text()
					result := make(map[string]interface{})
					result["filename"] = line
					p.ParseLineWithMetadata(subline, result)
					jsonresult,_ := json.Marshal(result)
					fmt.Println(string(jsonresult))
				}
			}
		} else { // handle just stdin data
			result := p.ParseLine(line)
			jsonresult,_ := json.Marshal(result)
			fmt.Println(string(jsonresult))
		}
	}
}
