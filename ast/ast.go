package ast

type File struct {
	Imports  []Import
	Exports  []Export
	Entries  []Entry
	Comments []Comment
}

type Import struct {
	Path     string
	Alias    string
	Position Position
}

type Export struct {
	Name     string
	Value    string
	Position Position
}

type Entry struct {
	Request  *Request
	Response *Response
	Comments []Comment
}

type Request struct {
	Method    string
	URL       string
	Headers   []Header
	Options   *OptionsSection
	Query     []KeyValue
	Form      []KeyValue
	Multipart []MultipartField
	Cookies   []KeyValue
	BasicAuth *BasicAuth
	Body      *Body
}

type Response struct {
	Version  string
	Status   int
	Headers  []Header
	Captures []Capture
	Asserts  []Assert
	Body     *Body
}

type Header struct {
	Name  string
	Value string
}

type KeyValue struct {
	Key   string
	Value string
}

type MultipartField struct {
	Name     string
	Value    string
	IsFile   bool
	FileType string
}

type BasicAuth struct {
	Username string
	Password string
}

type Body struct {
	Type    BodyType
	Content string
	Lang    string
}

type BodyType int

const (
	BodyNone BodyType = iota
	BodyJSON
	BodyXML
	BodyMultiline
	BodyOneline
	BodyBase64
	BodyHex
	BodyFile
)

type OptionsSection struct {
	Location       *bool
	MaxRedirs      *int
	Insecure       *bool
	Verbose        *bool
	Compressed     *bool
	FollowRedirect *bool
	Retry          *int
	RetryInterval  string
	Timeout        string
	ConnectTimeout string
	Delay          string
	Skip           *bool
	Output         string
	Variables      map[string]string
	Headers        map[string]string
	Proxy          string
	User           string
	UserAgent      string
	HTTP3          *bool
	CACert         string
	Cert           string
	Key            string
	AWSSigV4       string
	IPv4           *bool
	IPv6           *bool
	LimitRate      *int64
	PathAsIs       *bool
	UnixSocket     string
}

type Capture struct {
	Variable string
	Query    Query
	Filters  []Filter
	Redact   bool
}

type Assert struct {
	Query     Query
	Not       bool
	Predicate PredicateType
	Value     AssertValue
	Filters   []Filter
}

type PredicateType int

const (
	PredEqual PredicateType = iota
	PredNotEqual
	PredGreaterThan
	PredGreaterEqual
	PredLessThan
	PredLessEqual
	PredStartsWith
	PredEndsWith
	PredContains
	PredMatches
	PredExists
	PredIsBoolean
	PredIsEmpty
	PredIsFloat
	PredIsInteger
	PredIsIpv4
	PredIsIpv6
	PredIsIsoDate
	PredIsList
	PredIsNumber
	PredIsObject
	PredIsString
	PredIsUuid
	PredIncludes
	PredIsCollection
	PredIsDate
)

type QueryType int

const (
	QueryStatus QueryType = iota
	QueryVersion
	QueryHeader
	QueryCookie
	QueryBody
	QueryBytes
	QueryXPath
	QueryJSONPath
	QueryRegex
	QuerySHA256
	QueryMD5
	QueryURL
	QueryRedirects
	QueryIP
	QueryVariable
	QueryDuration
	QueryCertificate
)

type Query struct {
	Type  QueryType
	Value string
}

type Filter struct {
	Type  FilterType
	Value string
}

type FilterType int

const (
	FilterCount FilterType = iota
	FilterRegex
	FilterReplace
	FilterReplaceRegex
	FilterSplit
	FilterNth
	FilterFirst
	FilterLast
	FilterToInt
	FilterToFloat
	FilterToString
	FilterToDate
	FilterDateFormat
	FilterDaysAfterNow
	FilterDaysBeforeNow
	FilterBase64Decode
	FilterBase64Encode
	FilterBase64UrlSafeDecode
	FilterBase64UrlSafeEncode
	FilterDecode
	FilterUrlDecode
	FilterUrlEncode
	FilterUrlQueryParam
	FilterHtmlEscape
	FilterHtmlUnescape
	FilterToHex
	FilterUtf8Decode
	FilterUtf8Encode
	FilterXPath
	FilterJSONPath
	FilterLocation
	FilterUpper
	FilterLower
)

type AssertValue struct {
	Type   AssertValueType
	Str    string
	Int    int64
	Float  float64
	Bool   bool
	Bytes  []byte
	IsNull bool
}

type AssertValueType int

const (
	ValueString AssertValueType = iota
	ValueInt
	ValueFloat
	ValueBool
	ValueBytes
	ValueNull
)

type Comment struct {
	Text     string
	Position Position
}

type Position struct {
	Line int
	Col  int
}
