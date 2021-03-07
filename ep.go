package main

import(
	//"regexp"
	"fmt"
	//"log"
	"os"
	"bufio"
	"io/ioutil"
	"gopkg.in/yaml.v3"
	"sort"
	"encoding/json"
	//"github.com/vjeantet/grok"
	"github.com/trivago/grok"
)

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

func parsePatternConfiguration(configFile string) (conf) {

	var parsedconf conf

	yamlBuf, err := ioutil.ReadFile(configFile)
	if err != nil {
		fmt.Printf("yamlFile.Get err   #%v ", err)
	}

	yaml.Unmarshal(yamlBuf, &parsedconf.Patterns)
	sort.SliceStable(parsedconf.Patterns, func(i, j int) bool {
		return parsedconf.Patterns[i].Order < parsedconf.Patterns[j].Order
	})

	// map children	
	for i, pat := range parsedconf.Patterns {
		for _, subpat := range parsedconf.Patterns {
			if contains(subpat.Parent, pat.Name) {
				parsedconf.Patterns[i].Children = append(parsedconf.Patterns[i].Children, subpat.Name)
			}
		}
	}

	// pre-compile various patterns (using grok)
	for i, patConf := range parsedconf.Patterns {
		g, err := grok.New(grok.Config{Patterns: patConf.Grokpattern, NamedCapturesOnly: true})
		if err != nil {
			fmt.Println("err: ", err)
			continue
		}

		// main patterns
		for _, pat := range patConf.Pattern {
			cg,err := g.Compile(pat)
			if err != nil {
				fmt.Println("err: ", err)
				continue
			}
			parsedconf.Patterns[i].CompiledPattern = append(parsedconf.Patterns[i].CompiledPattern, *cg)
		}

		// optional patterns
		for _, pat := range patConf.Optionalpattern {
			cg,err := g.Compile(pat)
			if err != nil {
				fmt.Println("err: ", err)
				continue
			}
			parsedconf.Patterns[i].OptionalCompiledPattern = append(parsedconf.Patterns[i].OptionalCompiledPattern, *cg)
		}

		// condition patterns
		parsedconf.Patterns[i].CompiledCond = make(map[string]grok.CompiledGrok)
		for field, pat := range patConf.Cond {
			cg,err := g.Compile(pat)
			if err != nil {
				fmt.Println("err: ", err)
				continue
			}
			parsedconf.Patterns[i].CompiledCond[field] = *cg
		}

		// soft condition patterns
		parsedconf.Patterns[i].CompiledSoftcond = make(map[string]grok.CompiledGrok)
		for field, pat := range patConf.Softcond {
			cg,err := g.Compile(pat)
			if err != nil {
				fmt.Println("err: ", err)
				continue
			}
			parsedconf.Patterns[i].CompiledSoftcond[field] = *cg
		}
	}

	return parsedconf
}

func main() {

	patternconf = parsePatternConfiguration("patterns.yaml")

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
