package flexibleconfig

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/devopsfaith/krakend/config"
)

type Config struct {
	Settings string
	Partials string
	Parser   config.Parser
}

func NewTemplateParser(cfg Config) *TemplateParser {
	t := &TemplateParser{
		Partials: cfg.Partials,
		Parser:   cfg.Parser,
		Vars:     map[string]interface{}{},
	}
	if cfg.Settings != "" {
		files, err := ioutil.ReadDir(cfg.Settings)
		if err != nil {
			fmt.Println("error reading settings folder:", cfg.Settings, err)
			files = []os.FileInfo{}
		}
		for _, settingsFile := range files {
			b, err := ioutil.ReadFile(filepath.Join(cfg.Settings, settingsFile.Name()))
			if err != nil {
				fmt.Println("error procesing settings:", settingsFile, err)
				continue
			}
			var v map[string]interface{}
			if err := json.Unmarshal(b, &v); err != nil {
				fmt.Println("error procesing settings:", settingsFile, err)
				continue
			}
			t.Vars[strings.TrimRight(filepath.Base(settingsFile.Name()), ".json")] = v
		}
	}
	return t
}

type TemplateParser struct {
	Vars     map[string]interface{}
	Partials string
	Parser   config.Parser
}

func (t *TemplateParser) Parse(configFile string) (config.ServiceConfig, error) {
	tmpfile, err := ioutil.TempFile("", "KrakenD_parsed_config_template_")
	if err != nil {
		log.Fatal("creating the tmp file:", err)
	}

	defer os.Remove(tmpfile.Name())

	var buf bytes.Buffer

	tmpl, err := t.newConfigTemplate().ParseFiles(configFile)
	if err != nil {
		log.Fatal("parsing files:", err)
		return t.Parser.Parse(configFile)
	}
	err = tmpl.ExecuteTemplate(&buf, filepath.Base(configFile), t.Vars)
	if err != nil {
		log.Fatal("executing template:", err)
		return t.Parser.Parse(configFile)
	}

	if _, err = tmpfile.Write(buf.Bytes()); err != nil {
		log.Fatal("writting the tmp config:", err)
		return t.Parser.Parse(configFile)
	}
	if err = tmpfile.Close(); err != nil {
		log.Fatal("closing the tmp config:", err)
	}

	filename := tmpfile.Name() + ".json"
	if err := os.Rename(tmpfile.Name(), filename); err != nil {
		return config.ServiceConfig{}, err
	}

	return t.Parser.Parse(filename)
}

func (t *TemplateParser) newConfigTemplate() *template.Template {
	return template.New("config").Funcs(template.FuncMap{
		"marshal": t.marshal,
		"include": t.include,
	})
}

func (t *TemplateParser) marshal(v interface{}) string {
	a, _ := json.Marshal(v)
	return string(a)
}

func (t *TemplateParser) include(v interface{}) string {
	a, _ := ioutil.ReadFile(path.Join(t.Partials, v.(string)))
	return string(a)
}
