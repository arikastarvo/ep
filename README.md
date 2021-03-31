
# ep - event parser

Simple unstructured textual data parser using regex named groups (with grok support). Input data is consumed from stdin, parsed using defined patterns and printed to stdout as json. From there other utils such as `brimsec/zq`, `endgameinc/eql` or `stedolan/jq` can be used to query the data.

**use it at your own risk !**

```text
Usage of ep:
  -conf string
        set patterns file (default "patterns.yaml")
  -log string
        enable logging. "-" for stdout, filename otherwise
  -p string
        short version of -pattern
  -pattern string
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