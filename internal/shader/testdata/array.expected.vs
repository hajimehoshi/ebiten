uniform vec2 U0[4];

vec2[2] F0(void);
vec2[2] F1(void);

vec2[2] F0(void) {
	vec2 l0[2];
	l0[0] = vec2(0);
	l0[1] = vec2(0);
	return l0;
}

vec2[2] F1(void) {
	vec2 l0[2];
	l0[0] = vec2(0);
	l0[1] = vec2(0);
	vec2 l1[2];
	l1[0] = vec2(0);
	l1[1] = vec2(0);
	(l0)[0] = vec2(1.0);
	l1[0] = l0[0];
	l1[1] = l0[1];
	(l1)[1] = vec2(2.0);
	return l1;
}
