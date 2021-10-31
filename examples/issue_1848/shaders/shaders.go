package shaders

var (
	ShaderPaletteHorizontalSrc = []byte(
		`
	package main

	var Palette0 [4]vec4
	
	func palette(t float, a, b, c, d vec4) vec3 {
		var clr vec3
	
		if t >= a.a && t <= b.a {
			clr = mix(a.rgb*a.rgb, b.rgb*b.rgb, vec3((t-a.a)/(b.a-a.a)))
		} else if t >= b.a && t <= c.a {
			clr = mix(b.rgb*b.rgb, c.rgb*c.rgb, vec3((t-b.a)/(c.a-b.a)))
		} else {
			clr = mix(c.rgb*c.rgb, d.rgb*d.rgb, vec3((t-c.a)/(d.a-c.a)))
		}
		clr = clr*clr*(3.0-2.0*clr) // cubic smoothing
		return sqrt(clr)
	}
	
	func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
		p := position.xy/imageDstTextureSize()
		t := abs(length(p*2-vec2(1., 0.5))-0.5)
		t = p.x

		clr := palette(t, Palette0[0], Palette0[1], Palette0[2], Palette0[3])
	
		return vec4(clr, 1.0)
	}
`)

	ShaderPaletteVerticalSrc = []byte(
		`
package main

var Palette1 [4]vec4

func palette(t float, a, b, c, d vec4) vec3 {
	var clr vec3

	if t >= a.a && t <= b.a {
		clr = mix(a.rgb*a.rgb, b.rgb*b.rgb, vec3((t-a.a)/(b.a-a.a)))
	} else if t >= b.a && t <= c.a {
		clr = mix(b.rgb*b.rgb, c.rgb*c.rgb, vec3((t-b.a)/(c.a-b.a)))
	} else {
		clr = mix(c.rgb*c.rgb, d.rgb*d.rgb, vec3((t-c.a)/(d.a-c.a)))
	}
	clr = clr*clr*(3.0-2.0*clr) // cubic smoothing
	return sqrt(clr)
}

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	p := position.xy/imageDstTextureSize()
	t := abs(length(p*2-vec2(1., 0.5))-0.5)
	t = p.y

	clr := palette(t, Palette1[0], Palette1[1], Palette1[2], Palette1[3])

	return vec4(clr, 1.0)
}
`)
)
