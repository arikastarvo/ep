package parser

import(
	"encoding/json"
	//"encoding/xml"
	"github.com/clbanning/mxj/v2"
	"gopkg.in/yaml.v3"
	"ep/elog"
	//"log"
	//"fmt"
	"io"
	"io/ioutil"
	"os"
	"sort"
	"strings"
	"github.com/trivago/grok"
	"path/filepath"
	"regexp"
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

// add map key (pat name) to every pattern aso
func (obj *PatternEntry) UnmarshalYAML(unmarshal func(interface{}) error) error {

	logger.Debug("start parsing configuration")

	var aliasObj PatternEntryAlias
	err := unmarshal(&aliasObj)
	if err != nil {
		return err
	}
	
	for key, entry := range aliasObj {

		entry.Name = key
		aliasObj[key] = entry
		logger.Debug("starting to process", key)

		// iterate over children
		// 1) check if any children are references to additional files
		// 2) update parents on other patterns to reflect relationships both ways 
		for _,child := range entry.Children {
			logger.Debug("* processing child", child)
			if _, ok := aliasObj[child]; !ok {
				
				// no such pattern exists (check if it's a file)
				filename := configBasepath + "/" + child
				logger.Debug("no such child yet, checking if it is a file -", filename)
				if _, err := os.Stat(filename); err == nil {
					fileBuf, err := ioutil.ReadFile(filename)
					if err != nil {
						logger.Fatal("sub-config file (", filename, ") read error. Err: ", err)
					}
					
					var nestedChildParser Parser
					logger.Debug("* entering nesting for", child)
					yaml.Unmarshal(fileBuf, &nestedChildParser.Patterns)
					
					tmp := aliasObj[key]

					for nestedChildKey, nestedChildEntry := range nestedChildParser.Patterns {
						if _, ok := aliasObj[nestedChildKey]; ok {
							// just updating parents section of already existing event (only if it is top-level in it's own file context)
							if len(nestedChildEntry.Parent) == 0 && !contains(aliasObj[nestedChildKey].Parent, key) {
								tmp := aliasObj[nestedChildKey]
								tmp.Parent = append(tmp.Parent, key)
								logger.Debug("+ append parent", key, "to", nestedChildKey, ", now", tmp.Parent)
								aliasObj[nestedChildKey] = tmp
								//aliasObj[nestedChildKey].Parent = append(aliasObj[nestedChildKey].Parent, key)
							} else {
								logger.Debug("! not adding parent", key, "to", nestedChildKey, ", now", tmp.Parent)
							}
							
						} else {
							// adding a new event type to the list
							logger.Debug("+ add new entry ", nestedChildKey)

							// if this is the top-level entry for this file, then add parent (event that referenced this file)
							if len(nestedChildEntry.Parent) == 0 {
								nestedChildEntry.Parent = append(nestedChildEntry.Parent, key)
								logger.Debug("+ appending parent", key, "to newly added event", nestedChildKey)
							} else {
								logger.Debug("! not adding parent", key, "to newly added event", nestedChildKey)
							}
							aliasObj[nestedChildKey] = nestedChildEntry
						}
						// append newfound children
						if !contains(tmp.Children, nestedChildKey) {
							tmp.Children = append(tmp.Children, nestedChildKey)
							logger.Debug("+ add child", nestedChildKey, "to", key, ", now", tmp.Children)
						}
					}
					// mark sub event-types from files for later
					tmp.childrenFromReferencedFile = append(tmp.childrenFromReferencedFile, child)
					aliasObj[key] = tmp
					
					logger.Debug("* exit nesting for", child)
				} else {
					logger.Println("no file", filename)
				}
			} else {
				logger.Debug("* child", child, "already exists, not parsing again (just check for XX?)")
				
				// append newfound children
				if !contains(aliasObj[child].Parent, key) {
					tmp := aliasObj[child]
					tmp.Parent = append(tmp.Parent, key)
					logger.Debug("? should we add", key, "as parent to", child)
					aliasObj[child] = tmp
				} else {
					logger.Debug("? ", key, " already parent of", child)
				}
			}
		}


		// PARENTS
		// 1) .. check for file refs
		// 2) update children on other patterns to reflect relationships both ways 
		for _,entryParent := range entry.Parent {
			if _, ok := aliasObj[entryParent]; !ok {
				
				filename := configBasepath + "/" + entryParent
				logger.Debug("no such parent yet, checking if it is a file -", filename)

				if _, err := os.Stat(filename); err == nil {
					fileBuf, err := ioutil.ReadFile(filename)
					if err != nil {
						logger.Fatal("sub-config file (", filename, ") read error. Err: ", err)
					}
					
					var nestedParentParser Parser
					logger.Debug("* entering nesting for", entryParent)
					yaml.Unmarshal(fileBuf, &nestedParentParser.Patterns)
					
					tmp := aliasObj[key]

					for nestedParentKey, nestedParentEntry := range nestedParentParser.Patterns {

						//logger.Debug("checking for nested parent element", nestedParentKey)
						if _, ok := aliasObj[nestedParentKey]; ok {
							// just updating parents section of already existing event (only if it is top-level in it's own file context)
							if len(nestedParentEntry.Parent) == 0 && !contains(aliasObj[nestedParentKey].Parent, key) {
								tmp := aliasObj[nestedParentKey]
								tmp.Parent = append(tmp.Parent, key)
								logger.Debug("+ append parent", key, "to", nestedParentKey, ", now", tmp.Parent)
								aliasObj[nestedParentKey] = tmp
								//aliasObj[nestedParentKey].Parent = append(aliasObj[nestedParentKey].Parent, key)
							} else {
								logger.Debug("! not adding parent", key, "to", nestedParentKey, ", now", tmp.Parent)
							}
							
						} else {
							// adding a new event type to the list
							logger.Debug("+ add new entry ", nestedParentKey)

							// if this is the top-level entry for this file, then add parent (event that referenced this file)
							if len(nestedParentEntry.Parent) == 0 {
								nestedParentEntry.Children = append(nestedParentEntry.Children, key) 
								logger.Debug("+ appending parent", key, "to newly added event", nestedParentKey)
							} else {
								logger.Debug("! not adding parent", key, "to newly added event", nestedParentKey)
							}
							aliasObj[nestedParentKey] = nestedParentEntry
						}
						// append newfound parent
						if !contains(tmp.Children, nestedParentKey) {
							tmp.Parent = append(tmp.Parent, nestedParentKey)
							logger.Debug("+ add child", nestedParentKey, "to", key, ", now", tmp.Parent)
						}
					}
					// mark sub event-types from files for later
					tmp.parentsFromReferencedFile = append(tmp.parentsFromReferencedFile, entryParent)
					aliasObj[key] = tmp
					
					logger.Debug("* exit nesting for", entryParent)
				} else {
					logger.Println("no file", filename)
				}
			} else {
				if !contains(aliasObj[entryParent].Children, key) {
					tmp := aliasObj[entryParent]
					tmp.Children = append(tmp.Children, key)
					logger.Debug("+ adding child", key,"to", entryParent, ", now", tmp.Children)
					aliasObj[entryParent] = tmp
				}
			}
		}

		
		logger.Debug("finished processing", key)
	}
	*obj = PatternEntry(aliasObj)
	
	logger.Debug("finished parsing configuration")

	return nil
}

// pattern def (string or map)
func (a *Pattern) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var patternStruct PatternAlias
	// set default order to maxint
	patternStruct.Order = int(^uint(0) >> 1)
	err := unmarshal(&patternStruct)
	if err != nil {
		var stringArrDfinition StringArray
		err := unmarshal(&stringArrDfinition)
		if err != nil {
			return err
		}
		patternStruct.Pattern = stringArrDfinition
	}

	*a = Pattern(patternStruct)
	return nil
}
// end pattern def

