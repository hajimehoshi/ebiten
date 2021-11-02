package shaders

var (
	SinLayer0Shader = []byte(
		`
package main

var ScreenSize vec2
var Palette [4]vec4
var Frequency float
var Scale float
	
func palette(t float, a, b, c, d vec4) vec3 {
	var clr vec3

	t = abs(t)
	if t >= a.a && t <= b.a {
		clr = mix(a.rgb*a.rgb, b.rgb*b.rgb, vec3((t-a.a)/(b.a-a.a)))
	} else if t >= b.a && t <= c.a {
		clr = mix(b.rgb*b.rgb, c.rgb*c.rgb, vec3((t-b.a)/(c.a-b.a)))
	} else {
		clr = mix(c.rgb*c.rgb, d.rgb*d.rgb, vec3((t-c.a)/(d.a-c.a)))
	}
	clr = clr*clr*(3.0-2.0*clr)
	return sqrt(clr)
}

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	// Convert position in [-1, 1]
	p := position.xy / ScreenSize
	p -= 0.5
	p /= Scale

	c0 := palette(p.x, Palette[0], Palette[1], Palette[2], Palette[3])
	c1 := palette(p.y, Palette[0], Palette[1], Palette[2], Palette[3])
	t := sin(p.x*Frequency) + sin(p.y*Frequency)

	return vec4(mix(c0, c1, t), 1)
}
`)

	EightCircleBackgroundShader = []byte(
		`
package main

var ScreenSize vec2
var Positions [8]vec2
var Radiuses [8]float
var Aliasing float

var Palette0 [4]vec4
var Palette1 [4]vec4
var Palette2 [4]vec4
var Palette3 [4]vec4
var Palette4 [4]vec4
var Palette5 [4]vec4
var Palette6 [4]vec4
var Palette7 [4]vec4
	
func palette(t float, a, b, c, d vec4) vec3 {
	var clr vec3

	t = -t
	if t >= a.a && t <= b.a {
		clr = mix(a.rgb*a.rgb, b.rgb*b.rgb, vec3((t-a.a)/(b.a-a.a)))
	} else if t >= b.a && t <= c.a {
		clr = mix(b.rgb*b.rgb, c.rgb*c.rgb, vec3((t-b.a)/(c.a-b.a)))
	} else {
		clr = mix(c.rgb*c.rgb, d.rgb*d.rgb, vec3((t-c.a)/(d.a-c.a)))
	}
	clr = clr*clr*(3.0-2.0*clr)
	return sqrt(clr)
}

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	p := position.xy / ScreenSize.xy
	p -= 0.5

	// Drawing circles
	idx := -1.
	shortestDistance := 0.
	for i := 0; i < 8; i++ {
		r := Radiuses[i]
		pos := p - Positions[i]
		d := length(pos) - r
		if d < shortestDistance {
			idx = float(i)
			shortestDistance = d
		}
	}
	// We're not in a circle
	if idx < 0. {
		return vec4(0.)
	}

	// Resolving color
	var c vec4
	if idx == 0. {
		c = vec4(palette(shortestDistance, Palette0[0], Palette0[1], Palette0[2], Palette0[3]), 1.)
	} else if idx == 1. {
		c = vec4(palette(shortestDistance, Palette1[0], Palette1[1], Palette1[2], Palette1[3]), 1.)
	} else if idx == 2. {
		c = vec4(palette(shortestDistance, Palette2[0], Palette2[1], Palette2[2], Palette2[3]), 1.)
	} else if idx == 3. {
		c = vec4(palette(shortestDistance, Palette3[0], Palette3[1], Palette3[2], Palette3[3]), 1.)
	} else if idx == 4. {
		c = vec4(palette(shortestDistance, Palette4[0], Palette4[1], Palette4[2], Palette4[3]), 1.)
	} else if idx == 5. {
		c = vec4(palette(shortestDistance, Palette5[0], Palette5[1], Palette5[2], Palette5[3]), 1.)
	} else if idx == 6. {
		c = vec4(palette(shortestDistance, Palette6[0], Palette6[1], Palette6[2], Palette6[3]), 1.)
	} else if idx == 7. {
		c = vec4(palette(shortestDistance, Palette7[0], Palette7[1], Palette7[2], Palette7[3]), 1.)
	}

	return c*(1.-smoothstep(Aliasing, 0., -shortestDistance))
}
`)

	DancingXShader = []byte(
		`
package main

var ScreenSize vec2
var Width float
var Radius float
var Aliasing float
var Frequency float
var Palette [4]vec4

var Time float

func palette(t float, a, b, c, d vec4) vec3 {
	var clr vec3

	if t >= a.a && t <= b.a {
		clr = mix(a.rgb*a.rgb, b.rgb*b.rgb, vec3((t-a.a)/(b.a-a.a)))
	} else if t >= b.a && t <= c.a {
		clr = mix(b.rgb*b.rgb, c.rgb*c.rgb, vec3((t-b.a)/(c.a-b.a)))
	} else {
		clr = mix(c.rgb*c.rgb, d.rgb*d.rgb, vec3((t-c.a)/(d.a-c.a)))
	}
	clr = clr*clr*(3.0-2.0*clr)
	return sqrt(clr)
}

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	p := position.xy / ScreenSize.xy
	p -= 0.5
	p -= vec2(Time, sin(Time*Frequency)/3)
	p = abs(p)
    d := length(p-min(p.x+p.y,Width)*0.5) - Radius
	if d > 0. {
		return vec4(0)
	}

	c := vec4(palette(d, Palette[0], Palette[1], Palette[2], Palette[3]), 1)
	return c*(1-smoothstep(Aliasing, 0., -d))
}
`)

	RandomMaskShader = []byte(
		`
package main

// 25 useless uniforms of because desperate
var Maskvar0 float
var Maskvar1 float
var Maskvar2 float
var Maskvar3 float
var Maskvar4 float
var Maskvar5 float
var Maskvar6 float
var Maskvar7 float
var Maskvar8 float
var Maskvar9 float
var Maskvar10 float
var Maskvar11 float
var Maskvar12 float
var Maskvar13 float
var Maskvar14 float
var Maskvar15 float
var Maskvar16 float
var Maskvar17 float
var Maskvar18 float
var Maskvar19 float
var Maskvar20 float
var Maskvar21 float
var Maskvar22 float
var Maskvar23 float
var Maskvar24 float

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	mask := vec4(1, 0, 0, 1)
	mask *= (Maskvar0*0.01)
	mask *= (Maskvar1*0.01)
	mask *= (Maskvar2*0.01)
	mask *= (Maskvar3*0.01)
	mask *= (Maskvar4*0.01)	
	mask *= (Maskvar5*0.01)
	mask *= (Maskvar6*0.01)
	mask *= (Maskvar7*0.01)
	mask *= (Maskvar8*0.01)
	mask *= (Maskvar9*0.01)
	mask *= (Maskvar10*0.01)
	mask *= (Maskvar11*0.01)
	mask *= (Maskvar12*0.01)
	mask *= (Maskvar13*0.01)
	mask *= (Maskvar14*0.01)
	mask *= (Maskvar15*0.01)
	mask *= (Maskvar16*0.01)
	mask *= (Maskvar17*0.01)
	mask *= (Maskvar18*0.01)
	mask *= (Maskvar19*0.01)
	mask *= (Maskvar20*0.01)
	mask *= (Maskvar21*0.01)
	mask *= (Maskvar22*0.01)
	mask *= (Maskvar23*0.01)
	mask *= (Maskvar24*0.01)

	return imageSrc0UnsafeAt(texCoord)+mask
}
`)
)
