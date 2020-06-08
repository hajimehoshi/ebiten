package main

func Foo(foo vec2) vec4 {
	var r vec4
	{
		r.x = foo.x
		var foo vec3
		{
			r.y = foo.y
			var foo vec4
			r.z = foo.z
		}
		{
			r.y = foo.y
			var foo vec4
			r.z = foo.z
		}
	}
	return r
}
