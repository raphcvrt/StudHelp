package handlers

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
)

var Templates *template.Template

func LoadTemplates(templateDir string) error {
	// Ajout de la fonction truncate dans FuncMap
	funcMap := template.FuncMap{
		"isset": func(data interface{}, key string) bool {
			m, ok := data.(map[string]interface{})
			return ok && m[key] != nil
		},
		"truncate": func(text string, length int) string {
			if len(text) <= length {
				return text
			}
			return text[:length] + "...	"
		},
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
			return a - b
		},
		"seq": func(start, end int) []int {
			var result []int
			for i := start; i <= end; i++ {
				result = append(result, i)
			}
			return result
		},
	}

	templates := template.New("").Funcs(funcMap)

	pattern := filepath.Join(templateDir, "*.html")
	_, err := templates.ParseGlob(pattern)
	if err != nil {
		return fmt.Errorf("error parsing templates: %v", err)
	}

	Templates = templates
	return nil
}

func RenderTemplate(w http.ResponseWriter, tmpl string, data interface{}) error {
	if Templates == nil {
		return fmt.Errorf("templates not initialized")
	}

	// Create a new map to hold all template data
	templateData := make(map[string]interface{})

	// If the input data is already a map, merge it
	if dataMap, ok := data.(map[string]interface{}); ok {
		for k, v := range dataMap {
			templateData[k] = v
		}
	}

	// Always set the ContentTemplate
	templateData["ContentTemplate"] = tmpl

	return Templates.ExecuteTemplate(w, "base.html", templateData)
}
