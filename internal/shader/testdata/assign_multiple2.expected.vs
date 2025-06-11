vec2 F0(void);

vec2 F0(void) {
	float l0 = float(0);
	float l1 = float(0);
	float l2 = float(0);
	float l3 = float(0);
	l1 = 1.0;
	l3 = 2.0;
	l0 = l1;
	l2 = l3;
	return vec2(l0, l2);
}
