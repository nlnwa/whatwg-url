package url

var defaultParser = newDefaultParser()

func Parse(rawUrl string) (*Url, error) {
	return defaultParser.Parse(rawUrl)
}

func ParseRef(rawUrl, ref string) (*Url, error) {
	return defaultParser.ParseRef(rawUrl, ref)
}