type PatternAlias Pattern
type Pattern struct {
	Name string
	Pattern StringArray
	Json StringArray
	Xml StringArray
	compiledPattern []grok.CompiledGrok
	Optionalpattern StringArray
	optionalCompiledPattern []grok.CompiledGrok
	Grokpattern map[string]string
	Order int
	Keepfield bool
	Field string
	Fields map[string]string
	Parent StringArray
	addedParent StringArray
	parentsFromReferencedFile []string
	Children StringArray
	childrenFromReferencedFile []string
	Cond map[string]string
	compiledCond map[string]grok.CompiledGrok
	Softcond map[string]string
	compiledSoftcond map[string]grok.CompiledGrok
	Replace []map[string]string
	compiledReplace []map[string]interface{}
}

type PatternEntryAlias PatternEntry
type PatternEntry map[string]Pattern

type Parser struct {
	Patterns PatternEntry `,flow`
	sortedIndex []string
}

// parser public methods
func (p Parser) ParseLine(line string) map[string]interface{} {
	//return map[string]interface{}{"key": line}
	result := make(map[string]interface{})
	result["data"] = line
	p.parseLineInternal(result, "")
	return result
}

func (p Parser) ParseLineWithMetadata(line string, result map[string]interface{}) map[string]interface{} {
	//return map[string]interface{}{"key": line}
	//result := make(map[string]interface{})
	result["data"] = line
	p.parseLineInternal(result, "")
	return result
}

