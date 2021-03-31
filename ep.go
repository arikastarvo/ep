package main

import(
	//"regexp"
	"fmt"
	"flag"
	"log"
	"strconv"
	"io"
	"time"
	"os/user"
	"path/filepath"
	"os"
	"bufio"
	"io/ioutil"
	"gopkg.in/yaml.v3"
	"sort"
	"encoding/json"
	//"github.com/vjeantet/grok"
	"github.com/trivago/grok"
)

/** logging **/

/** logging **/
type logWriter struct {
    writer io.Writer
	timeFormat string
	user string
	hostname string
}

func (w logWriter) Write(b []byte) (n int, err error) {
	return w.writer.Write([]byte(time.Now().Format(w.timeFormat) + "\t" + "ep@" + w.hostname + "\t" + w.user + "\t" + strconv.Itoa(os.Getpid()) + "\t" + string(b)))
}
/** end logging **/

// this is for unmarshalling string or list of strings
// https://github.com/go-yaml/yaml/issues/100
type StringArray []string

func (a *StringArray) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var multi []string
	err := unmarshal(&multi)
	if err != nil {
		var single string
		err := unmarshal(&single)
		if err != nil {
			return err
		}
		*a = []string{single}
	} else {
		*a = multi
	}
	return nil
}
// end string/list-of-strings unmarshalling solution

// helper mehtods

func contains(s []string, e string) bool {
    for _, a := range s {
        if a == e {
            return true
        }
    }
    return false
}

// end helper methods

type pattern struct {
	Name string
	Pattern StringArray
	CompiledPattern []grok.CompiledGrok
	Optionalpattern StringArray
	OptionalCompiledPattern []grok.CompiledGrok
	Grokpattern map[string]string
	Order int
	Keepfield bool
	Field string
	Parent StringArray
	Children StringArray
	Cond map[string]string
	CompiledCond map[string]grok.CompiledGrok
	Softcond map[string]string
	CompiledSoftcond map[string]grok.CompiledGrok
}

type conf struct {
	Patterns []pattern `,flow`
}


func parseLine(result map[string]string, parent string) {

	// iterate over patterns
	for _, pat := range patternconf.Patterns {

		if parent != "" && !contains(pat.Parent, parent) {
			continue
		}

		field := "data"
		if pat.Field != "" {
			field = pat.Field
		}

		skip := false

		// check if hard conditions are met
		if len(pat.CompiledCond) > 0 {
			for condField, condCompiledPattern := range pat.CompiledCond {
				value, ok := result[condField]

				if ok { // we have this field, so we check for match

					if ! condCompiledPattern.MatchString(value) {
						// does not match, so we skip this pattern
						skip=true
						break
					}
				} else { // field doesn't exist, so skip matching this pattern
					skip=true
					break
				}
			}
		}

		// check if soft conditions are met
		if ! skip && len(pat.CompiledSoftcond) > 0 {
			for condField, condCompiledPattern := range pat.CompiledSoftcond {
				value, ok := result[condField]

				if ok { // we have this field, so we check for match
					if ! condCompiledPattern.MatchString(value) {
						skip=true
						break
					}
				} else {
					// soft conditions don't fail if field doesn't exist
				}
			}
		}

		// some conditions are unmet, so skip this pattern
		if skip {
			continue
		}

		// do pattern matching
		var match map[string]string
		for _, compiledPat := range pat.CompiledPattern {
			match = compiledPat.ParseString(result[field])
			if len(match) > 0 {
				break
			}
		}

		// if we have a match (captured values), then gather results, optionally parse child patterns and finally break look 	
		if len(match) > 0 {
			result["event_type"] = pat.Name
			result["event_type_path"] += "/" + pat.Name

			// delete source field if not stated otherwise
			value := result[field]
			if ! pat.Keepfield {
				delete(result, field)
			}

			// put data to results object
			for k,v := range match {
				result[k] = v
			}

			// execute optionalpattern matches
			if len(pat.OptionalCompiledPattern) > 0 {
				for _, optionalCompiledPatternItem := range pat.OptionalCompiledPattern {
					optMatch := optionalCompiledPatternItem.ParseString(value)
					for k,v := range optMatch {
						result[k] = v
					}
				}
			}

			// parse child patterns if there exists any
			if len(pat.Children) > 0 {
				parseLine(result, pat.Name)
			}
			break
		}
	}

}

// global variables
var patternconf conf
// global logger sh***
var logger *log.Logger

func parsePatternConfigurationFromFile(configFile string) (conf) {

	fileBuf, err := ioutil.ReadFile(configFile)
	if err != nil {
		logger.Fatal("config file (", configFile, ") read error. Err: ", err)
	}
	return parsePatternConfiguration(fileBuf)
}

func unmarshalRecursive(configuration []byte, forceParent string) (conf) {
	
	var parsedconf conf
	yaml.Unmarshal(configuration, &parsedconf.Patterns)
	sort.SliceStable(parsedconf.Patterns, func(i, j int) bool {
		return parsedconf.Patterns[i].Order < parsedconf.Patterns[j].Order
	})
	
	// index uniq pattern names && set forceParent
	var uniqPatterns []string
	for i, pat := range parsedconf.Patterns {
		
		if !contains(uniqPatterns, pat.Name) {
			uniqPatterns = append(uniqPatterns, pat.Name)
		}

		if len(forceParent) > 0 {
			parsedconf.Patterns[i].Parent = append(parsedconf.Patterns[i].Parent, forceParent)
		}
	}

	// let's try to include other referenced pattern conf files (only applicable to children definitions for now)
	for _, pat := range parsedconf.Patterns {
		for _, child := range pat.Children {
			if !contains(uniqPatterns, child) {
				if _, err := os.Stat(child); err == nil {
					//fmt.Println("going sub level for ", child)
					fileBuf, err := ioutil.ReadFile(child)
					if err != nil {
						logger.Fatal("sub-config file (", child, ") read error. Err: ", err)
					}
					parsedconf.Patterns = append(parsedconf.Patterns, unmarshalRecursive(fileBuf, pat.Name).Patterns...)
				}
			}
		}
	}

	return parsedconf
}

