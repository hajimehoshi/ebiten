void F0(out vec2 l0);

void F0(out vec2 l0) {
	float l1 = float(0);
	float l2 = float(0);
	float l3 = float(0);
	float l4 = float(0);
	l2 = 1.0;
	l1 = l2;
	l4 = 2.0;
	l3 = l4;
	l0 = vec2(l1, l3);
	return;
}
