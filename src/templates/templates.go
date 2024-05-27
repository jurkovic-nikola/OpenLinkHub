package templates

import (
	"OpenLinkHub/src/config"
	"OpenLinkHub/src/logger"
	"OpenLinkHub/src/structs"
	"fmt"
	"html/template"
	"strings"
	"sync"
)

var (
	templates = make(map[string]template.Template)
	mutex     = sync.Mutex{}
)

type Web struct {
	Title      string
	Tpl        *template.Template
	Device     *structs.Device
	SystemInfo interface{}
}

func Init() {
	mutex.Lock()
	defer mutex.Unlock()

	templateList := strings.Split(config.GetConfig().TemplateList, ",")
	for i := range templateList {
		values := strings.Split(templateList[i], ".")
		filename := fmt.Sprintf("web/%s", templateList[i])

		tpl, err := template.ParseFiles(filename)
		if err != nil {
			logger.Log(logger.Fields{"error": err, "template": filename}).Error("Failed to load template")
			continue
		}
		templates[values[0]] = *tpl
	}
}

func GetTemplate(name string) *template.Template {
	mutex.Lock()
	defer mutex.Unlock()
	val, ok := templates[name]
	if ok {
		return &val
	}
	return nil
}
