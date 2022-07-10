vec2 F0(in vec2 l0);
vec2 F1(in vec2 l0);

vec2 F0(in vec2 l0) {
	vec2 l1 = vec2(0);
	l1 = F1(l0);
	return l1;
}

vec2 F1(in vec2 l0) {
	return l0;
}
