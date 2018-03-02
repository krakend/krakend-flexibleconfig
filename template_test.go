package flexibleconfig

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/devopsfaith/krakend/config"
)

func ExampleTemplateParser_marshal() {
	tmpfile, err := ioutil.TempFile("", "KrakenD_parsed_config_template_0_")
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	defer os.Remove(tmpfile.Name())
	if _, err := tmpfile.Write(originalTemplate); err != nil {
		fmt.Println(err.Error())
		return
	}
	if err := tmpfile.Close(); err != nil {
		fmt.Println(err.Error())
		return
	}

	expectedCfg := config.ServiceConfig{
		Port:    1234,
		Version: 42,
	}
	tmpl := TemplateParser{
		Vars: map[string]interface{}{
			"Namespace1": map[string]interface{}{
				"Namespace1-key1": "value1",
				"Namespace1-key2": 2,
			},
			"Namespace2": map[string]interface{}{
				"Namespace2-key1": "value1000",
				"Namespace2-key2": 2000,
			},
			"Jsonplaceholder": "http://example.com",
			"Port":            1234,
		},
		Parser: config.ParserFunc(func(tmpPath string) (config.ServiceConfig, error) {
			data, err := ioutil.ReadFile(tmpPath)
			fmt.Println(string(data))
			if err != nil {
				fmt.Println(err.Error())
				return expectedCfg, err
			}
			return expectedCfg, nil
		}),
	}
	res, err := tmpl.Parse(tmpfile.Name())
	if err != nil {
		fmt.Println(err.Error())
	}
	if res.Port != expectedCfg.Port {
		fmt.Println("unexpected cfg")
	}
	if res.Version != expectedCfg.Version {
		fmt.Println("unexpected cfg")
	}

	// Output:
	// {
	//     "version": 42,
	//     "port": 1234,
	//     "endpoints": [
	//         {
	//             "endpoint": "/combination/{id}",
	//             "backend": [
	//                 {
	//                     "host": [
	//                         "http://example.com"
	//                     ],
	//                     "url_pattern": "/posts?userId={id}",
	//                     "is_collection": true,
	//                     "mapping": {
	//                         "collection": "posts"
	//                     },
	//                     "disable_host_sanitize": true,
	//                     "extra_config": {
	//                     	"namespace1": {"Namespace1-key1":"value1","Namespace1-key2":2}
	// 				    }
	//                 },
	//                 {
	//                     "host": [
	//                         "http://example.com"
	//                     ],
	//                     "url_pattern": "/users/{id}",
	//                     "mapping": {
	//                         "email": "personal_email"
	//                     },
	//                     "disable_host_sanitize": true,
	//                     "extra_config": {
	//                     	"namespace1": {"Namespace1-key1":"value1","Namespace1-key2":2},
	//                     	"namespace2": {"Namespace2-key1":"value1000","Namespace2-key2":2000}
	// 				    }
	//                 }
	//             ],
	//             "extra_config": {
	//             	"namespace3": { "supu": "tupu" },
	//             	"namespace2": {"Namespace2-key1":"value1000","Namespace2-key2":2000}
	// 		    }
	//         }
	//     ],
	//     "extra_config": {
	//         "namespace2": {"Namespace2-key1":"value1000","Namespace2-key2":2000}
	//     }
	// }
}

func ExampleTemplateParser_include() {
	tmpfile, err := ioutil.TempFile("", "KrakenD_parsed_config_template_1_")
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	defer os.Remove(tmpfile.Name())
	if _, err := tmpfile.Write(originalTemplate); err != nil {
		fmt.Println(err.Error())
		return
	}
	if err := tmpfile.Close(); err != nil {
		fmt.Println(err.Error())
		return
	}

	includeTmpfile, err := ioutil.TempFile("", "KrakenD_parsed_config_template_2_")
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	defer os.Remove(includeTmpfile.Name())
	if _, err := includeTmpfile.Write([]byte(fmt.Sprintf("{{ include \"%s\" }}", tmpfile.Name()))); err != nil {
		fmt.Println(err.Error())
		return
	}
	if err := includeTmpfile.Close(); err != nil {
		fmt.Println(err.Error())
		return
	}

	expectedCfg := config.ServiceConfig{
		Port:    1234,
		Version: 42,
	}
	tmpl := TemplateParser{
		Vars: map[string]interface{}{},
		Parser: config.ParserFunc(func(tmpPath string) (config.ServiceConfig, error) {
			data, err := ioutil.ReadFile(tmpPath)
			fmt.Println(string(data))
			if err != nil {
				fmt.Println(err)
				return expectedCfg, err
			}
			return expectedCfg, nil
		}),
	}
	res, err := tmpl.Parse(includeTmpfile.Name())
	if err != nil {
		fmt.Println(err.Error())
	}
	if res.Port != expectedCfg.Port {
		fmt.Println("unexpected cfg")
	}
	if res.Version != expectedCfg.Version {
		fmt.Println("unexpected cfg")
	}

	// Output:
	// {
	//     "version": 42,
	//     "port": {{ .Port }},
	//     "endpoints": [
	//         {
	//             "endpoint": "/combination/{id}",
	//             "backend": [
	//                 {
	//                     "host": [
	//                         "{{ .Jsonplaceholder }}"
	//                     ],
	//                     "url_pattern": "/posts?userId={id}",
	//                     "is_collection": true,
	//                     "mapping": {
	//                         "collection": "posts"
	//                     },
	//                     "disable_host_sanitize": true,
	//                     "extra_config": {
	//                     	"namespace1": {{ marshal .Namespace1 }}
	// 				    }
	//                 },
	//                 {
	//                     "host": [
	//                         "{{ .Jsonplaceholder }}"
	//                     ],
	//                     "url_pattern": "/users/{id}",
	//                     "mapping": {
	//                         "email": "personal_email"
	//                     },
	//                     "disable_host_sanitize": true,
	//                     "extra_config": {
	//                     	"namespace1": {{ marshal .Namespace1 }},
	//                     	"namespace2": {{ marshal .Namespace2 }}
	// 				    }
	//                 }
	//             ],
	//             "extra_config": {
	//             	"namespace3": { "supu": "tupu" },
	//             	"namespace2": {{ marshal .Namespace2 }}
	// 		    }
	//         }
	//     ],
	//     "extra_config": {
	//         "namespace2": {{ marshal .Namespace2 }}
	//     }
	// }
}

var originalTemplate = []byte(`{
    "version": 42,
    "port": {{ .Port }},
    "endpoints": [
        {
            "endpoint": "/combination/{id}",
            "backend": [
                {
                    "host": [
                        "{{ .Jsonplaceholder }}"
                    ],
                    "url_pattern": "/posts?userId={id}",
                    "is_collection": true,
                    "mapping": {
                        "collection": "posts"
                    },
                    "disable_host_sanitize": true,
                    "extra_config": {
                    	"namespace1": {{ marshal .Namespace1 }}
				    }
                },
                {
                    "host": [
                        "{{ .Jsonplaceholder }}"
                    ],
                    "url_pattern": "/users/{id}",
                    "mapping": {
                        "email": "personal_email"
                    },
                    "disable_host_sanitize": true,
                    "extra_config": {
                    	"namespace1": {{ marshal .Namespace1 }},
                    	"namespace2": {{ marshal .Namespace2 }}
				    }
                }
            ],
            "extra_config": {
            	"namespace3": { "supu": "tupu" },
            	"namespace2": {{ marshal .Namespace2 }}
		    }
        }
    ],
    "extra_config": {
        "namespace2": {{ marshal .Namespace2 }}
    }
}`)
