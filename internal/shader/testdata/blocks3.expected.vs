void F0(in vec2 l0, out vec4 l1);

void F0(in vec2 l0, out vec4 l1) {
	vec4 l2 = vec4(0);
	{
		vec3 l3 = vec3(0);
		vec3 l4 = vec3(0);
		(l2).x = (l0).x;
		{
			vec4 l4 = vec4(0);
			(l2).y = (l3).y;
			(l2).z = (l4).z;
		}
		{
			vec4 l5 = vec4(0);
			(l2).y = (l3).y;
			(l2).z = (l4).z;
		}
	}
	l1 = l2;
	return;
}
