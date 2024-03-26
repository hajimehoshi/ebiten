{{.License}}

{{.DoNotEdit}}

{{.BuildTag}}

package gamepaddb

func init() {
	if err := Update(controllerBytes); err != nil {
		panic(err)
	}
{{if eq .SDLPlatform "Windows" }} 
    if err := Update(glfwControllerBytes); err != nil {
        panic(err)
    }
{{end}}}

var controllerBytes = []byte{ {{ range $index, $element := .ControllerBytes }}{{ if $index }},{{ end }}{{ $element }}{{ end }} }

{{if eq .SDLPlatform "Windows" }} 
var glfwControllerBytes = []byte{ {{ range $index, $element := .GLFWGamePads }}{{ if $index }},{{ end }}{{ $element }}{{ end }} }
{{end}}