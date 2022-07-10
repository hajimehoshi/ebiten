vec4 F0(in vec4 l0);

vec4 F0(in vec4 l0) {
	vec4 l1 = vec4(0);
	l1 = (mat4(1.0)) * (l0);
	return l1;
}
