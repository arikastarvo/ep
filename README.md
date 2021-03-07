
# ep - event parser

Simple unstructured textual data parser using regex named groups (with grok support). Input data is consumed from stdin, parsed using defined patterns and printed to stdout as json. From there other utils such as `zq` can be used to query the data.
