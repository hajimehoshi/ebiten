vec2 F0(void);
vec2 F1(void);

vec2 F0(void) {
	vec2 l0 = vec2(0);
	vec2 l1 = vec2(0);
	vec2 l2 = vec2(0);
	vec2 l3 = vec2(0);
	l0 = F1();
	l1 = (1.0) * (l0);
	l2 = F1();
	l3 = (l2) * (1.0);
	return l1;
}

vec2 F1(void) {
	return vec2(0.0);
}
