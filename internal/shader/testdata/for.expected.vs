vec2 F0(void);

vec2 F0(void) {
	vec2 l0 = vec2(0);
	vec2 l2 = vec2(0);
	l0 = vec2(0.0);
	for (int l1 = 0; l1 < 100; l1++) {
		(l0).x = ((l0).x) + (float(l1));
	}
	l2 = vec2(0.0);
	for (float l3 = 10.0; l3 >= 0.0; l3 -= 2.0) {
		(l2).x = ((l2).x) + (float(l3));
	}
	return l0;
}