func (p Parser) parseLineInternal(result map[string]interface{}, parent string) {

	// iterate over patterns in sorted order
	for _,patKey := range p.sortedIndex {

		pat := p.Patterns[patKey]


		// if parent event does not match, then skip this one
		if (parent != "" && !contains(pat.Parent, parent)) || (parent == "" && len(pat.Parent) > 0) {
			continue
		}

		//logger.Debug("Evaluating pattern", pat.Name)

		// set the base field
		field := "data"
		if pat.Field != "" {
			field = pat.Field
		}

		skip := false

		// check if hard conditions are met
		if len(pat.compiledCond) > 0 {
			for condField, condCompiledPattern := range pat.compiledCond {
				value, ok := result[condField]

				if ok { // we have this field, so we check for match

					if strValue, ok := value.(string); ok && ! condCompiledPattern.MatchString(strValue) {
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
		if ! skip && len(pat.compiledSoftcond) > 0 {
			for condField, condCompiledPattern := range pat.compiledSoftcond {
				value, ok := result[condField]

				if ok { // we have this field, so we check for match
					if strValue, ok := value.(string); ok &&  ! condCompiledPattern.MatchString(strValue) {
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

		for _, compiledPat := range pat.compiledPattern {
			if fieldValue, ok := result[field].(string); ok {

				match = compiledPat.ParseString(fieldValue)

				// if we have a match (captured values), then gather results, optionally parse child patterns and finally break look 
				if len(match) > 0 {
					// break out of the pattern matching loop
					break
				}
			}
		}
		

		// if we have a match or there are no patterns to match (we interpret that as a whole line match) .. then do some stuff

		if len(match) > 0 || len(pat.compiledPattern) == 0 {
			
			// hackish solution for json/xml (match has to have the field)
			if len(pat.compiledPattern) == 0 {
				match = make(map[string]string)
				match[field] = result[field].(string)
			}

			// if we have any json fields, then try to parse them here
			if len(pat.Json) > 0 {
				for _, jsonField := range pat.Json {
					if fieldValue, ok := match[jsonField]; ok {
						if err := json.Unmarshal([]byte(fieldValue), &result); err != nil {
							// json parsing failed, skip this pattern and try luck with the next one
							break
						} else {
							delete(match, jsonField)
						}
					}
				}
			}
			// end json use-case

			// if we have a xml, then convert it here
			if len(pat.Xml) > 0 {
				for _, xmlField := range pat.Xml {
					if fieldValue, ok := match[xmlField]; ok {
						//if err := xml.Unmarshal([]byte(fieldValue), &result); err != nil {
						if resultMap, err := mxj.NewMapXml([]byte(fieldValue)); err != nil {
							// xml parsing failed, skip this pattern and try luck with the next one
							//logger.Debug("xml parse err:", err)
							break
						} else {
							for k,v := range resultMap {
								result[k] = v
							}
							delete(match, xmlField)
						}
					}
				}
			}
			// end xml use-case

			result["event_type"] = pat.Name
			if pathValue, ok := result["event_type_path"].(string); ok {
				result["event_type_path"] = pathValue + "/" + pat.Name
			} else {
				result["event_type_path"] = pat.Name
			}

			// delete source field if not stated otherwise
			value := result[field]
			if ! pat.Keepfield {
				delete(result, field)
			}

			// handle the case when we don't have matches because we dont have patterns
			if len(pat.compiledPattern) == 0 {

				// only add the field back if it's not part of json/xml .. in that case that should be eliminated (actually --- only when json/xml parsing succeeds, so there's a problem..)
				if !contains(pat.Json, field) && !contains(pat.Xml, field) {
					result[field] = value
				}
			} else {
				// put data to results object
				for k,v := range match {
					result[k] = v
				}
			}

			// execute optionalpattern matches
			if len(pat.optionalCompiledPattern) > 0 {
				for _, optionalCompiledPatternItem := range pat.optionalCompiledPattern {
					if strValue, ok := value.(string); ok {
						optMatch := optionalCompiledPatternItem.ParseString(strValue)
						for k,v := range optMatch {
							result[k] = v
						}
					}
				}
			}

			// execute replace definitions
			if len(pat.compiledReplace) > 0 {
				for _, compiledReplaceDefinition := range pat.compiledReplace {
					if fieldValue, ok := result[compiledReplaceDefinition["field"].(string)].(string); ok {
						var pat regexp.Regexp
						pat = compiledReplaceDefinition["pattern"].(regexp.Regexp)
						s := pat.ReplaceAllString(fieldValue, compiledReplaceDefinition["replace"].(string))
						result[compiledReplaceDefinition["field"].(string)] = s
					}
				}
			}

			// parse child patterns if there exists any
			if len(pat.Children) > 0 {
				p.parseLineInternal(result, pat.Name)
			}

			// after a sucessful match, break out of this event type (don't try to match other patterns or sibling event types)
			//break out
			break
		}
	}
	
}

func (p Parser) PrettyPrintPatterns() (){
	p.prettyPrintPatternsRecursive(os.Stdout, "", 0)
}

// parser private methods

func (p *Parser) reverseMapInheritence() {
}

func (p *Parser) preCompilePatterns() {

	// pre-compile various patterns (using grok)
	for i, patConf := range p.Patterns {
		tmpPattern := p.Patterns[i]

		if tmpPattern.Fields == nil {
			tmpPattern.Fields = make(map[string]string)
		}

		// add default grok patterns to all
		if (patConf.Grokpattern == nil) {
			tmpPattern.Grokpattern = make(map[string]string)
		}

		tmpPattern.Grokpattern["GD"] = ".*"
		// end add default groks
		g, err := grok.New(grok.Config{Patterns: patConf.Grokpattern, NamedCapturesOnly: true})
		if err != nil {
			logger.Println("could not create grok parser for ", patConf.Name, ". Err: ", err)
			continue
		}

		// catch special field types before they are overridden by grok
		for fieldName, fieldType := range tmpPattern.Fields {
			if fieldType == "json" && !contains(tmpPattern.Json, fieldName) {
				tmpPattern.Json = append(tmpPattern.Json, fieldName)
			}
			
			if fieldType == "xml" && !contains(tmpPattern.Xml, fieldName) {
				tmpPattern.Xml = append(tmpPattern.Xml, fieldName)
			}
		}

		// main patterns
		for _, pat := range patConf.Pattern {
			cg,err := g.Compile(pat)
			if err != nil {
				logger.Println("err: ", err)
				continue
			}
			tmpPattern.compiledPattern = append(tmpPattern.compiledPattern, *cg)
			for _, fieldName := range cg.GetFields() {
				if fieldName != "" {
					tmpPattern.Fields[fieldName] = "string"
				}
			}
		}

		// optional patterns
		for _, pat := range patConf.Optionalpattern {
			cg,err := g.Compile(pat)
			if err != nil {
				logger.Println("err: ", err)
				continue
			}
			tmpPattern.optionalCompiledPattern = append(tmpPattern.optionalCompiledPattern, *cg)
		}

		// condition patterns
		tmpPattern.compiledCond = make(map[string]grok.CompiledGrok)
		for field, pat := range patConf.Cond {
			cg,err := g.Compile(pat)
			if err != nil {
				logger.Println("err: ", err)
				continue
			}
			tmpPattern.compiledCond[field] = *cg
		}

		// soft condition patterns
		tmpPattern.compiledSoftcond = make(map[string]grok.CompiledGrok)
		for field, pat := range patConf.Softcond {
			cg,err := g.Compile(pat)
			if err != nil {
				logger.Println("err: ", err)
				continue
			}
			tmpPattern.compiledSoftcond[field] = *cg
		}

		// replace patterns
		for _, replaceDefinition := range patConf.Replace {
			//cg,err := g.Compile(replaceDefinition["pattern"])
			cg := regexp.MustCompile(replaceDefinition["pattern"])
			
			compiledReplaceDefinition := make(map[string]interface{})
			compiledReplaceDefinition["field"] = replaceDefinition["field"]
			compiledReplaceDefinition["replace"] = replaceDefinition["replace"]
			compiledReplaceDefinition["pattern"] = *cg
			tmpPattern.compiledReplace = append(tmpPattern.compiledReplace, compiledReplaceDefinition)
		}

		p.Patterns[i] = tmpPattern
	}
}

func (p *Parser) generateSortedIndex() {
	p.sortedIndex = make([]string, len(p.Patterns))
	i:=0
	for key,_ := range p.Patterns {
		p.sortedIndex[i] = key
		i++
	}
	
	sort.SliceStable(p.sortedIndex, func(i, j int) bool {
		return p.Patterns[p.sortedIndex[i]].Order < p.Patterns[p.sortedIndex[j]].Order
	})
}

func (p Parser) prettyPrintPatternsRecursive(output io.Writer, parent string, level int) (){
    
	for _,patKey := range p.sortedIndex {
	// for _, pat := range parsedconf.Patterns {
		pat := p.Patterns[patKey]
        if (len(parent) == 0 && len(pat.Parent) == 0) || (len(parent) > 0 && contains(pat.Parent, parent)) {
            //fmt.Print()
            //logger.Println(strings.Repeat("+", level), pat.Name)
			prefix := ""
			if level > 0 {
				prefix = strings.Repeat("+", level) + " "
			}
			output.Write([]byte(prefix + pat.Name + "\n"))
            if len(pat.Children) > 0 {
                p.prettyPrintPatternsRecursive(output, pat.Name, (level + 1))
            }
        }

    }
}
// end parser methods

// general helper methods

func contains(s []string, e string) bool {
    for _, a := range s {
        if a == e {
            return true
        }
    }
    return false
}

func index(slice []string, x string) int {
    for i, n := range slice {
        if x == n {
            return i
        }
    }
    return -1
}

func removeByValue(slice []string, value string) []string {
	idx := index(slice, value)
	if idx < 0 {
		logger.Println("VALUE", value,"not within", slice)
		return slice
	} else {
    	return append(slice[:idx], slice[idx+1:]...)
/*
		// Remove the element at index i from a.
		copy(slice[idx:], slice[idx+1:]) 	// Shift a[i+1:] left one index.
		slice[len(slice)-1] = ""     		// Erase last element (write zero value).
		slice = slice[:len(slice)-1]     	// Truncate slice.
		return slice*/
	}
}

func remove(slice []string, s int) []string {
    return append(slice[:s], slice[s+1:]...)
}

// end helper methods

var logger elog.ELogger
var configBasepath = "."

func SetELogger(output string, debug bool) {
	logger = elog.GetELogger(output, "ep/parser", debug)
}

func init() {
	SetELogger("-", false)
}

func ParserFromFile(configFile string) Parser {
	fileBuf, err := ioutil.ReadFile(configFile)
	if err != nil {
		logger.Println("config file (", configFile, ") read error. Err: ", err)
	}
	return ParserFromBytesWithConfigBasepath(fileBuf, filepath.Dir(configFile))
}

func ParserFromBytes(data []byte) Parser {
	return ParserFromBytesWithConfigBasepath(data, ".")
}

func ParserFromBytesWithConfigBasepath(data []byte, ConfigBasepath string) Parser {
	var p Parser
	configBasepath = ConfigBasepath
	yaml.Unmarshal(data, &p.Patterns)
	p.preCompilePatterns()
	p.generateSortedIndex()
	return p
}