func parsePatternConfiguration(configuration []byte) (conf) {

	// read configurations recursively
	parsedconf := unmarshalRecursive(configuration, "")

	// map children	(set children based on parents in configuration)
	for i, pat := range parsedconf.Patterns {
		for _, subpat := range parsedconf.Patterns {
			if contains(subpat.Parent, pat.Name) && !contains(parsedconf.Patterns[i].Children, subpat.Name) {
				parsedconf.Patterns[i].Children = append(parsedconf.Patterns[i].Children, subpat.Name)
			}
		}
	}

	// map parents (set parents based on children in configuration)
	for i, pat := range parsedconf.Patterns {
		for _, subpat := range parsedconf.Patterns {
			// check if we have this event and it hasn't set as parent yet
			if contains(subpat.Children, pat.Name) && !contains(parsedconf.Patterns[i].Parent, subpat.Name) {
				parsedconf.Patterns[i].Parent = append(parsedconf.Patterns[i].Parent, subpat.Name)
			}
		}
	}

	// pre-compile various patterns (using grok)
	for i, patConf := range parsedconf.Patterns {
		// add default grok patterns to all
		if (patConf.Grokpattern == nil) {
			patConf.Grokpattern = make(map[string]string)
		}
		patConf.Grokpattern["GD"] = ".*"
		// end add default groks
		g, err := grok.New(grok.Config{Patterns: patConf.Grokpattern, NamedCapturesOnly: true})
		if err != nil {
			logger.Println("could not create grok parser for ", patConf.Name, ". Err: ", err)
			continue
		}

		// main patterns
		for _, pat := range patConf.Pattern {
			cg,err := g.Compile(pat)
			if err != nil {
				logger.Println("err: ", err)
				continue
			}
			parsedconf.Patterns[i].CompiledPattern = append(parsedconf.Patterns[i].CompiledPattern, *cg)
		}

		// optional patterns
		for _, pat := range patConf.Optionalpattern {
			cg,err := g.Compile(pat)
			if err != nil {
				logger.Println("err: ", err)
				continue
			}
			parsedconf.Patterns[i].OptionalCompiledPattern = append(parsedconf.Patterns[i].OptionalCompiledPattern, *cg)
		}

		// condition patterns
		parsedconf.Patterns[i].CompiledCond = make(map[string]grok.CompiledGrok)
		for field, pat := range patConf.Cond {
			cg,err := g.Compile(pat)
			if err != nil {
				logger.Println("err: ", err)
				continue
			}
			parsedconf.Patterns[i].CompiledCond[field] = *cg
		}

		// soft condition patterns
		parsedconf.Patterns[i].CompiledSoftcond = make(map[string]grok.CompiledGrok)
		for field, pat := range patConf.Softcond {
			cg,err := g.Compile(pat)
			if err != nil {
				logger.Println("err: ", err)
				continue
			}
			parsedconf.Patterns[i].CompiledSoftcond[field] = *cg
		}
	}

	return parsedconf
}

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

func main() {

	logToFile := flag.String("log", "", "enable logging. \"-\" for stdout, filename otherwise")
	flag.Parse()

	/**
		set up logging
	**/

	if *logToFile == "" {
		logger = log.New(ioutil.Discard, "", 0)
	} else { // enable logging
		// get user info for log
		userobj, err := user.Current()
		usern := "-"
		if err != nil {
			log.Println("failed getting current user")
		}
		usern = userobj.Username

		// get host info for log
		hostname, err := os.Hostname()
		if err != nil {
			log.Println("failed getting machine hostname")
			hostname = "-"
		}

		if *logToFile == "-" { // to stdout
			logger = log.New(os.Stdout, "", 0)
			logger.SetFlags(0)
			logger.SetOutput(logWriter{writer: logger.Writer(), timeFormat: "2006-01-02T15:04:05.999Z07:00", user: usern, hostname: hostname})
		} else { // to file
			// get binary path
			logpath := *logToFile
			dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
			if err != nil {
				log.Println("couldn't get binary path - logfile path is relative to exec dir")
			} else {
				logpath = dir + "/" + *logToFile
			}

			f, err := os.OpenFile(logpath, os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
			if err != nil {
				log.Println("opening file " + *logToFile + " failed, writing log to stdout")
			} else {
				defer f.Close()
				logger = log.New(f, "", 0)
				logger.SetOutput(logWriter{writer: logger.Writer(), timeFormat: "2006-01-02T15:04:05.999Z", user: usern, hostname: hostname})
			}
		}
	}

	logger.Println("starting with conf values - pattern:", patternsArg, "; conf:", *patternConfFile)

	if len(patternsArg) > 0 {
		confStr := ""
		ord := 0
		for _, pat := range patternsArg {
			confStr += `- name: event
  pattern: "` + pat + `"
  order: ` + strconv.Itoa(ord) + ` 
`
			ord++
		}
		//fmt.Println(confStr)
		patternconf = parsePatternConfiguration([]byte(confStr))
	} else {
		patternconf = parsePatternConfigurationFromFile(*patternConfFile)
	}
	
	/*jsondata,_ := json.Marshal(patternconf)
	fmt.Println(string(jsondata))*/

	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		// read line from stdin
		var line = scanner.Text()
		result := make(map[string]string)
		result["data"] = line
		parseLine(result, "")

		jsonresult,_ := json.Marshal(result)
		fmt.Println(string(jsonresult))
	}
}
