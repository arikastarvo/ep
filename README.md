
# ep - event parser

Simple unstructured textual data parser using regex named groups (with grok support). Input data is consumed from stdin, parsed using defined patterns and printed to stdout as json. From there other utils such as `brimsec/zq`, `endgameinc/eql` or `stedolan/jq` can be used to query the data.

Can be used to parse existing files (piping contents to **ep** using **cat**), but primary focus is streaming content (from **tail -f** or other long-lived processes that generate unbounded data streams). Simple parsing patterns can be defined using command line parameters (-p/-pattern params). More complex pattern setups that provide option to differentiate different event types, nesting etc can be configured via yaml configuration files.

## configuration files
Configuration files are yaml files with map types. Suppored elements:
```yaml
{event-name}:                       # name of the event (field event_type), if pattern matches
  pattern: {ep-pattern}             # string|list, as long as one of these patterns match, this EP event type is a match
  optionalpattern: {ep-ppattern}    # string|list, if ${pattern} matches, optional patterns are used to extract additional data
  grokpattern: {grok-pattern}       # map, used to define additional grok patterns for this ep pattern (these grok patterns can be used )
  order: 0                          # int, defines parsing order
  field: {fieldname}                # string, use patterns on this field (default: data)
  keepfield: true                   # bool, can be used to exclude initial ${field} from results
  cond: {conditions}                # map, key/value pairs where key is field name and value is ep-pattern. ${pattern} is evaluated if all these conditions are satisfied (condition patterns match)
  softcond: {soft-conditions}       # map, key/value pair similar to ${cond}. only these fields are matched that exist.
  parent: {parents}                 # string|list, parent event type name(s)
  children: {children}              # string|list, child event type name(s) or filename(s) of another conf file (event types from there are considered as child event types)
```

If no additional configuration besides one pattern is required, shorthand notation can be used (string or list of strings):
```yaml
{event-name}: {ep-pattern}
{event-name}:
  - {ep-pattern}
  - {ep-pattern}
```

### ep-pattern
... is for matching. It can be pure regex with named groups or mixed with grok patterns. examples:
```
^(?P<data>.*)$                      will match all data to field 'data'
^%{GREEDYDATA:data}$                does exactly the same thing as previous pattern
^%{NUMBER:num}(?P<data>.*)$         will match all lines starting with an integer and followed by any arbitary data
```

## examples

This configuration will match all input lines as event type `event`. For lines with the value of an integer, input data will be set to field `int`, for other numerical values field `num` will be used. All other values will be stored to `data`. Patterns order is cruitial here as pattern matching will be evaluated in the order they were defined. This configuration is equal to using command line parameters `ep -p "^%{INT:int}$" -p "^%{NUMBER:num}$" -p "^%{GREEDYDATA:data}$"`
```yaml
event:
  - ^%{INT:int}$
  - ^%{NUMBER:num}$
  - ^%{GREEDYDATA:data}$
  
# in: 3
# out: {"event_type":"event","event_type_path":"/event","int":"3"}

# in: 4.5
# out: {"event_type":"event","event_type_path":"/event","num":"4.5"}

# in: foo
# out: {"data":"foo","event_type":"event","event_type_path":"/event"}
```

Following configuration will also capture all input lines, but different matches will be stored with different event type names. 
```yaml
int-event: ^%{INT:int}$
num-event: ^%{NUMBER:num}$
data-event: ^%{GREEDYDATA:data}$

# in: 3
# out: {"event_type":"int-event","event_type_path":"/int-event","int":"3"}

# in: 4.5
# out: {"event_type":"num-event","event_type_path":"/num-event","num":"4.5"}

# in: foo
# out: {"data":"foo","event_type":"data-event","event_type_path":"/data-event"}
```

Using inheritence
```yaml
number: ^%{NUMBER:num}$
int:
  pattern: ^%{INT:int}$
  parent: number
  field: num
event: ^%{GREEDYDATA:data}$

# in: 34
# out: {"event_type":"int","event_type_path":"/number/int","int":"34"}

# in: 45.3
# out: {"event_type":"number","event_type_path":"/number","num":"45.3"}
```
## usage

**use it at your own risk !**

```text
Usage of ep:
  -conf string
        set patterns file (default "patterns.yaml")
  -debug
        enable deug logging.
  -log string
        enable logging. "-" for stdout, filename otherwise
  -os
        output pattern conf (short format)
  -p value
        short version of -pattern
  -pattern value
        set pattern inline (if set, this is used instead of -conf)
```

**ep** usage
```bash
$ echo "Mar  7 00:00:00 localhost systemd: Started Session 0001 of user root " | ep

{"action":"Started Session","data":" Started Session 0001 of user root ","event_type":"systemd","host":"localhost","pid":"","program":"systemd","session_id":"0001","srctime":"Mar  7 00:00:00","type":"systemd","user":"root"}
```

**ep** with **brimsec/zq**
```bash
echo "Mar  7 00:00:00 localhost systemd: Started Session 0001 of user root " | ep | zq -i ndjson -f ndjson "event_type=systemd user=root | cut event_type,srctime,host,program,session_id,user,action" -

{"action":"Started Session","event_type":"systemd","host":"localhost","program":"systemd","session_id":"0001","srctime":"Mar  7 00:00:00","user":"root"}
```
**ep** with **endgameinc/eql**
```bash
$ echo "Mar  7 00:00:00 localhost systemd: Started Session 0001 of user root " | ep | eql query "systemd where user == 'root'"

{"action": "Started Session", "data": " Started Session 0001 of user root ", "event_type": "systemd", "host": "localhost", "pid": "", "program": "systemd", "session_id": "0001", "srctime": "Mar  7 00:00:00", "type": "systemd", "user": "root"}
```

**ep** with **stedolan/jq**
```bash
$ echo "Mar  7 00:00:00 localhost systemd: Started Session 0001 of user root " | ep | jq -c '. | select(.event_type=="systemd" and .user=="root") | {event_type, srctime, host, program, session_id, user, action }'

{"event_type":"systemd","srctime":"Mar  7 00:00:00","host":"localhost","program":"systemd","session_id":"0001","user":"root","action":"Started Session"}
```
