void F0(in vec2 l0, out vec4 l1);
void F1(in vec2 l0, out vec4 l1);
void F2(out vec2 l0, out vec2 l1);

void F0(in vec2 l0, out vec4 l1) {
	vec2 l2 = vec2(0);
	vec2 l3 = vec2(0);
	l2 = l0;
	l3 = l0;
	l1 = vec4(l2, l3);
	return;
}

void F1(in vec2 l0, out vec4 l1) {
	vec2 l2 = vec2(0);
	vec2 l3 = vec2(0);
	vec2 l4 = vec2(0);
	vec2 l5 = vec2(0);
	F2(l2, l3);
	l4 = l2;
	l5 = l3;
	l1 = vec4(l4, l5);
	return;
}

void F2(out vec2 l0, out vec2 l1) {
	l0 = vec2(0.0);
	l1 = vec2(0.0);
	return;
}
