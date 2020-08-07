void F0(out vec2 l0);

void F0(out vec2 l0) {
	vec2 l1 = vec2(0);
	vec2 l3 = vec2(0);
	l1 = vec2(0.0);
	for (int l2 = 0; l2 < 100; l2++) {
		(l1).x = ((l1).x) + (l2);
	}
	l3 = vec2(0.0);
	for (float l4 = 10.0; l4 >= 0.0; l4 -= 2.0) {
		(l3).x = ((l3).x) + (l4);
	}
	l0 = l1;
	return;
}
