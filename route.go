package dynamic_proxy

import (
	"regexp"
	"strings"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/lemon-1997/dynamic-proxy/httprule"
)

var encodedPathSplitter = regexp.MustCompile("(/|%2F)")

type Router interface {
	Add(method, path string, extra interface{}) error
	Match(method, path string) (map[string]string, interface{}, bool)
}

type Pattern struct {
	runtime.Pattern
	extra interface{}
}

type httpRouter struct {
	unescapeMode runtime.UnescapingMode
	patterns     map[string][]Pattern
}

func NewRouter() Router {
	return &httpRouter{
		unescapeMode: runtime.UnescapingModeDefault,
		patterns:     make(map[string][]Pattern),
	}
}

func (r *httpRouter) Add(method, path string, extra interface{}) error {
	c, err := httprule.Parse(path)
	if err != nil {
		return err
	}
	tmpl := c.Compile()
	p, err := runtime.NewPattern(tmpl.Version, tmpl.OpCodes, tmpl.Pool, tmpl.Verb)
	if err != nil {
		return err
	}
	r.patterns[method] = append(r.patterns[method], Pattern{
		Pattern: p,
		extra:   extra,
	})
	return nil
}

func (r *httpRouter) Match(method, path string) (map[string]string, interface{}, bool) {
	if r == nil {
		return nil, nil, false
	}
	if !strings.HasPrefix(path, "/") {
		return nil, nil, false
	}
	var pathComponents []string
	pathComponents = strings.Split(path[1:], "/")

	if r.unescapeMode == runtime.UnescapingModeAllCharacters {
		pathComponents = encodedPathSplitter.Split(path[1:], -1)
	} else {
		pathComponents = strings.Split(path[1:], "/")
	}

	lastPathComponent := pathComponents[len(pathComponents)-1]
	patterns := r.patterns[method]
	for _, item := range patterns {
		var verb string
		patVerb := item.Verb()

		idx := -1
		if patVerb != "" && strings.HasSuffix(lastPathComponent, ":"+patVerb) {
			idx = len(lastPathComponent) - len(patVerb) - 1
		}
		if idx == 0 {
			return nil, nil, false
		}

		comps := make([]string, len(pathComponents))
		copy(comps, pathComponents)

		if idx > 0 {
			comps[len(comps)-1], verb = lastPathComponent[:idx], lastPathComponent[idx+1:]
		}
		pathParams, err := item.MatchAndEscape(comps, verb, r.unescapeMode)
		if err != nil {
			continue
		}
		return pathParams, item.extra, true
	}
	return nil, nil, false
}
