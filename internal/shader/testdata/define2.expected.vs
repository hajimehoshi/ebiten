vec2 F0(void);
vec2 F1(void);

vec2 F0(void) {
	vec2 l0 = vec2(0);
	vec2 l1 = vec2(0);
	l0 = (1.0) * (F1());
	l1 = (F1()) * (1.0);
	return l0;
}

vec2 F1(void) {
	return vec2(0.0);
}
