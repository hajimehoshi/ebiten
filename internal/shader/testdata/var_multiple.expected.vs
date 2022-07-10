vec4 F0(in vec2 l0);
vec4 F1(in vec2 l0);
void F2(out vec2 l0, out vec2 l1);

vec4 F0(in vec2 l0) {
	vec2 l1 = vec2(0);
	vec2 l2 = vec2(0);
	l1 = l0;
	l2 = l0;
	return vec4(l1, l2);
}

vec4 F1(in vec2 l0) {
	vec2 l1 = vec2(0);
	vec2 l2 = vec2(0);
	vec2 l3 = vec2(0);
	vec2 l4 = vec2(0);
	F2(l1, l2);
	l3 = l1;
	l4 = l2;
	return vec4(l3, l4);
}

void F2(out vec2 l0, out vec2 l1) {
	l0 = vec2(0.0);
	l1 = vec2(0.0);
	return;
}
