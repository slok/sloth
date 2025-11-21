package ui

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"io/fs"
	"maps"
	"net/http"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/slok/sloth/internal/log"
)

var (
	//go:embed all:static
	staticFS embed.FS
	//go:embed all:templates
	templatesFS embed.FS
)

// tplRenderer is a util that will make rendering templates easier and standardize inside the server.
type tplRenderer struct {
	logger log.Logger
	tpls   *template.Template

	// Extra data.
	// This data will be available on all templates as `Common.{KEY}`.
	CommonData map[string]any
}

var allowedTemplateExtensions = map[string]struct{}{
	".html": {},
	".tpl":  {},
	".tmpl": {},
}

func newTplRenderer(logger log.Logger) (*tplRenderer, error) {
	// Discover all template directories to parse.
	templatePaths := []string{}
	err := fs.WalkDir(templatesFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Only add files with allowed extensions.
		extension := strings.ToLower(filepath.Ext(path))
		if _, ok := allowedTemplateExtensions[extension]; ok {
			templatePaths = append(templatePaths, path)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("could not discover template paths: %w", err)
	}

	// Parse all templates.
	templates, err := template.New("base").Funcs(template.FuncMap{
		"prettyPercent":              prettyPercent,
		"prettyPercentFixed":         prettyPercentFixed,
		"percentUpColorCSSClass":     PercentUpColorCSSClass,
		"percentDownColorCSSClass":   PercentDownColorCSSClass,
		"percentColorCSSClassCustom": PercentColorCSSClassCustom,
		"slothLogoSVG":               SlothLogoSVG,
	}).ParseFS(templatesFS, templatePaths...)
	if err != nil {
		return nil, fmt.Errorf("could not parse templates: %w", err)
	}

	return &tplRenderer{
		logger: logger,
		tpls:   templates,
		CommonData: map[string]any{
			"CSSPath":     urls.NonAppURL(staticPrefix + "/css"),
			"JSPath":      urls.NonAppURL(staticPrefix + "/js"),
			"ImagePath":   urls.NonAppURL(staticPrefix + "/img"),
			"HomeURL":     urls.AppURL("/"),
			"ServicesURL": urls.AppURL("/services"),
			"SLOsURL":     urls.AppURL("/slos"),
		},
	}, nil
}

func (t *tplRenderer) withCtxData(ctx context.Context) *tplRenderer {
	c := maps.Clone(t.CommonData)

	return &tplRenderer{
		logger:     t.logger,
		tpls:       t.tpls,
		CommonData: c,
	}
}

func (t *tplRenderer) WithRequestData(r *http.Request) *tplRenderer {
	c := maps.Clone(t.CommonData)

	return &tplRenderer{
		logger:     t.logger,
		tpls:       t.tpls,
		CommonData: c,
	}
}

func (t *tplRenderer) RenderResponse(ctx context.Context, w http.ResponseWriter, r *http.Request, tplName string, data any) {
	renderer := t.withCtxData(ctx).WithRequestData(r)

	d := struct {
		Common map[string]any
		Data   any
	}{
		Common: renderer.CommonData,
		Data:   data,
	}
	err := renderer.tpls.ExecuteTemplate(w, tplName, d)
	if err != nil {
		t.logger.Errorf("Could not render template: %s", err)
		// TODO(slok): Render 500 template.
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (t *tplRenderer) Render(ctx context.Context, r *http.Request, tplName string, data any) (string, error) {
	renderer := t.withCtxData(ctx).WithRequestData(r)

	d := struct {
		Common map[string]any
		Data   any
	}{
		Common: renderer.CommonData,
		Data:   data,
	}
	var b bytes.Buffer
	err := renderer.tpls.ExecuteTemplate(&b, tplName, d)
	if err != nil {
		return "", err
	}

	return b.String(), nil
}
