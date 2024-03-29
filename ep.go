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

	"path/filepath"
	"github.com/trivago/grok"
	"compress/gzip"
    //gzip "github.com/klauspost/pgzip"
	"sync"
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
	
    return info != nil && !info.IsDir()
}

func parseAndOutput(line string, p parser.Parser) {
	result := p.ParseLine(line)
	jsonresult,_ := json.Marshal(result)
	fmt.Println(string(jsonresult))
}

func parseAndOutputWithMetadata(line string, p parser.Parser, result map[string]interface{}, pathDetails bool) {
	p.ParseLineWithMetadata(line, result)
	if ! pathDetails {
		delete(result, "in_relative_path")
		delete(result, "in_absolute_path")
		delete(result, "in_filename")
		delete(result, "in_dir")
		delete(result, "gzip")
	}
	jsonresult,_ := json.Marshal(result)
	fmt.Println(string(jsonresult))
}

func main() {

	pathPatternConfFile := flag.String("pconf", "path-patterns.txt", "set patterns file for input file path metadata extraction")
	pathDetails := flag.Bool("pd", false, "output path details if dealing with files")
	logToFile := flag.String("log", "", "enable logging. \"-\" for stdout, filename otherwise")
	logDebug := flag.Bool("debug", false, "enable deug logging.")
	outputConfSimple := flag.Bool("os", false, "output pattern conf (short format)")
	outputConfJson := flag.Bool("oj", false, "output pattern conf structure as json")
	parseSingleThread := flag.Bool("single-threaded", false, "use single threaded processing")
	flag.Parse()

	/**
		set up logging
	**/
	logger = elog.GetELogger(*logToFile, "ep", *logDebug)
	parser.SetELogger(*logToFile, *logDebug)
	
	// find out executable dir (we use it as default basepath for conf files later)
	ex, err := os.Executable()
    if err != nil {
        panic(err)
    }
    exPath := filepath.Dir(ex)

	logger.Println("starting with conf values - pattern:", patternsArg, "; conf:", *patternConfFile)

	var p parser.Parser

	if len(patternsArg) > 0 {
		confStr := "event:\n"
		for _, pat := range patternsArg {
			confStr += "  - " + pat + "\n"
		}
		p = parser.ParserFromBytes([]byte(confStr))
	} else {
		
		var confFile string
		
		if fileExists(filepath.Join(exPath, *patternConfFile)) {
			confFile = filepath.Join(exPath, *patternConfFile)
		} else {
			confFile = *patternConfFile
		}
		p = parser.ParserFromFile(confFile)
	}
    
	if *outputConfSimple {
        p.PrettyPrintPatterns()
        os.Exit(0)
    }
	if *outputConfJson {
		jsondata,_ := json.Marshal(p)
		fmt.Println(string(jsondata))
		os.Exit(0)
	}

	var pathCompiledPatterns []*grok.CompiledGrok
	
	grokPatternsForPathPatternMatching := make(map[string]string)
	grokPatternsForPathPatternMatching["DIR"] = "[^\\/]+"

	var pConfFile string
	if fileExists(filepath.Join(exPath, *pathPatternConfFile)) {
		pConfFile = filepath.Join(exPath, *pathPatternConfFile)
	} else if fileExists(*pathPatternConfFile) {
		pConfFile = *pathPatternConfFile
	}

	if len(pConfFile) > 0 {
		
		file, err := os.Open(pConfFile)
		if err != nil {
			logger.Println("could not read path pattern conf file:",err)
		} else {
			g, err := grok.New(grok.Config{Patterns: grokPatternsForPathPatternMatching, NamedCapturesOnly: true})
			if err != nil {
				logger.Fatal("could not create grok parser for file metadata extraction. Err: ", err)
			}

			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				compiledPattern, err := g.Compile(scanner.Text())
				if err != nil {
					logger.Println("could not compile pattern for file metadata extraction. Err: ", err)
				}
				pathCompiledPatterns = append(pathCompiledPatterns, compiledPattern)
			}
		}
	}	

	scanner := bufio.NewScanner(os.Stdin)
	
	fileInputTypeSet := false
	fileInput := false

	var wg sync.WaitGroup

	logger.Debug("ready to accept events")

	for scanner.Scan() {
		// read line from stdin
		var line = scanner.Text()
		if !fileInputTypeSet {
			fileInput = fileExists(line)
		}
		
		if fileInput {	// handle files
			// create metadata & match stuff
			absolutePath,_ := filepath.Abs(line)
			filename := filepath.Base(absolutePath)
			dir := filepath.Dir(absolutePath)
			ext := filepath.Ext(line)
			
			
			// metadata extraction grok
			//var match map[string]string
			match := make(map[string]string)
			for _,compiledPattern := range pathCompiledPatterns {
				tmp_match := compiledPattern.ParseString(line)
				if len(tmp_match) > 0 {
					for k,v := range tmp_match {
						match[k] = v
					}
				}
			}

			file, err := os.Open(line)
			defer file.Close()
			if err != nil {
				logger.Fatal(err)
			} else {
				var subScanner *bufio.Scanner
				if ext == ".gz" {
					rawContents, err := gzip.NewReader(file)
					if err != nil {
						logger.Println("could not open file as gz, err:", err)
					}
					subScanner = bufio.NewScanner(rawContents)
				} else {
					subScanner = bufio.NewScanner(file)
				}
				
				
				// optionally, resize scanner's capacity for lines over 64K, see next example
				for subScanner.Scan() {
					var subline = subScanner.Text()
					result := make(map[string]interface{})
					result["in_relative_path"] = line
					result["in_absolute_path"] = absolutePath
					result["in_filename"] = filename
					result["in_dir"] = dir
					result["gzip"] = (ext == ".gz")
					for k,v := range match {
						result[k] = v
					}
					
					if *parseSingleThread {
						parseAndOutputWithMetadata(subline, p, result, *pathDetails)
					} else {
						wg.Add(1)
						go func() {
							defer wg.Done()
							parseAndOutputWithMetadata(subline, p, result, *pathDetails)
						}()
					}
				}
			}
		} else { // handle just stdin data
			if *parseSingleThread {
				parseAndOutput(line, p)
			} else {
				wg.Add(1)
				go func() {
					defer wg.Done()
					parseAndOutput(line, p)
				}()
			}
		}
	}
	wg.Wait()
}
