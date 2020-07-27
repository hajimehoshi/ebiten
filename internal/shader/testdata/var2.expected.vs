void F0(in vec2 l0, out vec4 l1);

void F0(in vec2 l0, out vec4 l1) {
	vec4 l2 = vec4(0);
	vec4 l3 = vec4(0);
	l2 = vec4(l0, 0.0, 1.0);
	(l2).x = (l2).x;
	l3 = l2;
	l1 = l3;
	return;
}
