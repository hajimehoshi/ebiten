void F0(in vec4 l0, out vec4 l1);

void F0(in vec4 l0, out vec4 l1) {
	vec4 l2 = vec4(0);
	l2 = (mat4(1.0)) * (l0);
	l1 = l2;
	return;
}
