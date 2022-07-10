vec4 F0(in vec2 l0);

vec4 F0(in vec2 l0) {
	vec4 l1 = vec4(0);
	{
		vec3 l2 = vec3(0);
		(l1).x = (l0).x;
		{
			vec4 l3 = vec4(0);
			(l1).y = (l2).y;
			(l1).z = (l3).z;
		}
		{
			vec4 l3 = vec4(0);
			(l1).y = (l2).y;
			(l1).z = (l3).z;
		}
	}
	return l1;
}
