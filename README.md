
# ep - event parser

Simple unstructured textual data parser using regex named groups (with grok support). Input data is consumed from stdin, parsed using defined patterns and printed to stdout as json. From there other utils such as `zq` can be used to query the data.

> echo "Mar  7 00:00:00 localhost systemd: Started Session 0001 of user root " | ep | zq -i ndjson -f ndjson "type=auditd | cut srctime,host,program,session_id,user" -  
> {"host":"localhost","pid":"","program":"systemd","session_id":"0001","srctime":"Mar  7 00:00:00","type":"systemd","user":"root"}