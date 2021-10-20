package flexibleconfig

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/luraproject/lura/v2/config"
)

type Config struct {
	Settings  string
	Partials  string
	Templates string
	Parser    config.Parser
	Path      string
}

func NewTemplateParser(cfg Config) *TemplateParser {
	t := &TemplateParser{
		Partials:  cfg.Partials,
		Templates: []string{},
		Parser:    cfg.Parser,
		Vars:      map[string]interface{}{},
		Path:      cfg.Path,
		err:       parserError{errors: map[string]error{}},
	}

	if cfg.Settings != "" {
		files, err := ioutil.ReadDir(cfg.Settings)
		if err != nil {
			t.err.errors[cfg.Settings] = err
			files = []os.FileInfo{}
		}
		for _, settingsFile := range files {
			if !strings.HasSuffix(settingsFile.Name(), ".json") {
				continue
			}
			b, err := ioutil.ReadFile(filepath.Join(cfg.Settings, settingsFile.Name()))
			if err != nil {
				t.err.errors[settingsFile.Name()] = err
				continue
			}
			var v map[string]interface{}
			if err := json.Unmarshal(b, &v); err != nil {
				t.err.errors[settingsFile.Name()] = err
				continue
			}
			t.Vars[strings.TrimSuffix(filepath.Base(settingsFile.Name()), ".json")] = v
		}
	}

	if cfg.Templates != "" {
		files, err := ioutil.ReadDir(cfg.Templates)
		if err != nil {
			t.err.errors[cfg.Templates] = err
			files = []os.FileInfo{}
		}
		for _, settingsFile := range files {
			if !strings.HasSuffix(settingsFile.Name(), ".tmpl") {
				continue
			}
			t.Templates = append(t.Templates, filepath.Join(cfg.Templates, settingsFile.Name()))
		}
	}

	t.funcMap = sprig.GenericFuncMap()
	t.funcMap["marshal"] = t.marshal
	t.funcMap["include"] = t.include

	return t
}

type TemplateParser struct {
	Vars      map[string]interface{}
	Partials  string
	Parser    config.Parser
	Templates []string
	Path      string
	err       parserError
	funcMap   template.FuncMap
}

func (t *TemplateParser) AddFunc(name string, f interface{}) {
	t.funcMap[name] = f
}

func (t *TemplateParser) Parse(configFile string) (config.ServiceConfig, error) {
	if len(t.err.errors) != 0 {
		return config.ServiceConfig{}, t.err
	}

	tmpfile, err := ioutil.TempFile("", "KrakenD_parsed_config_template_")
	if err != nil {
		log.Fatal("Couldn't create the temporary file:", err)
	}

	defer os.Remove(tmpfile.Name())

	var buf bytes.Buffer

	tmpl, err := template.New("config").Funcs(t.funcMap).ParseFiles(configFile)
	if err != nil {
		log.Fatal("Unable to parse configuration file:", err)
		return t.Parser.Parse(configFile)
	}
	if len(t.Templates) > 0 {
		tmpl, err = tmpl.ParseFiles(t.Templates...)
		if err != nil {
			log.Fatal("Error parsing sub-templates:", err)
			return t.Parser.Parse(configFile)
		}
	}
	err = tmpl.ExecuteTemplate(&buf, filepath.Base(configFile), t.Vars)
	if err != nil {
		log.Fatal("Found error while executing template:", err)
		return t.Parser.Parse(configFile)
	}

	if _, err = tmpfile.Write(buf.Bytes()); err != nil {
		log.Fatal("Unable to write the temporary configuration file:", err)
		return t.Parser.Parse(configFile)
	}
	if err = tmpfile.Close(); err != nil {
		log.Fatal("Unable to close the file after writing:", err)
	}

	filename := tmpfile.Name() + ".json"
	if t.Path != "" {
		filename = t.Path
	}
	if err := renameFile(tmpfile.Name(), filename); err != nil {
		return config.ServiceConfig{}, err
	}

	cfg, err := t.Parser.Parse(filename)

	if t.Path == "" {
		os.Remove(filename)
	}

	return cfg, err
}

func (t *TemplateParser) marshal(v interface{}) string {
	a, _ := json.Marshal(v)
	return string(a)
}

func (t *TemplateParser) include(v interface{}) string {
	a, _ := ioutil.ReadFile(path.Join(t.Partials, v.(string)))
	return string(a)
}

type parserError struct {
	errors map[string]error
}

func (p parserError) Error() string {
	msgs := make([]string, len(p.errors))
	var j int
	for i, e := range p.errors {
		msgs[j] = fmt.Sprintf("\t- %s: %s", i, e.Error())
		j++
	}
	return "loading flexible-config settings:\n" + strings.Join(msgs, "\n")
}

func renameFile(src string, dst string) (err error) {
	err = copyFile(src, dst)
	if err != nil {
		return fmt.Errorf("failed to copy source file %s to %s: %s", src, dst, err)
	}
	err = os.RemoveAll(src)
	if err != nil {
		return fmt.Errorf("failed to cleanup source file %s: %s", src, err)
	}
	return nil
}

// credit https://gist.github.com/r0l1/92462b38df26839a3ca324697c8cba04
func copyFile(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return
	}
	defer func() {
		if e := out.Close(); e != nil {
			err = e
		}
	}()

	_, err = io.Copy(out, in)
	if err != nil {
		return
	}

	err = out.Sync()
	if err != nil {
		return
	}

	si, err := os.Stat(src)
	if err != nil {
		return
	}
	err = os.Chmod(dst, si.Mode())
	if err != nil {
		return
	}

	return
}
